package goship

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/apps/site/app/notifications"
	"github.com/leomorpho/goship/apps/site/foundation"
	appweb "github.com/leomorpho/goship/apps/site/web"
	"github.com/leomorpho/goship/apps/site/web/controllers"
	"github.com/leomorpho/goship/apps/site/web/middleware"
	routeNames "github.com/leomorpho/goship/apps/site/web/routenames"
	"github.com/leomorpho/goship/apps/site/web/ui"
	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/pkg/runtimeplan"
	"github.com/rs/zerolog/log"
)

// BuildRouter is the canonical app-level router entrypoint.
func BuildRouter(c *foundation.Container) error {
	deps, err := appweb.NewRouteDeps(c)
	if err != nil {
		return err
	}

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

	userGroup := g.Group("/user", middleware.RequireNoAuthentication())

	login := controllers.NewLoginRoute(ctr)
	userGroup.GET("/login", login.Get).Name = routeNames.RouteNameLogin
	userGroup.POST("/login", login.Post).Name = routeNames.RouteNameLoginSubmit

	register := controllers.NewRegisterRoute(ctr, *deps.ProfileRepo, *deps.SubscriptionsRepo, deps.NotificationSendPermissionRepo)
	userGroup.GET("/register", register.Get).Name = routeNames.RouteNameRegister
	userGroup.POST("/register", register.Post).Name = routeNames.RouteNameRegisterSubmit

	forgot := controllers.NewForgotPasswordRoute(ctr)
	userGroup.GET("/password", forgot.Get).Name = routeNames.RouteNameForgotPassword
	userGroup.POST("/password", forgot.Post).Name = routeNames.RouteNameForgotPasswordSubmit

	resetGroup := userGroup.Group("/password/reset",
		middleware.LoadUser(c.ORM),
		middleware.LoadValidPasswordToken(c.Auth),
	)
	reset := controllers.NewResetPasswordRoute(ctr)
	resetGroup.GET("/token/:user/:password_token/:token", reset.Get).Name = routeNames.RouteNameResetPassword
	resetGroup.POST("/token/:user/:password_token/:token", reset.Post).Name = routeNames.RouteNameResetPasswordSubmit

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
	pwaPushNotificationsRepo := notifications.NewPwaPushNotificationsRepo(
		c.ORM,
		c.Config.App.VapidPublicKey,
		c.Config.App.VapidPrivateKey,
		c.Config.Mail.FromAddress,
	)

	var firebaseJSONAccessKeys *[]byte
	if len(c.Config.App.FirebaseJSONAccessKeys) > 0 {
		firebaseJSONAccessKeys = &c.Config.App.FirebaseJSONAccessKeys
	}
	fcmPushNotificationsRepo, err := notifications.NewFcmPushNotificationsRepo(c.ORM, firebaseJSONAccessKeys)
	if err != nil {
		return fmt.Errorf("build fcm notifications repo: %w", err)
	}

	region := strings.TrimSpace(c.Config.Phone.Region)
	if region == "" {
		region = "us-east-1"
	}
	smsSenderRepo, err := notifications.NewSMSSender(
		c.ORM,
		region,
		c.Config.Phone.SenderID,
		c.Config.Phone.ValidationCodeExpirationMinutes,
	)
	if err != nil {
		return fmt.Errorf("build sms sender repo: %w", err)
	}

	onboardingGroup := g.Group("/welcome", middleware.RequireAuthentication())
	preferences := controllers.NewPreferencesRoute(
		ctr,
		deps.ProfileRepo,
		pwaPushNotificationsRepo,
		deps.NotificationSendPermissionRepo,
		deps.SubscriptionsRepo,
		smsSenderRepo,
	)
	onboardingGroup.GET("/preferences", preferences.Get).Name = routeNames.RouteNamePreferences
	onboardingGroup.GET("/preferences/phone", preferences.GetPhoneComponent).Name = routeNames.RouteNameGetPhone
	onboardingGroup.GET("/preferences/phone/verification", preferences.GetPhoneVerificationComponent).Name = routeNames.RouteNameGetPhoneVerification
	onboardingGroup.POST("/preferences/phone/verification", preferences.SubmitPhoneVerificationCode).Name = routeNames.RouteNameSubmitPhoneVerification
	onboardingGroup.POST("/preferences/phone/save", preferences.SavePhoneInfo).Name = routeNames.RouteNameUpdatePhoneNum
	onboardingGroup.GET("/preferences/display-name/get", preferences.GetDisplayName).Name = routeNames.RouteNameGetDisplayName
	onboardingGroup.POST("/preferences/display-name/save", preferences.SaveDisplayName).Name = routeNames.RouteNameUpdateDisplayName

	deleteAccountRoute := controllers.NewDeleteAccountRoute(ctr, deps.ProfileRepo, deps.SubscriptionsRepo)
	onboardingGroup.GET("/preferences/delete-account", deleteAccountRoute.DeleteAccountPage).Name = routeNames.RouteNameDeleteAccountPage
	onboardingGroup.GET("/preferences/delete-account/now", deleteAccountRoute.DeleteAccountRequest).Name = routeNames.RouteNameDeleteAccountRequest

	finishOnboarding := controllers.NewOnboardingRoute(ctr, c.ORM)
	onboardingGroup.GET("/finish-onboarding", finishOnboarding.Get).Name = routeNames.RouteNameFinishOnboarding

	profilePrefs := controllers.NewProfilePrefsRoute(ctr, c.ORM)
	onboardingGroup.GET("/profileBio", profilePrefs.GetBio).Name = routeNames.RouteNameGetBio
	onboardingGroup.POST("/profileBio/update", profilePrefs.UpdateBio).Name = routeNames.RouteNameUpdateBio

	outgoingNotifications := controllers.NewPushNotifsRoute(ctr, pwaPushNotificationsRepo, fcmPushNotificationsRepo, deps.NotificationSendPermissionRepo)
	onboardingGroup.GET("/subscription/push", outgoingNotifications.GetPushSubscriptions).Name = routeNames.RouteNameGetPushSubscriptions
	onboardingGroup.POST("/subscription/:platform", outgoingNotifications.RegisterSubscription).Name = routeNames.RouteNameRegisterSubscription
	onboardingGroup.DELETE("/subscription/:platform", outgoingNotifications.DeleteSubscription).Name = routeNames.RouteNameDeleteSubscription
	onboardingGroup.GET("/email-subscription/unsubscribe/:permission/:token", outgoingNotifications.DeleteEmailSubscription).Name = routeNames.RouteNameDeleteEmailSubscriptionWithToken

	allGroup := g.Group("/auth", middleware.RequireAuthentication())
	logout := controllers.NewLogoutRoute(ctr)
	allGroup.GET("/logout", logout.Get, middleware.RequireAuthentication()).Name = routeNames.RouteNameLogout

	onboardedGroup := g.Group("/auth", middleware.RequireAuthentication(), middleware.RedirectToOnboardingIfNotComplete())

	verifyEmail := controllers.NewVerifyEmailRoute(ctr)
	g.GET("/email/verify/:token", verifyEmail.Get).Name = routeNames.RouteNameVerifyEmail

	homeFeed := controllers.NewHomeFeedRoute(ctr, *deps.ProfileRepo, &c.Config.App.PageSize)
	onboardedGroup.GET("/homeFeed", homeFeed.Get, middleware.SetLastSeenOnline(c.Auth)).Name = routeNames.RouteNameHomeFeed
	onboardedGroup.GET("/homeFeed/buttons", homeFeed.GetHomeButtons).Name = routeNames.RouteNameGetHomeFeedButtons

	singleProfile := controllers.NewProfileRoutes(ctr, deps.ProfileRepo)
	onboardedGroup.GET("/profile", singleProfile.Get).Name = routeNames.RouteNameProfile

	uploadPhoto := controllers.NewUploadPhotoRoutes(ctr, deps.ProfileRepo, deps.StorageRepo, c.Config.Storage.PhotosMaxFileSizeMB)
	onboardedGroup.GET("/uploadPhoto", uploadPhoto.Get).Name = routeNames.RouteNameUploadPhoto
	onboardedGroup.POST("/uploadPhoto", uploadPhoto.Post).Name = routeNames.RouteNameUploadPhotoSubmit
	onboardedGroup.DELETE("/uploadPhoto/:image_id", uploadPhoto.Delete).Name = routeNames.RouteNameUploadPhotoDelete

	currProfilePhoto := controllers.NewCurrProfilePhotoRoutes(ctr, deps.ProfileRepo, deps.StorageRepo, c.Config.Storage.PhotosMaxFileSizeMB)
	onboardedGroup.GET("/currProfilePhoto", currProfilePhoto.Get).Name = routeNames.RouteNameCurrentProfilePhoto
	onboardedGroup.POST("/currProfilePhoto", currProfilePhoto.Post).Name = routeNames.RouteNameCurrentProfilePhotoSubmit

	normalNotificationsCount := controllers.NewNormalNotificationsCountRoute(ctr, *deps.ProfileRepo)
	onboardedGroup.GET("/notifications/normalNotificationsCount", normalNotificationsCount.Get).Name = routeNames.RouteNameNormalNotificationsCount

	payments := controllers.NewPaymentsRoute(ctr, c.ORM, deps.SubscriptionsRepo)
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
	payments := controllers.NewPaymentsRoute(ctr, c.ORM, deps.SubscriptionsRepo)
	e.POST(deps.StripeWebhookPath, payments.HandleWebhook).Name = routeNames.RouteNamePaymentProcessorWebhook
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
