package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/hibiken/asynq"
	"github.com/leomorpho/goship/app/goship"
	"github.com/leomorpho/goship/pkg/repos/notifierrepo"
	"github.com/leomorpho/goship/pkg/repos/profilerepo"
	storagerepo "github.com/leomorpho/goship/pkg/repos/storage"
	"github.com/leomorpho/goship/pkg/repos/subscriptions"
	"github.com/leomorpho/goship/pkg/services"
	"github.com/leomorpho/goship/pkg/tasks"
)

func main() {
	// Start a new container
	c := services.NewContainer()
	defer func() {
		if err := c.Shutdown(); err != nil {
			c.Web.Logger.Fatal(err)
		}
	}()
	if err := validateWorkerConfig(*c.Config); err != nil {
		log.Fatalf("invalid worker runtime configuration: %v", err)
	}

	// Build the worker server
	cacheHost := strings.TrimSpace(c.Config.Cache.Hostname)
	if cacheHost == "" || strings.EqualFold(cacheHost, "FILL") {
		log.Printf("cache hostname is unset/placeholder (%q); defaulting to localhost for local worker", c.Config.Cache.Hostname)
		cacheHost = "localhost"
	}

	cachePort := c.Config.Cache.Port
	if cachePort == 0 {
		cachePort = 6379
	}

	srv := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     fmt.Sprintf("%s:%d", cacheHost, cachePort),
			DB:       c.Config.Cache.Database,
			Password: c.Config.Cache.Password,
		},
		asynq.Config{
			// See asynq.Config for all available options and explanation
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)

	// Build the router, which is needed to get the reverse of routes by name in some tasks.
	if err := goship.BuildRouter(c); err != nil {
		c.Web.Logger.Fatalf("failed to build router: %v", err)
	}

	storageRepo := storagerepo.NewStorageClient(c.Config, c.ORM)
	subscriptionsRepo := subscriptions.NewSubscriptionsRepo(
		c.ORM, c.Config.App.OperationalConstants.ProTrialTimespanInDays,
		c.Config.App.OperationalConstants.PaymentFailedGracePeriodInDays)
	profileRepo := profilerepo.NewProfileRepo(c.ORM, storageRepo, subscriptionsRepo)

	plannedNotificationRepo := notifierrepo.NewPlannedNotificationsRepo(
		c.ORM, subscriptionsRepo)

	emailSubscriptionConfirmationProcessor := tasks.NewEmailSubscriptionConfirmationProcessor(
		c.Mail, c.Config,
	)

	emailUpdateProcessor := tasks.NewEmailUpdateProcessor(c, c.ORM)

	deactivateExpiredSubscriptionsProcessor := tasks.NewDeactivateExpiredSubscriptionsProcessor(subscriptionsRepo)
	allDailyConvoNotificationsProcessor := tasks.NewAllDailyConvoNotificationsProcessor(c.ORM, profileRepo, plannedNotificationRepo, c.CoreJobs, 30)
	dailyConvoNotificationsProcessor := tasks.NewDailyConvoNotificationsProcessor(c.Notifier, c.Web, subscriptionsRepo, plannedNotificationRepo)
	deleteStaleNotificationsProcessor := tasks.NewDeleteStaleNotificationsProcessor(
		c.ORM, c.Config.App.OperationalConstants.DeleteStaleNotificationAfterDays,
	)

	// Map task types to the handlers
	mux := asynq.NewServeMux()
	mux.Handle(tasks.TypeEmailSubscriptionConfirmation, emailSubscriptionConfirmationProcessor)
	mux.Handle(tasks.TypeEmailUpdates, emailUpdateProcessor)
	mux.Handle(tasks.TypeDeactivateExpiredSubscriptions, deactivateExpiredSubscriptionsProcessor)
	mux.Handle(tasks.TypeAllDailyConvoNotifications, allDailyConvoNotificationsProcessor)
	mux.Handle(tasks.TypeDailyConvoNotification, dailyConvoNotificationsProcessor)
	mux.Handle(tasks.TypeDeleteStaleNotifications, deleteStaleNotificationsProcessor)

	// Start the worker server
	if err := srv.Run(mux); err != nil {
		log.Fatalf("could not run worker server: %v", err)
	}
}
