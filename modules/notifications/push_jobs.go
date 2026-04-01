package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
)

const DeliverPushNotificationJobName = "notifications.deliver_push"

type DeliverPushNotificationPayload struct {
	ProfileID   int    `json:"profile_id"`
	Platform    string `json:"platform"`
	Title       string `json:"title"`
	Message     string `json:"message"`
	UnreadCount int    `json:"unread_count"`
	SendSound   bool   `json:"send_sound"`
}

func (s *NotifierService) enqueuePushNotification(ctx context.Context, payload DeliverPushNotificationPayload) error {
	if s.jobs == nil {
		slog.Warn("push delivery job skipped because jobs runtime is unavailable", "profileID", payload.ProfileID, "platform", payload.Platform)
		return nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = s.jobs.Enqueue(ctx, DeliverPushNotificationJobName, raw, EnqueueOptions{
		Queue:      "default",
		MaxRetries: 3,
	})
	return err
}

func (s *NotifierService) HandleDeliverPushNotificationJob(ctx context.Context, payload []byte) error {
	var in DeliverPushNotificationPayload
	if err := json.Unmarshal(payload, &in); err != nil {
		return fmt.Errorf("decode %s payload: %w", DeliverPushNotificationJobName, err)
	}
	platform := ParsePlatform(in.Platform)
	if platform == nil {
		return fmt.Errorf("unknown push platform %q", in.Platform)
	}
	canSend, err := s.canSendPushForPlatform(ctx, in.ProfileID, *platform)
	if err != nil {
		return err
	}
	if !canSend {
		return nil
	}

	switch *platform {
	case PlatformPWAPush:
		if s.pwaPushService == nil {
			return nil
		}
		if err := s.pwaPushService.SendPushNotifications(ctx, in.ProfileID, in.Title, in.Message, in.UnreadCount); err != nil {
			return err
		}
	case PlatformFCMPush:
		if s.fcmPushService == nil {
			return nil
		}
		if err := s.fcmPushService.SendPushNotifications(ctx, in.ProfileID, in.Title, in.Message, in.UnreadCount, in.SendSound); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported push platform %q", in.Platform)
	}

	slog.Debug("delivered push notification from async job", "profileID", in.ProfileID, "platform", in.Platform)
	return nil
}
