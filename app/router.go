package goship

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship-modules/notifications"
	notificationroutes "github.com/leomorpho/goship-modules/notifications/routes"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	paidsubscriptionroutes "github.com/leomorpho/goship-modules/paidsubscriptions/routes"
	"github.com/leomorpho/goship/app/foundation"
	"github.com/leomorpho/goship/app/web/controllers"
	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/backup"
	"github.com/leomorpho/goship/framework/logging"
	"github.com/leomorpho/goship/framework/runtimeplan"
	frameworksecurity "github.com/leomorpho/goship/framework/security"
	frameworkweb "github.com/leomorpho/goship/framework/web"
	frameworkcontrollers "github.com/leomorpho/goship/framework/web/controllers"
	"github.com/leomorpho/goship/framework/web/middleware"
	routeNames "github.com/leomorpho/goship/framework/web/routenames"
	"github.com/leomorpho/goship/framework/web/ui"
	twofamodule "github.com/leomorpho/goship/modules/2fa"
	adminmodule "github.com/leomorpho/goship/modules/admin"
	authmodule "github.com/leomorpho/goship/modules/auth"
	profilemodule "github.com/leomorpho/goship/modules/profile"
	pwamodule "github.com/leomorpho/goship/modules/pwa"
)

type RouterModules struct {
	PaidSubscriptions *paidsubscriptions.Service
	Notifications     *notifications.Services
}

// BuildRouter is the canonical app-level router entrypoint.
func BuildRouter(c *foundation.Container, modules RouterModules) error {
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

	// Create a slog logger.
	logger := logging.NewLogger(c.Config.Log)

	if err := frameworkweb.RegisterStaticRoutes(c); err != nil {
		return err
	}

	// Non static file route groups.
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

func resolveStartupWebFeatures(c *foundation.Container) (runtimeplan.Plan, runtimeplan.WebFeatures, error) {
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

func registerPublicRoutes(c *foundation.Container, g *echo.Group, ctr ui.Controller, deps *frameworkweb.RouteDeps) error {
	landingPage := controllers.NewLandingPageRoute(ctr)
	g.GET("/", landingPage.Get).Name = routeNames.RouteNameLandingPage

	clearCookie := frameworkcontrollers.NewClearCookiesRoute(ctr)
	g.GET("/clear-cookie", clearCookie.Get).Name = routeNames.RouteNameClearCookie

	healthcheck := frameworkcontrollers.NewHealthCheckRoute(ctr)
	g.GET("/up", healthcheck.GetLiveness).Name = routeNames.RouteNameHealthcheck
	g.GET("/health", healthcheck.GetLiveness).Name = routeNames.RouteNameHealthLiveness
	g.GET("/health/ready", healthcheck.GetReadiness).Name = routeNames.RouteNameHealthReadiness

	g.GET(c.Config.App.TestSentryUrl, func(ctx echo.Context) error {
		panic("Test error for Sentry")
	})

	pwaModule := pwamodule.NewModule(pwamodule.NewRouteService(ctr))
	if err := pwaModule.RegisterRoutes(g); err != nil {
		return err
	}

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

	mailPreview := controllers.NewMailPreviewRoute(ctr)
	mailGroup := g.Group("/dev/mail")
	mailGroup.GET("", mailPreview.Index).Name = routeNames.RouteNameMailPreviewIndex
	mailGroup.GET("/welcome", mailPreview.Welcome).Name = routeNames.RouteNameMailPreviewWelcome
	mailGroup.GET("/password-reset", mailPreview.PasswordReset).Name = routeNames.RouteNameMailPreviewPasswordReset
	mailGroup.GET("/verify-email", mailPreview.VerifyEmail).Name = routeNames.RouteNameMailPreviewVerifyEmail
}

func registerAuthRoutes(c *foundation.Container, g *echo.Group, ctr ui.Controller, deps *frameworkweb.RouteDeps) error {
	twoFactorService := twofamodule.NewService(
		twofamodule.NewSQLStore(c.Database, c.Config.Adapters.DB),
		string(c.Config.App.Name),
		c.Config.App.EncryptionKey,
	)
	authModule := authmodule.New(authmodule.Deps{
		Controller:                    ctr,
		ProfileService:                *deps.ProfileService,
		SubscriptionsService:          deps.SubscriptionsRepo,
		NotificationPermissionService: deps.NotificationPermissionService,
		TwoFactorAuth:                 twoFactorService,
	})
	if err := authModule.RegisterRoutes(g); err != nil {
		return err
	}

	onboardingGroup := g.Group("/welcome", middleware.RequireAuthentication())
	preferences := controllers.NewPreferencesRoute(
		ctr,
		deps.ProfileService,
		deps.PwaPushService,
		deps.NotificationPermissionService,
		deps.SubscriptionsRepo,
		deps.SMSSenderService,
	)
	onboardingGroup.GET("/preferences", preferences.Get).Name = routeNames.RouteNamePreferences
	onboardingGroup.GET("/preferences/phone", preferences.GetPhoneComponent).Name = routeNames.RouteNameGetPhone
	onboardingGroup.GET("/preferences/phone/verification", preferences.GetPhoneVerificationComponent).Name = routeNames.RouteNameGetPhoneVerification
	onboardingGroup.POST("/preferences/phone/verification", preferences.SubmitPhoneVerificationCode).Name = routeNames.RouteNameSubmitPhoneVerification
	onboardingGroup.POST("/preferences/phone/save", preferences.SavePhoneInfo).Name = routeNames.RouteNameUpdatePhoneNum
	onboardingGroup.GET("/preferences/display-name/get", preferences.GetDisplayName).Name = routeNames.RouteNameGetDisplayName
	onboardingGroup.POST("/preferences/display-name/save", preferences.SaveDisplayName).Name = routeNames.RouteNameUpdateDisplayName

	deleteAccountRoute := controllers.NewDeleteAccountRoute(ctr, deps.ProfileService, deps.SubscriptionsRepo)
	onboardingGroup.GET("/preferences/delete-account", deleteAccountRoute.DeleteAccountPage).Name = routeNames.RouteNameDeleteAccountPage
	onboardingGroup.GET("/preferences/delete-account/now", deleteAccountRoute.DeleteAccountRequest).Name = routeNames.RouteNameDeleteAccountRequest

	finishOnboarding := controllers.NewOnboardingRoute(ctr, deps.ProfileService)
	onboardingGroup.GET("/finish-onboarding", finishOnboarding.Get).Name = routeNames.RouteNameFinishOnboarding

	profilePrefs := controllers.NewProfilePrefsRoute(ctr, deps.ProfileService)
	onboardingGroup.GET("/profileBio", profilePrefs.GetBio).Name = routeNames.RouteNameGetBio
	onboardingGroup.POST("/profileBio/update", profilePrefs.UpdateBio).Name = routeNames.RouteNameUpdateBio

	twoFactorModule := twofamodule.NewModule(twofamodule.ModuleDeps{
		Controller: ctr,
		Service:    twoFactorService,
	})
	if err := twoFactorModule.RegisterRoutes(g); err != nil {
		return err
	}

	onboardedGroup := g.Group("/auth", middleware.RequireAuthentication(), middleware.RedirectToOnboardingIfNotComplete())

	homeFeed := controllers.NewHomeFeedRoute(ctr, *deps.ProfileService, &c.Config.App.PageSize)
	onboardedGroup.GET("/homeFeed", homeFeed.Get, middleware.SetLastSeenOnline(c.Auth)).Name = routeNames.RouteNameHomeFeed
	onboardedGroup.GET("/homeFeed/buttons", homeFeed.GetHomeButtons).Name = routeNames.RouteNameGetHomeFeedButtons

	profileModule := profilemodule.NewModule(profilemodule.ModuleDeps{
		Controller:     ctr,
		ProfileService: deps.ProfileService,
		MaxFileSizeMB:  c.Config.Storage.PhotosMaxFileSizeMB,
	})
	if err := profileModule.RegisterRoutes(onboardedGroup); err != nil {
		return err
	}

	notificationsModule := notificationroutes.NewRouteModule(notificationroutes.RouteModuleDeps{
		Controller:                    ctr,
		ProfileService:                deps.ProfileService,
		NotifierService:               deps.NotifierService,
		PwaPushService:                deps.PwaPushService,
		FcmPushService:                deps.FcmPushService,
		NotificationPermissionService: deps.NotificationPermissionService,
	})
	if err := notificationsModule.RegisterOnboardingRoutes(onboardingGroup); err != nil {
		return err
	}
	if err := notificationsModule.RegisterRoutes(onboardedGroup); err != nil {
		return err
	}

	paymentsModule := paidsubscriptionroutes.NewRouteModule(ctr, deps.SubscriptionsRepo)
	if err := paymentsModule.RegisterRoutes(onboardedGroup); err != nil {
		return err
	}
	adminPanelModule := adminmodule.New(adminmodule.ModuleDeps{
		Controller: ctr,
		DB:         c.Database,
		AuditLogs:  c.AuditLogs,
		Flags:      c.Flags,
	})
	if err := adminPanelModule.RegisterRoutes(onboardedGroup); err != nil {
		return err
	}

	// ship:routes:auth:start
	// ship:routes:auth:end

	return nil
}

func registerExternalRoutes(c *foundation.Container, e *echo.Group, ctr ui.Controller, deps *frameworkweb.RouteDeps) error {
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
		)

		managedGroup := e.Group("/managed", middleware.RequireManagedHookSignature(verifier))
		managedGroup.GET("/status", managedHooks.GetRuntimeStatus).Name = routeNames.RouteNameManagedStatus
		managedGroup.POST("/backup", managedHooks.StartBackup).Name = routeNames.RouteNameManagedBackup
		managedGroup.POST("/restore", managedHooks.StartRestore).Name = routeNames.RouteNameManagedRestore
	}

	// ship:routes:external:start
	// ship:routes:external:end

	return nil
}

func registerRealtimeRoutes(c *foundation.Container, s *echo.Group, ctr ui.Controller) error {
	if c.Notifier == nil {
		return errors.New("cannot register realtime routes: notifier is nil")
	}

	onboardedGroup := s.Group("/auth", middleware.RequireAuthentication())
	realtime := frameworkcontrollers.NewRealtimeRoute(ctr, *c.Notifier)
	onboardedGroup.GET("/realtime", realtime.Get).Name = routeNames.RouteNameRealtime
	return nil
}
