package routes

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/pkg/controller"
	"github.com/leomorpho/goship/pkg/middleware"
	"github.com/leomorpho/goship/pkg/repos/emailsmanager"
	"github.com/leomorpho/goship/pkg/repos/notifierrepo"
	"github.com/leomorpho/goship/pkg/repos/profilerepo"
	storagerepo "github.com/leomorpho/goship/pkg/repos/storage"
	"github.com/leomorpho/goship/pkg/repos/subscriptions"
	routeNames "github.com/leomorpho/goship/pkg/routing/routenames"
	"github.com/leomorpho/goship/pkg/runtimeplan"
	"github.com/leomorpho/goship/pkg/services"
	slogecho "github.com/samber/slog-echo"
	"github.com/ziflex/lecho/v3"
)

const (
	defaultStripeWebhookPath = "/Q2HBfAY7iid59J1SUN8h1Y3WxJcPWA/payments/webhooks"
	pathServiceWorker        = "/service-worker.js"
	pathAndroidAssetLinks    = "/.well-known/assetlinks.json"
)

type routeDeps struct {
	emailSubscriptionRepo          *emailsmanager.EmailSubscriptionRepo
	storageRepo                    *storagerepo.StorageClient
	profileRepo                    *profilerepo.ProfileRepo
	subscriptionsRepo              *subscriptions.SubscriptionsRepo
	notificationSendPermissionRepo *notifierrepo.NotificationSendPermissionRepo
	stripeWebhookPath              string
}

func sseSkipper(c echo.Context) bool {
	// Skip timeout middleware for SSE endpoint.
	return c.Path() == "/auth/realtime"
}

func newRouteDeps(c *services.Container) (*routeDeps, error) {
	deps := &routeDeps{}
	deps.emailSubscriptionRepo = emailsmanager.NewEmailSubscriptionRepo(c.ORM)
	deps.storageRepo = storagerepo.NewStorageClient(c.Config, c.ORM)
	deps.subscriptionsRepo = subscriptions.NewSubscriptionsRepo(
		c.ORM,
		c.Config.App.OperationalConstants.ProTrialTimespanInDays,
		c.Config.App.OperationalConstants.PaymentFailedGracePeriodInDays,
	)
	deps.profileRepo = profilerepo.NewProfileRepo(c.ORM, deps.storageRepo, deps.subscriptionsRepo)
	deps.notificationSendPermissionRepo = notifierrepo.NewNotificationSendPermissionRepo(c.ORM)

	deps.stripeWebhookPath = strings.TrimSpace(c.Config.App.StripeWebhookPath)
	if deps.stripeWebhookPath == "" {
		deps.stripeWebhookPath = defaultStripeWebhookPath
	}

	return deps, nil
}

// BuildRouter builds the router.
func BuildRouter(c *services.Container) error {
	deps, err := newRouteDeps(c)
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

	registerStaticRoutes(c)

	// Non static file route groups.
	g := c.Web.Group("")
	e := c.Web.Group("")
	s := c.Web.Group("")

	if c.Config.HTTP.TLS.Enabled {
		applyTLSRedirect(g, e, s)
	}

	applyMainMiddleware(c, g, logger, deps, webFeatures)
	applyRealtimeMiddleware(c, s, deps)
	applyExternalMiddleware(c, e, deps)

	ctr := controller.NewController(c)
	errorHandler := NewErrorHandler(ctr)
	c.Web.HTTPErrorHandler = errorHandler.Get

	if err := registerAppRoutes(c, g, e, s, ctr, webFeatures, deps); err != nil {
		return err
	}

	return nil
}

func registerStaticRoutes(c *services.Container) {
	// Static files with proper cache control.
	c.Web.Group("", middleware.CacheControl(c.Config.Cache.Expiration.StaticFile), echomw.Gzip()).
		Static(config.StaticPrefix, config.StaticDir)

	// Custom handler for serving the service worker script with specific headers.
	c.Web.GET(pathServiceWorker, func(ctx echo.Context) error {
		ctx.Response().Header().Set(echo.HeaderContentType, "application/javascript")
		ctx.Response().Header().Set("Service-Worker-Allowed", "/")
		ctx.Response().Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", c.Config.Cache.Expiration.StaticFile))
		return ctx.File("./service-worker.js")
	})

	// Custom handler for serving Android asset links.
	c.Web.GET(pathAndroidAssetLinks, func(ctx echo.Context) error {
		ctx.Response().Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", c.Config.Cache.Expiration.StaticFile))
		return ctx.File("./pwabuilder-android-wrapper/assetlinks.json")
	})
}

func applyTLSRedirect(groups ...*echo.Group) {
	for _, group := range groups {
		group.Use(echomw.HTTPSRedirect())
	}
}

func commonMiddleware(c *services.Container, deps *routeDeps, sessionStore *sessions.CookieStore) []echo.MiddlewareFunc {
	return []echo.MiddlewareFunc{
		echomw.RemoveTrailingSlashWithConfig(echomw.TrailingSlashConfig{RedirectCode: http.StatusMovedPermanently}),
		echomw.Recover(),
		echomw.Secure(),
		echomw.RequestID(),
		middleware.LogRequestID(),
		session.Middleware(sessionStore),
		middleware.LoadAuthenticatedUser(c.Auth, deps.profileRepo, deps.subscriptionsRepo),
		echomw.CSRFWithConfig(echomw.CSRFConfig{
			TokenLookup:  "form:csrf,header:X-CSRF-Token,query:csrf",
			CookieMaxAge: 172800, // 48h
		}),
		lecho.Middleware(lecho.Config{Logger: c.Logger}),
	}
}

func applyMainMiddleware(c *services.Container, g *echo.Group, logger *slog.Logger, deps *routeDeps, webFeatures runtimeplan.WebFeatures) {
	sessionStore := sessions.NewCookieStore([]byte(c.Config.App.EncryptionKey))
	base := commonMiddleware(c, deps, sessionStore)

	mw := []echo.MiddlewareFunc{
		echomw.Gzip(),
		slogecho.New(logger),
		echomw.TimeoutWithConfig(echomw.TimeoutConfig{Skipper: sseSkipper, Timeout: c.Config.App.Timeout}),
	}
	mw = append(mw, base...)
	mw = append(mw, middleware.SetDeviceTypeToServe())
	g.Use(mw...)

	if webFeatures.EnablePageCache {
		g.Use(middleware.ServeCachedPage(c.Cache))
	} else {
		log.Info().Msg("page cache middleware disabled (cache dependency unavailable or web process disabled)")
	}
}

func applyRealtimeMiddleware(c *services.Container, s *echo.Group, deps *routeDeps) {
	sessionStore := sessions.NewCookieStore([]byte(c.Config.App.EncryptionKey))
	base := commonMiddleware(c, deps, sessionStore)
	mw := []echo.MiddlewareFunc{echomw.Logger()}
	mw = append(mw, base...)
	s.Use(mw...)
}

func applyExternalMiddleware(c *services.Container, e *echo.Group, deps *routeDeps) {
	sessionStore := sessions.NewCookieStore([]byte(c.Config.App.EncryptionKey))
	base := commonMiddleware(c, deps, sessionStore)
	mw := []echo.MiddlewareFunc{
		echomw.Gzip(),
		echomw.TimeoutWithConfig(echomw.TimeoutConfig{Skipper: sseSkipper, Timeout: c.Config.App.Timeout}),
	}
	mw = append(mw, base...)
	e.Use(mw...)
}

// registerAppRoutes keeps app-level route composition centralized while still
// using domain blocks for readability.
func registerAppRoutes(
	c *services.Container,
	g, e, s *echo.Group,
	ctr controller.Controller,
	webFeatures runtimeplan.WebFeatures,
	deps *routeDeps,
) error {
	registerPublicDomainRoutes(c, g, ctr, deps)
	registerDocsDomainRoutes(g, ctr)

	if c.Config.App.OperationalConstants.UserSignupEnabled {
		if err := registerAuthDomainRoutes(c, g, ctr, deps); err != nil {
			return err
		}
		registerExternalDomainRoutes(c, e, ctr, deps)
		if webFeatures.EnableRealtime {
			if err := registerRealtimeDomainRoutes(c, s, ctr); err != nil {
				return err
			}
		} else {
			log.Info().Msg("realtime SSE routes disabled (notifier/pubsub dependency unavailable)")
		}
	}

	return nil
}

// ==================== Public Domain ====================

func registerPublicDomainRoutes(c *services.Container, g *echo.Group, ctr controller.Controller, deps *routeDeps) {
	landingPage := NewLandingPageRoute(ctr)
	g.GET("/", landingPage.Get).Name = routeNames.RouteNameLandingPage

	clearCookie := NewClearCookiesRoute(ctr)
	g.GET("/clear-cookie", clearCookie.Get).Name = routeNames.RouteNameClearCookie

	healthcheck := NewHealthCheckRoute(ctr)
	g.GET("/up", healthcheck.Get).Name = routeNames.RouteNameHealthcheck

	// TODO: remove once sentry is stable.
	g.GET(c.Config.App.TestSentryUrl, func(ctx echo.Context) error {
		panic("Test error for Sentry")
	})

	emailSubscribe := NewEmailSubscribeRoute(ctr, *deps.emailSubscriptionRepo, *c.Config)
	g.GET("/emailSubscribe", emailSubscribe.Get).Name = routeNames.RouteNameEmailSubscribe
	g.POST("/emailSubscribe", emailSubscribe.Post).Name = routeNames.RouteNameEmailSubscribeSubmit

	verifyEmailSubscription := NewVerifyEmailSubscriptionRoute(ctr, *deps.emailSubscriptionRepo)
	g.GET("/email/subscription/:token", verifyEmailSubscription.Get).Name = routeNames.RouteNameVerifyEmailSubscription

	installApp := NewInstallAppRoute(ctr)
	g.GET("/install-app", installApp.GetInstallPage).Name = routeNames.RouteNameInstallApp

	about := NewAboutUsRoute(ctr)
	g.GET("/about", about.Get).Name = routeNames.RouteNameAboutUs

	privacyPolicy := NewPrivacyPolicyRoute(ctr)
	g.GET("/privacy-policy", privacyPolicy.Get).Name = routeNames.RouteNamePrivacyPolicy

	userGroup := g.Group("/user", middleware.RequireNoAuthentication())

	login := NewLoginRoute(ctr)
	userGroup.GET("/login", login.Get).Name = routeNames.RouteNameLogin
	userGroup.POST("/login", login.Post).Name = routeNames.RouteNameLoginSubmit

	register := NewRegisterRoute(ctr, *deps.profileRepo, *deps.subscriptionsRepo, deps.notificationSendPermissionRepo)
	userGroup.GET("/register", register.Get).Name = routeNames.RouteNameRegister
	userGroup.POST("/register", register.Post).Name = routeNames.RouteNameRegisterSubmit

	forgot := NewForgotPasswordRoute(ctr)
	userGroup.GET("/password", forgot.Get).Name = routeNames.RouteNameForgotPassword
	userGroup.POST("/password", forgot.Post).Name = routeNames.RouteNameForgotPasswordSubmit

	resetGroup := userGroup.Group("/password/reset",
		middleware.LoadUser(c.ORM),
		middleware.LoadValidPasswordToken(c.Auth),
	)
	reset := NewResetPasswordRoute(ctr)
	resetGroup.GET("/token/:user/:password_token/:token", reset.Get).Name = routeNames.RouteNameResetPassword
	resetGroup.POST("/token/:user/:password_token/:token", reset.Post).Name = routeNames.RouteNameResetPasswordSubmit

	if ctr.Container.Config.App.Environment != config.EnvProduction {
		err := NewErrorHandler(ctr)
		g.GET("/error/400", err.GetHttp400BadRequest)
		g.GET("/error/401", err.GetHttp401Unauthorized)
		g.GET("/error/403", err.GetHttp403Forbidden)
		g.GET("/error/404", err.GetHttp404NotFound)
		g.GET("/error/500", err.GetHttp500InternalServerError)
	}
}

// ==================== Docs Domain ====================

func registerDocsDomainRoutes(g *echo.Group, ctr controller.Controller) {
	docsRoute := NewDocsRoute(ctr)
	g.GET("/docs", docsRoute.GetDocsHome).Name = routeNames.RouteNameDocs
	g.GET("/docs/gettingStarted", docsRoute.GetDocsGettingStarted).Name = routeNames.RouteNameDocsGettingStarted
	g.GET("/docs/guidedTour", docsRoute.GetDocsGuidedTour).Name = routeNames.RouteNameDocsGuidedTour
	g.GET("/docs/architecture", docsRoute.GetDocsArchitecture).Name = routeNames.RouteNameDocsArchitecture
}

// ==================== Auth Domain ====================

func registerAuthDomainRoutes(c *services.Container, g *echo.Group, ctr controller.Controller, deps *routeDeps) error {
	pwaPushNotificationsRepo := notifierrepo.NewPwaPushNotificationsRepo(
		c.ORM,
		c.Config.App.VapidPublicKey,
		c.Config.App.VapidPrivateKey,
		c.Config.Mail.FromAddress,
	)

	var firebaseJSONAccessKeys *[]byte
	if len(c.Config.App.FirebaseJSONAccessKeys) > 0 {
		firebaseJSONAccessKeys = &c.Config.App.FirebaseJSONAccessKeys
	}
	fcmPushNotificationsRepo, err := notifierrepo.NewFcmPushNotificationsRepo(c.ORM, firebaseJSONAccessKeys)
	if err != nil {
		return fmt.Errorf("build fcm notifications repo: %w", err)
	}

	region := strings.TrimSpace(c.Config.Phone.Region)
	if region == "" {
		region = "us-east-1"
	}
	smsSenderRepo, err := notifierrepo.NewSMSSender(
		c.ORM,
		region,
		c.Config.Phone.SenderID,
		c.Config.Phone.ValidationCodeExpirationMinutes,
	)
	if err != nil {
		return fmt.Errorf("build sms sender repo: %w", err)
	}

	onboardingGroup := g.Group("/welcome", middleware.RequireAuthentication())
	preferences := NewPreferencesRoute(
		ctr,
		deps.profileRepo,
		pwaPushNotificationsRepo,
		deps.notificationSendPermissionRepo,
		deps.subscriptionsRepo,
		smsSenderRepo,
	)
	onboardingGroup.GET("/preferences", preferences.Get).Name = routeNames.RouteNamePreferences
	onboardingGroup.GET("/preferences/phone", preferences.GetPhoneComponent).Name = routeNames.RouteNameGetPhone
	onboardingGroup.GET("/preferences/phone/verification", preferences.GetPhoneVerificationComponent).Name = routeNames.RouteNameGetPhoneVerification
	onboardingGroup.POST("/preferences/phone/verification", preferences.SubmitPhoneVerificationCode).Name = routeNames.RouteNameSubmitPhoneVerification
	onboardingGroup.POST("/preferences/phone/save", preferences.SavePhoneInfo).Name = routeNames.RouteNameUpdatePhoneNum
	onboardingGroup.GET("/preferences/display-name/get", preferences.GetDisplayName).Name = routeNames.RouteNameGetDisplayName
	onboardingGroup.POST("/preferences/display-name/save", preferences.SaveDisplayName).Name = routeNames.RouteNameUpdateDisplayName

	deleteAccountRoute := NewDeleteAccountRoute(ctr, deps.profileRepo, deps.subscriptionsRepo)
	onboardingGroup.GET("/preferences/delete-account", deleteAccountRoute.DeleteAccountPage).Name = routeNames.RouteNameDeleteAccountPage
	onboardingGroup.GET("/preferences/delete-account/now", deleteAccountRoute.DeleteAccountRequest).Name = routeNames.RouteNameDeleteAccountRequest

	finishOnboarding := NewOnboardingRoute(ctr, c.ORM, c.Tasks)
	onboardingGroup.GET("/finish-onboarding", finishOnboarding.Get).Name = routeNames.RouteNameFinishOnboarding

	profilePrefs := NewProfilePrefsRoute(ctr, c.ORM)
	onboardingGroup.GET("/profileBio", profilePrefs.GetBio).Name = routeNames.RouteNameGetBio
	onboardingGroup.POST("/profileBio/update", profilePrefs.UpdateBio).Name = routeNames.RouteNameUpdateBio

	outgoingNotifications := NewPushNotifsRoute(ctr, pwaPushNotificationsRepo, fcmPushNotificationsRepo, deps.notificationSendPermissionRepo)
	onboardingGroup.GET("/subscription/push", outgoingNotifications.GetPushSubscriptions).Name = routeNames.RouteNameGetPushSubscriptions
	onboardingGroup.POST("/subscription/:platform", outgoingNotifications.RegisterSubscription).Name = routeNames.RouteNameRegisterSubscription
	onboardingGroup.DELETE("/subscription/:platform", outgoingNotifications.DeleteSubscription).Name = routeNames.RouteNameDeleteSubscription
	onboardingGroup.GET("/email-subscription/unsubscribe/:permission/:token", outgoingNotifications.DeleteEmailSubscription).Name = routeNames.RouteNameDeleteEmailSubscriptionWithToken

	allGroup := g.Group("/auth", middleware.RequireAuthentication())
	logout := NewLogoutRoute(ctr)
	allGroup.GET("/logout", logout.Get, middleware.RequireAuthentication()).Name = routeNames.RouteNameLogout

	onboardedGroup := g.Group("/auth", middleware.RequireAuthentication(), middleware.RedirectToOnboardingIfNotComplete())

	verifyEmail := NewVerifyEmailRoute(ctr)
	g.GET("/email/verify/:token", verifyEmail.Get).Name = routeNames.RouteNameVerifyEmail

	homeFeed := NewHomeFeedRoute(ctr, *deps.profileRepo, &c.Config.App.PageSize)
	onboardedGroup.GET("/homeFeed", homeFeed.Get, middleware.SetLastSeenOnline(c.Auth)).Name = routeNames.RouteNameHomeFeed
	onboardedGroup.GET("/homeFeed/buttons", homeFeed.GetHomeButtons).Name = routeNames.RouteNameGetHomeFeedButtons

	singleProfile := NewProfileRoutes(ctr, deps.profileRepo)
	onboardedGroup.GET("/profile", singleProfile.Get).Name = routeNames.RouteNameProfile

	uploadPhoto := NewUploadPhotoRoutes(ctr, deps.profileRepo, deps.storageRepo, c.Config.Storage.PhotosMaxFileSizeMB)
	onboardedGroup.GET("/uploadPhoto", uploadPhoto.Get).Name = routeNames.RouteNameUploadPhoto
	onboardedGroup.POST("/uploadPhoto", uploadPhoto.Post).Name = routeNames.RouteNameUploadPhotoSubmit
	onboardedGroup.DELETE("/uploadPhoto/:image_id", uploadPhoto.Delete).Name = routeNames.RouteNameUploadPhotoDelete

	currProfilePhoto := NewCurrProfilePhotoRoutes(ctr, deps.profileRepo, deps.storageRepo, c.Config.Storage.PhotosMaxFileSizeMB)
	onboardedGroup.GET("/currProfilePhoto", currProfilePhoto.Get).Name = routeNames.RouteNameCurrentProfilePhoto
	onboardedGroup.POST("/currProfilePhoto", currProfilePhoto.Post).Name = routeNames.RouteNameCurrentProfilePhotoSubmit

	normalNotificationsCount := NewNormalNotificationsCountRoute(ctr, *deps.profileRepo)
	onboardedGroup.GET("/notifications/normalNotificationsCount", normalNotificationsCount.Get).Name = routeNames.RouteNameNormalNotificationsCount

	payments := NewPaymentsRoute(ctr, c.ORM, deps.subscriptionsRepo)
	onboardedGroup.GET("/payments/get-public-key", payments.GetPaymentProcessorPublickey).Name = routeNames.RouteNamePaymentProcessorGetPublicKey
	onboardedGroup.POST("/payments/create-checkout-session", payments.CreateCheckoutSession).Name = routeNames.RouteNameCreateCheckoutSession
	onboardedGroup.POST("/payments/create-portal-session", payments.CreatePortalSession).Name = routeNames.RouteNameCreatePortalSession
	onboardedGroup.GET("/payments/pricing", payments.PricingPage).Name = routeNames.RouteNamePricingPage
	onboardedGroup.GET("/payments/success", payments.SuccessfullySubscribed).Name = routeNames.RouteNamePaymentProcessorSuccess

	return nil
}

// ==================== External Domain ====================

func registerExternalDomainRoutes(c *services.Container, g *echo.Group, ctr controller.Controller, deps *routeDeps) {
	payments := NewPaymentsRoute(ctr, c.ORM, deps.subscriptionsRepo)
	g.POST(deps.stripeWebhookPath, payments.HandleWebhook).Name = routeNames.RouteNamePaymentProcessorWebhook
}

// ==================== Realtime Domain ====================

// registerRealtimeDomainRoutes wires SSE routes because they have no read timeout set.
func registerRealtimeDomainRoutes(c *services.Container, g *echo.Group, ctr controller.Controller) error {
	if c.Notifier == nil {
		return errors.New("cannot register realtime routes: notifier is nil")
	}

	onboardedGroup := g.Group("/auth", middleware.RequireAuthentication())
	realtime := NewRealtimeRoute(ctr, *c.Notifier)
	onboardedGroup.GET("/realtime", realtime.Get).Name = routeNames.RouteNameRealtime
	return nil
}
