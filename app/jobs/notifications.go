package tasks

import (
	"context"
	"database/sql"
	"encoding/json"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship-modules/notifications"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	"github.com/leomorpho/goship/app/web/routenames"
	dbqueries "github.com/leomorpho/goship/db/queries"
	"github.com/leomorpho/goship/framework/core"
	"github.com/leomorpho/goship/framework/domain"
	"log/slog"
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
		taskRunner              core.Jobs
		timespanInMinutes       int
		plannedNotificationRepo plannedNotificationSource
	}
)

func NewAllDailyConvoNotificationsProcessor(
	plannedNotificationRepo plannedNotificationSource,
	taskRunner core.Jobs,
	timespanInMinutes int,
) *AllDailyConvoNotificationsProcessor {
	return &AllDailyConvoNotificationsProcessor{
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
			slog.Error("failed to marshal TypeDailyConvoNotification payload", "error", err)
			continue
		}
		if _, err := d.taskRunner.Enqueue(ctx, TypeDailyConvoNotification, payload, core.EnqueueOptions{
			Timeout:   120 * time.Second,
			Retention: 24 * time.Hour,
		}); err != nil {
			slog.Error("failed to start TypeDailyConvoNotification task", "error", err)
		}
	}

	return nil
}

// ----------------------------------------------------------

const TypeDailyConvoNotification = "notification.subset_daily_conversation"

type (
	DailyConvoNotificationsProcessor struct {
		notifierService         *notifications.NotifierService
		echoServer              *echo.Echo
		subscriptionRepo        *paidsubscriptions.Service
		plannedNotificationRepo plannedNotificationSource
	}

	DailyConvoNotificationsPayload struct {
		ProfileIDs []int
	}
)

func NewDailyConvoNotificationsProcessor(
	notifierService *notifications.NotifierService,
	e *echo.Echo,
	subscriptionRepo *paidsubscriptions.Service,
	plannedNotificationRepo plannedNotificationSource,
) *DailyConvoNotificationsProcessor {

	return &DailyConvoNotificationsProcessor{
		notifierService:         notifierService,
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
		slog.Error("Error unmarshalling payload", "error", err)
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
			slog.Error("failed to get currently active plan", "error", err, "profileID", profileID)
			return err
		}
		var title string

		if !d.subscriptionRepo.IsPaidProduct(prod) {
			title = "🌤 Today's free question!"
		} else {
			title = "🌤 Today's question!"
		}

		url := d.echoServer.Reverse(routenames.RouteNameHomeFeed)
		err = d.notifierService.PublishNotification(ctx, domain.Notification{
			Type:                      domain.NotificationTypeDailyConversationReminder,
			ProfileID:                 profileID,
			Title:                     title,
			Text:                      randomDailyNotifText,
			ReadInNotificationsCenter: true,
			Link:                      url,
		}, true, true)
		if err != nil {
			slog.Error("failed to send notification",
				"error", err,
				"profileID", profileID,
				"type", domain.NotificationTypeDailyConversationReminder.Value)
		}
	}

	return nil
}

// ----------------------------------------------------------
const TypeDeleteStaleNotifications = "notification.recycling"

type (
	DeleteStaleNotificationsProcessor struct {
		db         *sql.DB
		postgresql bool
		numDays    int
	}
)

func NewDeleteStaleNotificationsProcessor(db *sql.DB, dialect string, numDays int) *DeleteStaleNotificationsProcessor {
	d := strings.ToLower(strings.TrimSpace(dialect))
	return &DeleteStaleNotificationsProcessor{
		db:         db,
		postgresql: d == "postgres" || d == "postgresql" || d == "pgx",
		numDays:    numDays,
	}
}
func (d *DeleteStaleNotificationsProcessor) ProcessTask(
	ctx context.Context, t *asynq.Task,
) error {
	deleteBeforeQuery, err := dbqueries.Get("delete_notifications_before")
	if err != nil {
		return err
	}
	_, err = d.db.ExecContext(ctx, d.bind(deleteBeforeQuery), time.Now().Add(time.Hour*-24*time.Duration(d.numDays)))

	if err != nil {
		return err
	}

	// Delete all daily notifications that are older than 48h
	deleteDailyBeforeQuery, lookupErr := dbqueries.Get("delete_daily_notifications_before")
	if lookupErr != nil {
		return lookupErr
	}
	_, err = d.db.ExecContext(ctx, d.bind(deleteDailyBeforeQuery), time.Now().Add(time.Hour*-48), domain.NotificationTypeDailyConversationReminder.Value)

	if err != nil {
		return err
	}

	return nil
}

func (d *DeleteStaleNotificationsProcessor) bind(query string) string {
	if !d.postgresql || strings.Count(query, "?") == 0 {
		return query
	}
	var b strings.Builder
	b.Grow(len(query) + 8)
	arg := 1
	for _, r := range query {
		if r == '?' {
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(arg))
			arg++
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
