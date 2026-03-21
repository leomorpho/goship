package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hibiken/asynq"
	"github.com/leomorpho/goship-modules/notifications"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	"github.com/leomorpho/goship/app"
	"github.com/leomorpho/goship/app/foundation"
	frameworkbootstrap "github.com/leomorpho/goship/framework/bootstrap"
	"github.com/leomorpho/goship/framework/events"
	storagerepo "github.com/leomorpho/goship/framework/repos/storage"
	profilesvc "github.com/leomorpho/goship/modules/profile"
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

	plansCatalog, err := paidsubscriptions.BuildDefaultCatalog()
	if err != nil {
		log.Fatalf("failed to build subscription plans catalog: %v", err)
	}
	paidSubscriptionsService := paidsubscriptions.NewServiceWithCatalog(paidsubscriptions.NewSQLStore(
		c.Database,
		c.Config.Adapters.DB,
		c.Config.App.OperationalConstants.ProTrialTimespanInDays,
		c.Config.App.OperationalConstants.PaymentFailedGracePeriodInDays,
	), plansCatalog)
	if err := wireJobsModule(c); err != nil {
		log.Fatalf("failed to initialize jobs module: %v", err)
	}
	storageClient := storagerepo.NewStorageClient(c.Config, c.Database, c.Config.Adapters.DB)
	profileService := profilesvc.NewProfileServiceWithDBDeps(
		c.Database,
		c.Config.Adapters.DB,
		storageClient,
		paidSubscriptionsService,
		profilesvc.NewBobNotificationCountStore(c.Database, c.Config.Adapters.DB),
	)

	var firebaseJSONAccessKeys *[]byte
	if len(c.Config.App.FirebaseJSONAccessKeys) > 0 {
		firebaseJSONAccessKeys = &c.Config.App.FirebaseJSONAccessKeys
	}
	notificationServices, err := notifications.New(notifications.RuntimeDeps{
		DB:                                  c.Database,
		DBDialect:                           c.Config.Adapters.DB,
		PubSub:                              frameworkbootstrap.AdaptNotificationsPubSub(c.CorePubSub),
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

	// Map task types to the handlers
	mux := asynq.NewServeMux()
	mux.HandleFunc(events.AsyncJobName, func(ctx context.Context, task *asynq.Task) error {
		return events.DeliverAsync(ctx, c.EventBus, task.Payload())
	})

	stopScheduler := startWorkerScheduler(c.Scheduler)
	defer stopScheduler()

	// Start the worker server
	if err := srv.Run(mux); err != nil {
		log.Fatalf("could not run worker server: %v", err)
	}
}

func wireJobsModule(c *foundation.Container) error {
	runtime, err := frameworkbootstrap.WireJobsRuntime(c.Config, c.Database, frameworkbootstrap.JobsProcessWorker)
	if err != nil {
		return err
	}
	c.CoreJobs = runtime.Jobs
	c.CoreJobsInspector = runtime.Inspector
	return nil
}
