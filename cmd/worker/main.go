package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hibiken/asynq"
	"github.com/leomorpho/goship-modules/notifications"
	shipapp "github.com/leomorpho/goship/app"
	frameworkbootstrap "github.com/leomorpho/goship/framework/bootstrap"
	"github.com/leomorpho/goship/framework/events"
)

func main() {
	// Start a new container
	c := shipapp.NewContainer()
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

	firstPartyServices, err := frameworkbootstrap.BuildFirstPartyServices(c, frameworkbootstrap.JobsProcessWorker)
	if err != nil {
		log.Fatalf("failed to initialize first-party services: %v", err)
	}

	// Build the router, which is needed to get the reverse of routes by name in some tasks.
	if err := shipapp.BuildRouter(c, shipapp.RouterModules{
		PaidSubscriptions: firstPartyServices.PaidSubscriptions,
		Notifications:     firstPartyServices.Notifications,
	}); err != nil {
		c.Web.Logger.Fatalf("failed to build router: %v", err)
	}

	// Map task types to the handlers
	mux := asynq.NewServeMux()
	mux.HandleFunc(events.AsyncJobName, func(ctx context.Context, task *asynq.Task) error {
		return events.DeliverAsync(ctx, c.EventBus, task.Payload())
	})
	if firstPartyServices.Notifications != nil && firstPartyServices.Notifications.Notifier != nil {
		mux.HandleFunc(notifications.DeliverPushNotificationJobName, func(ctx context.Context, task *asynq.Task) error {
			return firstPartyServices.Notifications.Notifier.HandleDeliverPushNotificationJob(ctx, task.Payload())
		})
	}

	stopScheduler := startWorkerScheduler(c.Scheduler)
	defer stopScheduler()

	// Start the worker server
	if err := srv.Run(mux); err != nil {
		log.Fatalf("could not run worker server: %v", err)
	}
}
