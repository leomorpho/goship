package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/leomorpho/goship-modules/notifications"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	shipapp "github.com/leomorpho/goship/app"
	frameworkbootstrap "github.com/leomorpho/goship/framework/bootstrap"
	"github.com/leomorpho/goship/framework/events"
)

func timeoutMiddleware(next http.Handler, writeTimeout time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the request is an SSE request
		if r.Header.Get("Accept") == "text/event-stream" {
			// SSE request, set indefinite write timeout
			next.ServeHTTP(w, r)
		} else {
			// For non-SSE requests, set a standard write timeout
			ctx, cancel := context.WithTimeout(r.Context(), writeTimeout) // Adjust timeout as needed
			defer cancel()

			next.ServeHTTP(w, r.WithContext(ctx))
		}
	})
}

func main() {
	// Start a new container
	c := shipapp.NewContainer()
	defer func() {
		if err := c.Shutdown(); err != nil {
			c.Web.Logger.Fatal(err)
		}
	}()

	firstPartyServices, err := frameworkbootstrap.BuildFirstPartyServices(c, frameworkbootstrap.JobsProcessWeb)
	if err != nil {
		c.Web.Logger.Fatalf("failed to initialize first-party services: %v", err)
	}

	// Build the router
	if err := shipapp.BuildRouter(c, shipapp.RouterModules{
		PaidSubscriptions: firstPartyServices.PaidSubscriptions,
		Notifications:     firstPartyServices.Notifications,
	}); err != nil {
		c.Web.Logger.Fatalf("failed to build router: %v", err)
	}

	jobsCtx, jobsCancel := context.WithCancel(context.Background())
	defer jobsCancel()
	if err := startEmbeddedJobsWorker(jobsCtx, c, firstPartyServices.PaidSubscriptions, firstPartyServices.Notifications); err != nil {
		c.Web.Logger.Fatalf("failed to start embedded jobs worker: %v", err)
	}

	// Start the server
	go func() {
		srv := http.Server{
			Addr:        fmt.Sprintf("%s:%d", c.Config.HTTP.Hostname, c.Config.HTTP.Port),
			Handler:     timeoutMiddleware(c.Web, c.Config.HTTP.WriteTimeout),
			ReadTimeout: c.Config.HTTP.ReadTimeout,
			IdleTimeout: c.Config.HTTP.IdleTimeout,
		}

		if c.Config.HTTP.TLS.Enabled {
			certs, err := tls.LoadX509KeyPair(c.Config.HTTP.TLS.Certificate, c.Config.HTTP.TLS.Key)
			if err != nil {
				c.Web.Logger.Fatalf("cannot load TLS certificate: %v", err)
			}

			srv.TLSConfig = &tls.Config{
				Certificates: []tls.Certificate{certs},
			}
		}

		if err := c.Web.StartServer(&srv); errors.Is(err, http.ErrServerClosed) {
			c.Web.Logger.Fatalf("shutting down the server: %v", err)
		}
	}()

	// // Start the scheduler service to queue periodic tasks
	// go func() {
	// 	if err := c.Tasks.StartScheduler(); err != nil {
	// 		c.Web.Logger.Fatalf("scheduler shutdown: %v", err)
	// 	}
	// }()

	// seeder.RunIdempotentSeeder(c.Config, c.Database)

	// // Start the scheduled tasks
	// if err := c.Tasks.
	// 	New(tasks.TypeDeactivateExpiredSubscriptions).
	// 	Periodic("@every 6h").
	// 	Timeout(120 * time.Second).
	// 	Retain(24 * time.Hour).
	// 	Save(); err != nil {
	// 	c.Web.Logger.Fatalf("failed to register scheduler task: %v", err)
	// }
	// if err := c.Tasks.
	// 	New(tasks.TypeDeleteStaleNotifications).
	// 	Periodic("@every 12h").
	// 	Timeout(120 * time.Second).
	// 	Retain(24 * time.Hour).
	// 	Save(); err != nil {
	// 	c.Web.Logger.Fatalf("failed to register scheduler task: %v", err)
	// }
	// // NOTE: we run the following task every 30 minutes, but it will check if the same notif type has
	// // not already been sent to services.
	// if err := c.Tasks.
	// 	New(tasks.TypeAllDailyConvoNotifications).
	// 	Periodic("@every 30m").
	// 	Timeout(120 * time.Second).
	// 	Retain(24 * time.Hour).
	// 	Save(); err != nil {
	// 	c.Web.Logger.Fatalf("failed to register scheduler task: %v", err)
	// }
	// if err := c.Tasks.
	// 	New(tasks.TypeEmailUpdates).
	// 	Periodic("@every 6h").
	// 	Timeout(30 * time.Minute).
	// 	Retain(48 * time.Hour).
	// 	Save(); err != nil {
	// 	c.Web.Logger.Fatalf("failed to run startup task: %v", err)
	// }

	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	signal.Notify(quit, os.Kill)
	<-quit
	jobsCancel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := c.CoreJobs.Stop(ctx); err != nil {
		c.Web.Logger.Fatal(err)
	}
	if err := c.Web.Shutdown(ctx); err != nil {
		c.Web.Logger.Fatal(err)
	}
}
func startEmbeddedJobsWorker(
	ctx context.Context,
	c *shipapp.Container,
	paidSubscriptionsService *paidsubscriptions.Service,
	notificationServices *notifications.Services,
) error {
	_ = paidSubscriptionsService
	_ = notificationServices
	if c.Config.Adapters.Jobs != "backlite" {
		return nil
	}

	if err := c.CoreJobs.Register(events.AsyncJobName, func(ctx context.Context, payload []byte) error {
		return events.DeliverAsync(ctx, c.EventBus, payload)
	}); err != nil {
		return err
	}
	if notificationServices != nil && notificationServices.Notifier != nil {
		if err := c.CoreJobs.Register(notifications.DeliverPushNotificationJobName, notificationServices.Notifier.HandleDeliverPushNotificationJob); err != nil {
			return err
		}
	}

	go func() {
		if err := c.CoreJobs.StartWorker(ctx); err != nil && !errors.Is(err, context.Canceled) {
			c.Web.Logger.Errorf("embedded jobs worker stopped: %v", err)
		}
	}()
	return nil
}
