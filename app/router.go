package goship

import (
	"errors"
	"log/slog"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship-modules/notifications"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	"github.com/leomorpho/goship/app/foundation"
	appweb "github.com/leomorpho/goship/app/web"
	"github.com/leomorpho/goship/app/web/controllers"
	"github.com/leomorpho/goship/app/web/middleware"
	routeNames "github.com/leomorpho/goship/app/web/routenames"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/runtimeplan"
	authmodule "github.com/leomorpho/goship/modules/auth"
	"github.com/rs/zerolog/log"
)

type RouterModules struct {
	PaidSubscriptions *paidsubscriptions.Service
	Notifications     *notifications.Services
}

// BuildRouter is the canonical app-level router entrypoint.
func BuildRouter(c *foundation.Container, modules RouterModules) error {
	if modules.PaidSubscriptions == nil {
		return errors.New("missing paid subscriptions module")
	}
	if modules.Notifications == nil {
		return errors.New("missing notifications module")
	}

	deps, err := appweb.NewRouteDeps(c, modules.PaidSubscriptions, modules.Notifications)
	if err != nil {
		return err
	}
	c.Notifier = modules.Notifications.Notifier

	plan, err := runtimeplan.Resolve(c.Config)
	if err != nil {
		log.Warn().
			Err(err).
			Msg("invalid runtime plan configuration, falling back to safe web defaults")
		plan = runtimeplan.Plan{
			Profile: string(c.Config.Runtime.Profile),
			RunWeb:  true,
			Adapters: runtimeplan.Adapters{
				PubSub: c.Config.Adapters.PubSub,
			},
		}
	}
	webFeatures := runtimeplan.ResolveWebFeatures(plan, c.Cache != nil, c.Notifier != nil)

	// Create a slog logger, which logs to json.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	appweb.RegisterStaticRoutes(c)

	// Non static file route groups.
	g := c.Web.Group("")
	e := c.Web.Group("")
	s := c.Web.Group("")

	if c.Config.HTTP.TLS.Enabled {
		appweb.ApplyTLSRedirect(g, e, s)
	}

	appweb.ApplyMainMiddleware(c, g, logger, deps, webFeatures)
	appweb.ApplyRealtimeMiddleware(c, s, deps)
	appweb.ApplyExternalMiddleware(c, e, deps)

	ctr := ui.NewController(c)
	errorHandler := controllers.NewErrorHandler(ctr)
	c.Web.HTTPErrorHandler = errorHandler.Get

	if err := registerPublicRoutes(c, g, ctr, deps); err != nil {
		return err
	}
	if err := registerDocsRoutes(g, ctr); err != nil {
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
		log.Info().Msg("realtime SSE routes disabled (notifier/pubsub dependency unavailable)")
	}

	return nil
}

func registerPublicRoutes(c *foundation.Container, g *echo.Group, ctr ui.Controller, deps *appweb.RouteDeps) error {
	landingPage := controllers.NewLandingPageRoute(ctr)
	g.GET("/", landingPage.Get).Name = routeNames.RouteNameLandingPage

	clearCookie := controllers.NewClearCookiesRoute(ctr)
	g.GET("/clear-cookie", clearCookie.Get).Name = routeNames.RouteNameClearCookie

	healthcheck := controllers.NewHealthCheckRoute(ctr)
	g.GET("/up", healthcheck.Get).Name = routeNames.RouteNameHealthcheck

	// TODO: remove once sentry is stable.
	g.GET(c.Config.App.TestSentryUrl, func(ctx echo.Context) error {
		panic("Test error for Sentry")
	})

	emailSubscribe := controllers.NewEmailSubscribeRoute(ctr, *deps.EmailSubscriptions, *c.Config)
	g.GET("/emailSubscribe", emailSubscribe.Get).Name = routeNames.RouteNameEmailSubscribe
	g.POST("/emailSubscribe", emailSubscribe.Post).Name = routeNames.RouteNameEmailSubscribeSubmit

	verifyEmailSubscription := controllers.NewVerifyEmailSubscriptionRoute(ctr, *deps.EmailSubscriptions)
	g.GET("/email/subscription/:token", verifyEmailSubscription.Get).Name = routeNames.RouteNameVerifyEmailSubscription

	installApp := controllers.NewInstallAppRoute(ctr)
	g.GET("/install-app", installApp.GetInstallPage).Name = routeNames.RouteNameInstallApp

	about := controllers.NewAboutUsRoute(ctr)
	g.GET("/about", about.Get).Name = routeNames.RouteNameAboutUs

	privacyPolicy := controllers.NewPrivacyPolicyRoute(ctr)
	g.GET("/privacy-policy", privacyPolicy.Get).Name = routeNames.RouteNamePrivacyPolicy

	if ctr.Container.Config.App.Environment != config.EnvProduction {
		errHandler := controllers.NewErrorHandler(ctr)
		g.GET("/error/400", errHandler.GetHttp400BadRequest)
		g.GET("/error/401", errHandler.GetHttp401Unauthorized)
		g.GET("/error/403", errHandler.GetHttp403Forbidden)
		g.GET("/error/404", errHandler.GetHttp404NotFound)
		g.GET("/error/500", errHandler.GetHttp500InternalServerError)
	}

	// ship:routes:public:start
	// ship:routes:public:end

	return nil
}

func registerDocsRoutes(g *echo.Group, ctr ui.Controller) error {
	docsRoute := controllers.NewDocsRoute(ctr)
	g.GET("/docs", docsRoute.GetDocsHome).Name = routeNames.RouteNameDocs
	g.GET("/docs/gettingStarted", docsRoute.GetDocsGettingStarted).Name = routeNames.RouteNameDocsGettingStarted
	g.GET("/docs/guidedTour", docsRoute.GetDocsGuidedTour).Name = routeNames.RouteNameDocsGuidedTour
	g.GET("/docs/architecture", docsRoute.GetDocsArchitecture).Name = routeNames.RouteNameDocsArchitecture
	return nil
}

func registerAuthRoutes(c *foundation.Container, g *echo.Group, ctr ui.Controller, deps *appweb.RouteDeps) error {
	authModule := authmodule.New(authmodule.Deps{
		Controller:                    ctr,
		ProfileService:                *deps.ProfileService,
		SubscriptionsService:          deps.SubscriptionsRepo,
		NotificationPermissionService: deps.NotificationPermissionService,
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

	outgoingNotifications := controllers.NewPushNotifsRoute(
		ctr,
		deps.ProfileService,
		deps.PwaPushService,
		deps.FcmPushService,
		deps.NotificationPermissionService,
	)
	onboardingGroup.GET("/subscription/push", outgoingNotifications.GetPushSubscriptions).Name = routeNames.RouteNameGetPushSubscriptions
	onboardingGroup.POST("/subscription/:platform", outgoingNotifications.RegisterSubscription).Name = routeNames.RouteNameRegisterSubscription
	onboardingGroup.DELETE("/subscription/:platform", outgoingNotifications.DeleteSubscription).Name = routeNames.RouteNameDeleteSubscription
	onboardingGroup.GET("/email-subscription/unsubscribe/:permission/:token", outgoingNotifications.DeleteEmailSubscription).Name = routeNames.RouteNameDeleteEmailSubscriptionWithToken

	onboardedGroup := g.Group("/auth", middleware.RequireAuthentication(), middleware.RedirectToOnboardingIfNotComplete())

	homeFeed := controllers.NewHomeFeedRoute(ctr, *deps.ProfileService, &c.Config.App.PageSize)
	onboardedGroup.GET("/homeFeed", homeFeed.Get, middleware.SetLastSeenOnline(c.Auth)).Name = routeNames.RouteNameHomeFeed
	onboardedGroup.GET("/homeFeed/buttons", homeFeed.GetHomeButtons).Name = routeNames.RouteNameGetHomeFeedButtons

	singleProfile := controllers.NewProfileRoutes(ctr, deps.ProfileService)
	onboardedGroup.GET("/profile", singleProfile.Get).Name = routeNames.RouteNameProfile

	uploadPhoto := controllers.NewUploadPhotoRoutes(ctr, deps.ProfileService, deps.StorageRepo, c.Config.Storage.PhotosMaxFileSizeMB)
	onboardedGroup.GET("/uploadPhoto", uploadPhoto.Get).Name = routeNames.RouteNameUploadPhoto
	onboardedGroup.POST("/uploadPhoto", uploadPhoto.Post).Name = routeNames.RouteNameUploadPhotoSubmit
	onboardedGroup.DELETE("/uploadPhoto/:image_id", uploadPhoto.Delete).Name = routeNames.RouteNameUploadPhotoDelete

	currProfilePhoto := controllers.NewCurrProfilePhotoRoutes(ctr, deps.ProfileService, deps.StorageRepo, c.Config.Storage.PhotosMaxFileSizeMB)
	onboardedGroup.GET("/currProfilePhoto", currProfilePhoto.Get).Name = routeNames.RouteNameCurrentProfilePhoto
	onboardedGroup.POST("/currProfilePhoto", currProfilePhoto.Post).Name = routeNames.RouteNameCurrentProfilePhotoSubmit

	normalNotificationsCount := controllers.NewNormalNotificationsCountRoute(ctr, deps.ProfileService)
	onboardedGroup.GET("/notifications/normalNotificationsCount", normalNotificationsCount.Get).Name = routeNames.RouteNameNormalNotificationsCount

	payments := controllers.NewPaymentsRoute(ctr, deps.SubscriptionsRepo)
	onboardedGroup.GET("/payments/get-public-key", payments.GetPaymentProcessorPublickey).Name = routeNames.RouteNamePaymentProcessorGetPublicKey
	onboardedGroup.POST("/payments/create-checkout-session", payments.CreateCheckoutSession).Name = routeNames.RouteNameCreateCheckoutSession
	onboardedGroup.POST("/payments/create-portal-session", payments.CreatePortalSession).Name = routeNames.RouteNameCreatePortalSession
	onboardedGroup.GET("/payments/pricing", payments.PricingPage).Name = routeNames.RouteNamePricingPage
	onboardedGroup.GET("/payments/success", payments.SuccessfullySubscribed).Name = routeNames.RouteNamePaymentProcessorSuccess

	// ship:routes:auth:start
	// ship:routes:auth:end

	return nil
}

func registerExternalRoutes(c *foundation.Container, e *echo.Group, ctr ui.Controller, deps *appweb.RouteDeps) error {
	payments := controllers.NewPaymentsRoute(ctr, deps.SubscriptionsRepo)
	e.POST(deps.StripeWebhookPath, payments.HandleWebhook).Name = routeNames.RouteNamePaymentProcessorWebhook

	// ship:routes:external:start
	// ship:routes:external:end

	return nil
}

func registerRealtimeRoutes(c *foundation.Container, s *echo.Group, ctr ui.Controller) error {
	if c.Notifier == nil {
		return errors.New("cannot register realtime routes: notifier is nil")
	}

	onboardedGroup := s.Group("/auth", middleware.RequireAuthentication())
	realtime := controllers.NewRealtimeRoute(ctr, *c.Notifier)
	onboardedGroup.GET("/realtime", realtime.Get).Name = routeNames.RouteNameRealtime
	return nil
}
