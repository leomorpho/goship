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

	jobsmodule "github.com/leomorpho/goship-modules/jobs"
	"github.com/leomorpho/goship-modules/notifications"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	"github.com/leomorpho/goship/app"
	"github.com/leomorpho/goship/app/foundation"
	profilesvc "github.com/leomorpho/goship/app/profile"
	storagerepo "github.com/leomorpho/goship/framework/repos/storage"
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
	c := foundation.NewContainer()
	defer func() {
		if err := c.Shutdown(); err != nil {
			c.Web.Logger.Fatal(err)
		}
	}()

	paidSubscriptionsService := paidsubscriptions.New(paidsubscriptions.NewSQLStore(
		c.Database,
		c.Config.Adapters.DB,
		c.Config.App.OperationalConstants.ProTrialTimespanInDays,
		c.Config.App.OperationalConstants.PaymentFailedGracePeriodInDays,
	))

	if err := wireJobsModule(c); err != nil {
		c.Web.Logger.Fatalf("failed to initialize jobs module: %v", err)
	}
	storageClient := storagerepo.NewStorageClient(c.Config, c.ORM)
	profileService := profilesvc.NewProfileServiceWithDeps(
		c.ORM,
		storageClient,
		paidSubscriptionsService,
		profilesvc.NewBobNotificationCountStore(c.Database, c.Config.Adapters.DB),
	)

	var firebaseJSONAccessKeys *[]byte
	if len(c.Config.App.FirebaseJSONAccessKeys) > 0 {
		firebaseJSONAccessKeys = &c.Config.App.FirebaseJSONAccessKeys
	}
	notificationServices, err := notifications.New(notifications.RuntimeDeps{
		ORM:                                 nil,
		DB:                                  c.Database,
		DBDialect:                           c.Config.Adapters.DB,
		PubSub:                              foundation.AdaptNotificationsPubSub(c.CorePubSub),
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
		c.Web.Logger.Fatalf("failed to initialize notifications module: %v", err)
	}

	// Build the router
	if err := goship.BuildRouter(c, goship.RouterModules{
		PaidSubscriptions: paidSubscriptionsService,
		Notifications:     notificationServices,
	}); err != nil {
		c.Web.Logger.Fatalf("failed to build router: %v", err)
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

	// seeder.RunIdempotentSeeder(c.Config, c.ORM)

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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := c.Web.Shutdown(ctx); err != nil {
		c.Web.Logger.Fatal(err)
	}
}

func wireJobsModule(c *foundation.Container) error {
	switch c.Config.Adapters.Jobs {
	case "asynq":
		mod, err := jobsmodule.New(jobsmodule.Config{
			Backend: jobsmodule.BackendRedis,
			Redis: jobsmodule.RedisConfig{
				Addr:     fmt.Sprintf("%s:%d", c.Config.Cache.Hostname, c.Config.Cache.Port),
				Password: c.Config.Cache.Password,
				DB:       c.Config.Cache.Database,
			},
		})
		if err != nil {
			return err
		}
		c.CoreJobs = foundation.AdaptModuleJobs(mod.Jobs())
		c.CoreJobsInspector = foundation.AdaptModuleJobsInspector(mod.Inspector())
		return nil
	case "dbqueue":
		mod, err := jobsmodule.New(jobsmodule.Config{
			Backend: jobsmodule.BackendSQL,
			SQLDB:   c.Database,
		})
		if err == nil {
			c.CoreJobs = foundation.AdaptModuleJobs(mod.Jobs())
			c.CoreJobsInspector = foundation.AdaptModuleJobsInspector(mod.Inspector())
		}
		return err
	case "inproc":
		return nil
	default:
		return fmt.Errorf("unsupported jobs adapter %q", c.Config.Adapters.Jobs)
	}
}
