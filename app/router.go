package app

import (
	"errors"
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship-modules/notifications"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	paidsubscriptionroutes "github.com/leomorpho/goship-modules/paidsubscriptions/routes"
	"github.com/leomorpho/goship/config/runtimeplan"
	frameworkapi "github.com/leomorpho/goship/framework/api"
	frameworkbootstrap "github.com/leomorpho/goship/framework/bootstrap"
	"github.com/leomorpho/goship/framework/logging"
	frameworkweb "github.com/leomorpho/goship/framework/http"
	frameworkcontrollers "github.com/leomorpho/goship/framework/http/controllers"
	"github.com/leomorpho/goship/framework/http/ui"
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
	v1 := e.Group("/api/v1") // ship:routes:api:v1:start
	v1.GET("/status", func(ctx echo.Context) error {
		return frameworkapi.OK(ctx, map[string]string{"status": "ok"})
	}).Name = "api_v1_status"
	// ship:routes:api:v1:end

	if c.Config.HTTP.TLS.Enabled {
		frameworkweb.ApplyTLSRedirect(g, e)
	}

	frameworkweb.ApplyMainMiddleware(c, g, logger, deps, webFeatures)
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
	_ = webFeatures

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

	features := runtimeplan.ResolveWebFeatures(plan, c.Cache != nil, c.Notifier != nil)
	return plan, features, nil
}

func registerPublicRoutes(c *frameworkbootstrap.Container, g *echo.Group, ctr ui.Controller, deps *frameworkweb.RouteDeps) error {
	_ = deps
	_ = ctr
	g.GET("/", func(ctx echo.Context) error {
		return ctx.String(200, "GoShip")
	}).Name = "landing_page"

	g.GET("/clear-cookie", func(ctx echo.Context) error {
		_ = c.Auth.Logout(ctx)
		return ctx.Redirect(303, "/")
	}).Name = "clear_cookie"

	g.GET(c.Config.App.TestSentryUrl, func(ctx echo.Context) error {
		panic("Test error for Sentry")
	})

	// ship:routes:public:start
	// ship:routes:public:end

	return nil
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

	// ship:routes:external:start
	// ship:routes:external:end

	return nil
}
