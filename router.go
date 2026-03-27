package goship

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship-modules/notifications"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	paidsubscriptionroutes "github.com/leomorpho/goship-modules/paidsubscriptions/routes"
	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/backup"
	frameworkbootstrap "github.com/leomorpho/goship/framework/bootstrap"
	"github.com/leomorpho/goship/framework/logging"
	"github.com/leomorpho/goship/framework/runtimeplan"
	frameworksecurity "github.com/leomorpho/goship/framework/security"
	frameworkweb "github.com/leomorpho/goship/framework/web"
	frameworkcontrollers "github.com/leomorpho/goship/framework/web/controllers"
	"github.com/leomorpho/goship/framework/web/middleware"
	routeNames "github.com/leomorpho/goship/framework/web/routenames"
	"github.com/leomorpho/goship/framework/web/ui"
)

type RouterModules struct {
	PaidSubscriptions *paidsubscriptions.Service
	Notifications     *notifications.Services
}

// BuildRouter is the canonical GoShip router entrypoint.
func BuildRouter(c *frameworkbootstrap.Container, modules RouterModules) error {
	if c == nil {
		return errors.New("invalid router container: nil")
	}
	if modules.PaidSubscriptions == nil {
		return errors.New("missing paid subscriptions module")
	}
	if modules.Notifications == nil {
		return errors.New("missing notifications module")
	}

	c.Notifier = modules.Notifications.Notifier
	_, webFeatures, err := resolveStartupWebFeatures(c)
	if err != nil {
		return err
	}
	deps, err := frameworkweb.NewRouteDeps(c, modules.PaidSubscriptions, modules.Notifications)
	if err != nil {
		return err
	}

	logger := logging.NewLogger(c.Config.Log)

	if err := frameworkweb.RegisterStaticRoutes(c); err != nil {
		return err
	}

	g := c.Web.Group("")
	e := c.Web.Group("")
	s := c.Web.Group("")
	v1 := e.Group("/api/v1") // ship:routes:api:v1:start
	_ = v1
	// ship:routes:api:v1:end

	if c.Config.HTTP.TLS.Enabled {
		frameworkweb.ApplyTLSRedirect(g, e, s)
	}

	frameworkweb.ApplyMainMiddleware(c, g, logger, deps, webFeatures)
	frameworkweb.ApplyRealtimeMiddleware(c, s, logger, deps)
	frameworkweb.ApplyExternalMiddleware(c, e, logger, deps)

	ctr := ui.NewController(c)
	errorHandler := frameworkcontrollers.NewErrorHandler(ctr)
	c.Web.HTTPErrorHandler = errorHandler.Get

	if err := registerPublicRoutes(c, g, ctr, deps); err != nil {
		return err
	}
	if !c.Config.App.OperationalConstants.UserSignupEnabled {
		return nil
	}
	if err := registerAuthRoutes(c, g, ctr, deps); err != nil {
		return err
	}
	if err := registerExternalRoutes(c, e, ctr, deps); err != nil {
		return err
	}
	if webFeatures.EnableRealtime {
		if err := registerRealtimeRoutes(c, s, ctr); err != nil {
			return err
		}
	} else {
		slog.Info("realtime SSE routes disabled (notifier/pubsub dependency unavailable)")
	}

	return nil
}

func resolveStartupWebFeatures(c *frameworkbootstrap.Container) (runtimeplan.Plan, runtimeplan.WebFeatures, error) {
	if c == nil || c.Config == nil {
		return runtimeplan.Plan{}, runtimeplan.WebFeatures{}, errors.New("invalid runtime container: config is nil")
	}

	plan, err := runtimeplan.Resolve(c.Config)
	if err != nil {
		return runtimeplan.Plan{}, runtimeplan.WebFeatures{}, fmt.Errorf("invalid runtime plan: %w", err)
	}

	if plan.RunWeb && c.Notifier == nil {
		return runtimeplan.Plan{}, runtimeplan.WebFeatures{}, errors.New("invalid startup capability contract: realtime requires notifier service")
	}

	features := runtimeplan.ResolveWebFeatures(plan, c.Cache != nil, c.Notifier != nil)
	return plan, features, nil
}

func registerPublicRoutes(c *frameworkbootstrap.Container, g *echo.Group, ctr ui.Controller, deps *frameworkweb.RouteDeps) error {
	_ = deps
	landingPage := frameworkcontrollers.NewLandingPageRoute(ctr)
	g.GET("/", landingPage.Get).Name = routeNames.RouteNameLandingPage

	clearCookie := frameworkcontrollers.NewClearCookiesRoute(ctr)
	g.GET("/clear-cookie", clearCookie.Get).Name = routeNames.RouteNameClearCookie

	g.GET(c.Config.App.TestSentryUrl, func(ctx echo.Context) error {
		panic("Test error for Sentry")
	})

	if ctr.Container.Config.App.Environment != config.EnvProduction {
		errHandler := frameworkcontrollers.NewErrorHandler(ctr)
		g.GET("/error/400", errHandler.GetHttp400BadRequest)
		g.GET("/error/401", errHandler.GetHttp401Unauthorized)
		g.GET("/error/403", errHandler.GetHttp403Forbidden)
		g.GET("/error/404", errHandler.GetHttp404NotFound)
		g.GET("/error/500", errHandler.GetHttp500InternalServerError)
	}
	registerMailPreviewRoutes(g, ctr)

	// ship:routes:public:start
	// ship:routes:public:end

	return nil
}

func registerMailPreviewRoutes(g *echo.Group, ctr ui.Controller) {
	if ctr.Container == nil || ctr.Container.Config == nil || ctr.Container.Config.App.Environment != config.EnvDevelop {
		return
	}

	mailPreview := frameworkcontrollers.NewMailPreviewRoute(ctr)
	mailGroup := g.Group("/dev/mail")
	mailGroup.GET("", mailPreview.Index).Name = routeNames.RouteNameMailPreviewIndex
	mailGroup.GET("/welcome", mailPreview.Welcome).Name = routeNames.RouteNameMailPreviewWelcome
	mailGroup.GET("/password-reset", mailPreview.PasswordReset).Name = routeNames.RouteNameMailPreviewPasswordReset
	mailGroup.GET("/verify-email", mailPreview.VerifyEmail).Name = routeNames.RouteNameMailPreviewVerifyEmail
}

func registerAuthRoutes(c *frameworkbootstrap.Container, g *echo.Group, ctr ui.Controller, deps *frameworkweb.RouteDeps) error {
	_ = c
	_ = g
	_ = ctr
	_ = deps

	// ship:routes:auth:start
	// ship:routes:auth:end

	return nil
}

func registerExternalRoutes(c *frameworkbootstrap.Container, e *echo.Group, ctr ui.Controller, deps *frameworkweb.RouteDeps) error {
	paymentsModule := paidsubscriptionroutes.NewRouteModule(ctr, deps.SubscriptionsRepo)
	if err := paymentsModule.RegisterExternalRoutes(e, deps.StripeWebhookPath); err != nil {
		return err
	}

	if c.Config.Managed.Enabled {
		managedHooks := frameworkcontrollers.NewManagedHooksRoute(ctr, frameworkcontrollers.ManagedHooksDeps{
			BackupDriver:  backup.NewSQLiteDriver(),
			RestoreDriver: backup.NoopRestorer{},
		})
		verifier := frameworksecurity.NewManagedHookVerifier(
			c.Config.Managed.HooksSecret,
			time.Duration(c.Config.Managed.HooksMaxSkewSeconds)*time.Second,
			time.Duration(c.Config.Managed.HooksNonceTTLSeconds)*time.Second,
		).WithPreviousSecret(c.Config.Managed.HooksPreviousSecret)

		managedGroup := e.Group("/managed", middleware.RequireManagedHookSignature(verifier))
		managedGroup.GET("/status", managedHooks.GetRuntimeStatus).Name = routeNames.RouteNameManagedStatus
		managedGroup.POST("/backup", managedHooks.StartBackup).Name = routeNames.RouteNameManagedBackup
		managedGroup.POST("/restore", managedHooks.StartRestore).Name = routeNames.RouteNameManagedRestore
	}

	// ship:routes:external:start
	// ship:routes:external:end

	return nil
}

func registerRealtimeRoutes(c *frameworkbootstrap.Container, s *echo.Group, ctr ui.Controller) error {
	if c.Notifier == nil {
		return errors.New("cannot register realtime routes: notifier is nil")
	}

	onboardedGroup := s.Group("/auth", middleware.RequireAuthentication())
	realtime := frameworkcontrollers.NewRealtimeRoute(ctr, *c.Notifier)
	onboardedGroup.GET("/realtime", realtime.Get).Name = routeNames.RouteNameRealtime
	return nil
}
