package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
	"github.com/mikestefanello/pagoda/ent"
	"github.com/mikestefanello/pagoda/ent/notification"
	"github.com/mikestefanello/pagoda/pkg/domain"
	"github.com/mikestefanello/pagoda/pkg/repos/notifierrepo"
	"github.com/mikestefanello/pagoda/pkg/repos/profilerepo"
	"github.com/mikestefanello/pagoda/pkg/repos/subscriptions"
	"github.com/mikestefanello/pagoda/pkg/routing/routenames"
	"github.com/mikestefanello/pagoda/pkg/services"
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
	AllDailyConvoNotificationsProcessor struct {
		orm                     *ent.Client
		profileRepo             *profilerepo.ProfileRepo
		taskRunner              *services.TaskClient
		timespanInMinutes       int
		plannedNotificationRepo *notifierrepo.PlannedNotificationsRepo
	}
)

func NewAllDailyConvoNotificationsProcessor(
	orm *ent.Client,
	profileRepo *profilerepo.ProfileRepo,
	plannedNotificationRepo *notifierrepo.PlannedNotificationsRepo,
	taskRunner *services.TaskClient,
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

		if err := d.taskRunner.
			New(TypeDailyConvoNotification).
			Payload(DailyConvoNotificationsPayload{ProfileIDs: batch}).
			Timeout(120 * time.Second).
			Retain(24 * time.Hour).
			Save(); err != nil {
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
		notifierRepo            *notifierrepo.NotifierRepo
		echoServer              *echo.Echo
		subscriptionRepo        *subscriptions.SubscriptionsRepo
		plannedNotificationRepo *notifierrepo.PlannedNotificationsRepo
	}

	DailyConvoNotificationsPayload struct {
		ProfileIDs []int
	}
)

func NewDailyConvoNotificationsProcessor(
	notifierRepo *notifierrepo.NotifierRepo,
	e *echo.Echo,
	subscriptionRepo *subscriptions.SubscriptionsRepo,
	plannedNotificationRepo *notifierrepo.PlannedNotificationsRepo,
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
