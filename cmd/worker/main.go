package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/hibiken/asynq"
	"github.com/leomorpho/goship-modules/notifications"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	"github.com/leomorpho/goship/app"
	"github.com/leomorpho/goship/app/foundation"
	"github.com/leomorpho/goship/app/jobs"
	profilesvc "github.com/leomorpho/goship/app/profile"
	storagerepo "github.com/leomorpho/goship/framework/repos/storage"
)

func main() {
	// Start a new container
	c := foundation.NewContainer()
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

	paidSubscriptionsService := paidsubscriptions.New(paidsubscriptions.NewEntStore(
		c.ORM,
		c.Config.App.OperationalConstants.ProTrialTimespanInDays,
		c.Config.App.OperationalConstants.PaymentFailedGracePeriodInDays,
	))
	storageClient := storagerepo.NewStorageClient(c.Config, c.ORM)
	profileService := profilesvc.NewProfileService(c.ORM, storageClient, paidSubscriptionsService)

	var firebaseJSONAccessKeys *[]byte
	if len(c.Config.App.FirebaseJSONAccessKeys) > 0 {
		firebaseJSONAccessKeys = &c.Config.App.FirebaseJSONAccessKeys
	}
	notificationServices, err := notifications.New(notifications.RuntimeDeps{
		ORM:                                 c.ORM,
		PubSub:                              c.CorePubSub,
		SubscriptionService:                 paidSubscriptionsService,
		VapidPublicKey:                      c.Config.App.VapidPublicKey,
		VapidPrivateKey:                     c.Config.App.VapidPrivateKey,
		MailFromAddress:                     c.Config.Mail.FromAddress,
		FirebaseJSONAccessKeys:              firebaseJSONAccessKeys,
		SMSRegion:                           c.Config.Phone.Region,
		SMSSenderID:                         c.Config.Phone.SenderID,
		SMSValidationCodeExpirationMinutes:  c.Config.Phone.ValidationCodeExpirationMinutes,
		GetNumNotificationsForProfileByIDFn: profileService.GetCountOfUnseenNotifications,
	})
	if err != nil {
		log.Fatalf("failed to initialize notifications module: %v", err)
	}

	// Build the router, which is needed to get the reverse of routes by name in some tasks.
	if err := goship.BuildRouter(c, goship.RouterModules{
		PaidSubscriptions: paidSubscriptionsService,
		Notifications:     notificationServices,
	}); err != nil {
		c.Web.Logger.Fatalf("failed to build router: %v", err)
	}

	emailSubscriptionConfirmationProcessor := tasks.NewEmailSubscriptionConfirmationProcessor(
		c.Mail, c.Config,
	)

	emailUpdateProcessor := tasks.NewEmailUpdateProcessor(c, c.ORM)

	deactivateExpiredSubscriptionsProcessor := tasks.NewDeactivateExpiredSubscriptionsProcessor(paidSubscriptionsService)
	allDailyConvoNotificationsProcessor := tasks.NewAllDailyConvoNotificationsProcessor(c.ORM, profileService, notificationServices.PlannedNotificationsService, c.CoreJobs, 30)
	dailyConvoNotificationsProcessor := tasks.NewDailyConvoNotificationsProcessor(notificationServices.Notifier, c.Web, paidSubscriptionsService, notificationServices.PlannedNotificationsService)
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
