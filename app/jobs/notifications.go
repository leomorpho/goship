package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/notifications"
	"github.com/leomorpho/goship/app/profiles"
	"github.com/leomorpho/goship/app/subscriptions"
	"github.com/leomorpho/goship/app/web/routenames"
	"github.com/leomorpho/goship/db/ent"
	"github.com/leomorpho/goship/db/ent/notification"
	"github.com/leomorpho/goship/framework/core"
	"github.com/leomorpho/goship/framework/domain"
	"github.com/rs/zerolog/log"
)

var dailyNotifText = []string{
	"A good relationship is when someone accepts your past, supports your present, and texts you back quickly.",
	"Strong relationships are built on trust, honesty, and the ability to laugh at each other’s jokes.",
	"The most important thing in communication is hearing what isn’t said – and pretending you understood.",
}

// ----------------------------------------------------------

const TypeAllDailyConvoNotifications = "notification.all_daily_conversation"

type (
	plannedNotificationSource interface {
		CreateNotificationTimeObjects(ctx context.Context, notifType domain.NotificationType, permissionType domain.NotificationPermissionType) error
		ProfileIDsCanGetPlannedNotificationNow(ctx context.Context, now time.Time, notifType domain.NotificationType, scopeProfileIDs *[]int) ([]int, error)
	}

	AllDailyConvoNotificationsProcessor struct {
		orm                     *ent.Client
		profileRepo             *profiles.ProfileRepo
		taskRunner              core.Jobs
		timespanInMinutes       int
		plannedNotificationRepo plannedNotificationSource
	}
)

func NewAllDailyConvoNotificationsProcessor(
	orm *ent.Client,
	profileRepo *profiles.ProfileRepo,
	plannedNotificationRepo plannedNotificationSource,
	taskRunner core.Jobs,
	timespanInMinutes int,
) *AllDailyConvoNotificationsProcessor {
	return &AllDailyConvoNotificationsProcessor{
		orm:                     orm,
		profileRepo:             profileRepo,
		plannedNotificationRepo: plannedNotificationRepo,
		taskRunner:              taskRunner,
		timespanInMinutes:       timespanInMinutes,
	}
}

func (d *AllDailyConvoNotificationsProcessor) ProcessTask(
	ctx context.Context, t *asynq.Task,
) error {
	err := d.plannedNotificationRepo.CreateNotificationTimeObjects(
		ctx, domain.NotificationTypeDailyConversationReminder, domain.NotificationPermissionDailyReminder)
	if err != nil {
		return err
	}

	profileIDs, err := d.plannedNotificationRepo.ProfileIDsCanGetPlannedNotificationNow(
		ctx, time.Now(), domain.NotificationTypeDailyConversationReminder, nil)
	if err != nil {
		return err
	}

	// Start tasks in batches to notify users who should received the
	// daily notification for the current hour.
	batchSize := 50

	for i := 0; i < len(profileIDs); i += batchSize {
		end := i + batchSize
		if end > len(profileIDs) {
			end = len(profileIDs)
		}
		batch := profileIDs[i:end]

		payload, err := json.Marshal(DailyConvoNotificationsPayload{ProfileIDs: batch})
		if err != nil {
			log.Error().Err(err).Msg("failed to marshal TypeDailyConvoNotification payload")
			continue
		}
		if _, err := d.taskRunner.Enqueue(ctx, TypeDailyConvoNotification, payload, core.EnqueueOptions{
			Timeout:   120 * time.Second,
			Retention: 24 * time.Hour,
		}); err != nil {
			log.Error().Err(err).
				Msg("failed to start TypeDailyConvoNotification task")
		}
	}

	return nil
}

// ----------------------------------------------------------

const TypeDailyConvoNotification = "notification.subset_daily_conversation"

type (
	DailyConvoNotificationsProcessor struct {
		orm                     *ent.Client
		notifierRepo            *notifications.NotifierRepo
		echoServer              *echo.Echo
		subscriptionRepo        *subscriptions.SubscriptionsRepo
		plannedNotificationRepo plannedNotificationSource
	}

	DailyConvoNotificationsPayload struct {
		ProfileIDs []int
	}
)

func NewDailyConvoNotificationsProcessor(
	notifierRepo *notifications.NotifierRepo,
	e *echo.Echo,
	subscriptionRepo *subscriptions.SubscriptionsRepo,
	plannedNotificationRepo plannedNotificationSource,
) *DailyConvoNotificationsProcessor {

	return &DailyConvoNotificationsProcessor{
		notifierRepo:            notifierRepo,
		echoServer:              e,
		subscriptionRepo:        subscriptionRepo,
		plannedNotificationRepo: plannedNotificationRepo,
	}
}
func (d *DailyConvoNotificationsProcessor) ProcessTask(
	ctx context.Context, t *asynq.Task,
) error {

	var p DailyConvoNotificationsPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		fmt.Printf("Error unmarshalling payload: %v\n", err)
		return err
	}

	wantedProfileIDs, err := d.plannedNotificationRepo.ProfileIDsCanGetPlannedNotificationNow(
		ctx, time.Now(), domain.NotificationTypeDailyConversationReminder, &p.ProfileIDs,
	)
	if err != nil {
		return err
	}

	for _, profileID := range wantedProfileIDs {

		// Generate a random index
		randomIndex := rand.Intn(len(dailyNotifText))
		// Select a random item from the list
		randomDailyNotifText := dailyNotifText[randomIndex]

		prod, _, _, err := d.subscriptionRepo.GetCurrentlyActiveProduct(ctx, profileID)
		if err != nil {
			log.Error().
				Err(err).
				Int("profileID", profileID).
				Msg("failed to get currently active plan")
			return err
		}
		var title string

		if prod == &domain.ProductTypeFree {
			title = "🌤 Today's free question!"
		} else {
			title = "🌤 Today's question!"
		}

		url := d.echoServer.Reverse(routenames.RouteNameHomeFeed)
		err = d.notifierRepo.PublishNotification(ctx, domain.Notification{
			Type:                      domain.NotificationTypeDailyConversationReminder,
			ProfileID:                 profileID,
			Title:                     title,
			Text:                      randomDailyNotifText,
			ReadInNotificationsCenter: true,
			Link:                      url,
		}, true, true)
		if err != nil {
			log.Error().
				Err(err).
				Int("profileID", profileID).
				Str("type", domain.NotificationTypeDailyConversationReminder.Value).
				Msg("failed to send notification")
		}
	}

	return nil
}

// ----------------------------------------------------------
const TypeDeleteStaleNotifications = "notification.recycling"

type (
	DeleteStaleNotificationsProcessor struct {
		orm     *ent.Client
		numDays int
	}
)

func NewDeleteStaleNotificationsProcessor(orm *ent.Client, numDays int) *DeleteStaleNotificationsProcessor {
	return &DeleteStaleNotificationsProcessor{
		orm:     orm,
		numDays: numDays,
	}
}
func (d *DeleteStaleNotificationsProcessor) ProcessTask(
	ctx context.Context, t *asynq.Task,
) error {

	_, err := d.orm.Notification.
		Delete().
		Where(
			notification.CreatedAtLT(time.Now().Add(time.Hour * -24 * time.Duration(d.numDays))),
		).
		Exec(ctx)

	if err != nil {
		return err
	}

	// Delete all daily notifications that are older than 48h
	_, err = d.orm.Notification.
		Delete().
		Where(
			notification.CreatedAtLT(time.Now().Add(time.Hour*-48)),
			notification.TypeIn(
				notification.Type(domain.NotificationTypeDailyConversationReminder.Value),
			),
		).
		Exec(ctx)

	if err != nil {
		return err
	}

	return nil
}
