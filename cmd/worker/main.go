package main

import (
	"fmt"
	"log"

	"github.com/hibiken/asynq"
	"github.com/mikestefanello/pagoda/config"
	"github.com/mikestefanello/pagoda/pkg/repos/notifierrepo"
	"github.com/mikestefanello/pagoda/pkg/repos/profilerepo"
	storagerepo "github.com/mikestefanello/pagoda/pkg/repos/storage"
	"github.com/mikestefanello/pagoda/pkg/repos/subscriptions"
	"github.com/mikestefanello/pagoda/pkg/routing/routes"
	"github.com/mikestefanello/pagoda/pkg/services"
	"github.com/mikestefanello/pagoda/pkg/tasks"
)

func main() {
	// Load the configuration
	cfg, err := config.GetConfig()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	// Build the worker server
	srv := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     fmt.Sprintf("%s:%d", cfg.Cache.Hostname, cfg.Cache.Port),
			DB:       cfg.Cache.Database,
			Password: cfg.Cache.Password,
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

	// Start a new container
	c := services.NewContainer()
	defer func() {
		if err := c.Shutdown(); err != nil {
			c.Web.Logger.Fatal(err)
		}
	}()

	// Build the router, which is needed to get the reverse of routes by name in some tasks.
	routes.BuildRouter(c)

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
	allDailyConvoNotificationsProcessor := tasks.NewAllDailyConvoNotificationsProcessor(c.ORM, profileRepo, plannedNotificationRepo, c.Tasks, 30)
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
