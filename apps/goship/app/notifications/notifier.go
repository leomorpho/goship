package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/leomorpho/goship/pkg/core"
	"github.com/rs/zerolog/log"

	"github.com/leomorpho/goship/pkg/domain"
)

/*
NotifierRepo manages the full lifecycle of notifications. That includes:
- Storage in DB.
- Publishing to event stream (pubsub).
- Create push notifications (TODO) for mobile apps.
*/
type NotifierRepo struct {
	pubSubClient             core.PubSub
	notificationStorageRepo  NotificationStorage
	pwaPushNotificationsRepo *PwaPushNotificationsRepo
	fcmPushNotificationsRepo *FcmPushNotificationsRepo
	getNumNotifsCount        func(context.Context, int) (int, error)
}

// SSEEvent is the notifier-level realtime event payload exposed to callers.
type SSEEvent struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

func NewNotifierRepo(
	pubSubClient core.PubSub,
	notificationStorageRepo NotificationStorage,
	pwaPushNotificationsRepo *PwaPushNotificationsRepo,
	fcmPushNotificationsRepo *FcmPushNotificationsRepo,
	getNumNotifsCount func(context.Context, int) (int, error),
) *NotifierRepo {
	return &NotifierRepo{
		pubSubClient:             pubSubClient,
		notificationStorageRepo:  notificationStorageRepo,
		pwaPushNotificationsRepo: pwaPushNotificationsRepo,
		fcmPushNotificationsRepo: fcmPushNotificationsRepo,
		getNumNotifsCount:        getNumNotifsCount,
	}
}

// CreateNotification creates and stores a notification, then publishes it
func (s *NotifierRepo) PublishNotification(
	ctx context.Context, notification domain.Notification, storeInDB bool, sendPushNotif bool,
) error {

	// TODO: we may NOT want to store the entire notif in the DB, especially if it can be derived live. We may want custom marshaller to store
	// only non-repeating data relevant to each notification. Notifications could then be built live with templates. The downside is this would
	// increase complexity quite a bit. As long as we're fine deleting notifications after a few weeks, it should be fine to store them in DB.
	if storeInDB {
		log.Debug().
			Int("profileID", notification.ProfileID).
			Int("profileIDWhoCausedNotif", notification.ProfileIDWhoCausedNotif).
			Str("notificationType", notification.Type.Value).
			Msg("creating persistent notification")
		_, err := s.notificationStorageRepo.CreateNotification(ctx, notification)
		if err != nil {
			return err
		}
	}

	// Send a notif to update the notification count. This is handled internally to keep
	// good ergonomics. The client can listen to this notification type to call APIs, for example,
	// which will then update the notification counts.
	// TODO: if we re-use the messaging notifications, we'll need to defined the Type of this notif
	// accordingly. For example we should use NotificationTypeIncrementNumUnseenMessages and NotificationTypeDecrementNumUnseenMessages
	// for private messages, but NotificationTypeUpdateNumNotifications for general notifications.
	err := s.publishEvent(ctx, fmt.Sprint(notification.ProfileID), SSEEvent{
		Type: domain.NotificationTypeUpdateNumNotifications.Value,
		Data: "n/a",
	})
	if err != nil {
		return err
	}

	// Send push notification
	if sendPushNotif {
		numNotifs, err := s.getNumNotifsCount(ctx, notification.ProfileID)
		if err != nil {
			log.Error().Err(err).Int("profileID", notification.ProfileID).Msg("failed to get number of notifications for profile")
			return err
		}

		if s.pwaPushNotificationsRepo != nil {
			err = s.pwaPushNotificationsRepo.SendPushNotifications(ctx, notification.ProfileID, notification.Title, notification.Text, numNotifs)
			if err != nil {
				return err
			}
			log.Debug().
				Int("profileID", notification.ProfileID).
				Int("profileIDWhoCausedNotif", notification.ProfileIDWhoCausedNotif).
				Str("notificationType", notification.Type.Value).
				Msg("sent pwa push notifications")
		}
		if s.fcmPushNotificationsRepo != nil {
			err = s.fcmPushNotificationsRepo.SendPushNotifications(ctx, notification.ProfileID, notification.Title, notification.Text, numNotifs, true)
			if err != nil {
				return err
			}
			log.Debug().
				Int("profileID", notification.ProfileID).
				Int("profileIDWhoCausedNotif", notification.ProfileIDWhoCausedNotif).
				Str("notificationType", notification.Type.Value).
				Msg("sent fcm push notifications")
		}

	}

	// Publish the notification to the user-specific topic
	return s.publishEvent(ctx, fmt.Sprint(notification.ProfileID), SSEEvent{
		Type: notification.Type.Value,
		Data: notification.Text,
	})
}

// SendSSEUpdate sends an SSE HTML blob update to a profile
func (s *NotifierRepo) SendSSEUpdate(
	ctx context.Context, notification domain.Notification,
) error {
	// Publish the notification to the user-specific topic
	return s.publishEvent(ctx, fmt.Sprint(notification.ProfileID), SSEEvent{
		Type: notification.Type.Value,
		Data: notification.Text,
	})
}

func (s *NotifierRepo) HasNotificationForResourceAndPerson(
	ctx context.Context, notifType domain.NotificationType, profileIDWhoCausedNotif, resourceID *int, maxAge time.Duration,
) (exists bool, err error) {
	return s.notificationStorageRepo.HasNotificationForResourceAndPerson(
		ctx, notifType, profileIDWhoCausedNotif, resourceID, maxAge)
}

// GetNotifications retrieves notifications for a user
func (s *NotifierRepo) GetNotifications(
	ctx context.Context, profileID int, onlyUnread bool, beforeTimestamp *time.Time, pageSize *int,
) ([]*domain.Notification, error) {
	notifications, err := s.notificationStorageRepo.GetNotificationsByProfileID(
		ctx, profileID, onlyUnread, beforeTimestamp, pageSize,
	)
	if err != nil {
		return nil, err
	}
	return notifications, nil
}

// MarkNotificationRead marks a specific notification as read
func (s *NotifierRepo) MarkNotificationRead(
	ctx context.Context, notificationID int, profileID *int,
) error {
	err := s.notificationStorageRepo.MarkNotificationAsRead(ctx, notificationID, profileID)
	if err != nil {
		return err
	}

	// Update notification counts
	if profileID != nil {
		err = s.emitNumNotificationUpdateEvent(ctx, *profileID)
		if err != nil {
			return err
		}
		return s.resetIosFCMNotificationsBadge(ctx, *profileID)
	}

	return nil
}

// MarkNotificationRead marks a specific notification as read
func (s *NotifierRepo) MarkAllNotificationRead(
	ctx context.Context, profileID int,
) error {
	err := s.notificationStorageRepo.MarkAllNotificationAsRead(ctx, profileID)
	if err != nil {
		return err
	}

	return s.resetIosFCMNotificationsBadge(ctx, profileID)
}

func (s *NotifierRepo) resetIosFCMNotificationsBadge(ctx context.Context, profileID int) error {
	if s.fcmPushNotificationsRepo != nil {
		numNotifs := 0
		err := s.fcmPushNotificationsRepo.SendPushNotifications(ctx, profileID, "", "", numNotifs, false)
		if err != nil {
			return err
		}
		log.Debug().
			Int("profileID", profileID).Int("countNotifs", numNotifs).
			Msg("sent fcm push notifications to reset ios app badge count")
	}
	return nil
}

// MarkNotificationRead marks a specific notification as read
func (s *NotifierRepo) MarkNotificationUnread(
	ctx context.Context, notificationID int, profileID *int,
) error {
	err := s.notificationStorageRepo.MarkNotificationAsUnread(ctx, notificationID, profileID)
	if err != nil {
		return err
	}

	// Update notification counts
	if profileID != nil {
		return s.emitNumNotificationUpdateEvent(ctx, *profileID)
	}
	return nil
}

func (s *NotifierRepo) emitNumNotificationUpdateEvent(ctx context.Context, profileID int) error {
	// Update notification counts
	return s.PublishNotification(
		ctx,
		domain.Notification{
			Type:      domain.NotificationTypeUpdateNumNotifications,
			ProfileID: profileID,
			Text:      "n/a",
		}, false, false,
	)
}

// DeleteNotification deletes a notification by its ID.
func (s *NotifierRepo) DeleteNotification(ctx context.Context, notificationID int, profileID *int) error {
	err := s.notificationStorageRepo.DeleteNotification(ctx, notificationID, profileID)
	if err != nil {
		return err
	}

	return nil
}

// SSESubscribe to a topic to get live notifications from it.
func (s *NotifierRepo) SSESubscribe(
	ctx context.Context, topic string,
) (<-chan SSEEvent, error) {
	subCtx, cancel := context.WithCancel(ctx)
	out := make(chan SSEEvent)
	sub, err := s.pubSubClient.Subscribe(subCtx, topic, func(hctx context.Context, _ string, payload []byte) error {
		var event SSEEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			return err
		}
		select {
		case out <- event:
			return nil
		case <-hctx.Done():
			return hctx.Err()
		}
	})
	if err != nil {
		cancel()
		close(out)
		return nil, err
	}

	go func() {
		<-subCtx.Done()
		cancel()
		_ = sub.Close()
		close(out)
	}()

	return out, nil
}

func (s *NotifierRepo) publishEvent(ctx context.Context, topic string, event SSEEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return s.pubSubClient.Publish(ctx, topic, payload)
}
