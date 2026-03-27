package web

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"github.com/leomorpho/goship-modules/notifications"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/config/runtimeplan"
	frameworkbootstrap "github.com/leomorpho/goship/framework/bootstrap"
	"github.com/leomorpho/goship/framework/logging"
	frameworkmiddleware "github.com/leomorpho/goship/framework/middleware"
	storagerepo "github.com/leomorpho/goship/framework/storage"
	frameworkcontrollers "github.com/leomorpho/goship/framework/http/controllers"
	webmiddleware "github.com/leomorpho/goship/framework/http/middleware"
	"github.com/leomorpho/goship/framework/http/ui"
	i18nmodule "github.com/leomorpho/goship/modules/i18n"
	slogecho "github.com/samber/slog-echo"
)

const (
	defaultStripeWebhookPath = "/Q2HBfAY7iid59J1SUN8h1Y3WxJcPWA/payments/webhooks"
	pathAndroidAssetLinks    = "/.well-known/assetlinks.json"
)

type RouteDeps struct {
	StorageRepo                   *storagerepo.StorageClient
	SubscriptionsRepo             *paidsubscriptions.Service
	NotifierService               *notifications.NotifierService
	NotificationPermissionService *notifications.NotificationPermissionService
	PwaPushService                *notifications.PwaPushService
	FcmPushService                *notifications.FcmPushService
	SMSSenderService              *notifications.SMSSender
	StripeWebhookPath             string
}

func NewRouteDeps(
	c *frameworkbootstrap.Container,
	paidSubscriptions *paidsubscriptions.Service,
	notificationServices *notifications.Services,
) (*RouteDeps, error) {
	deps := &RouteDeps{}
	deps.StorageRepo = storagerepo.NewStorageClient(c.Config, c.Database, c.Config.Adapters.DB)
	deps.SubscriptionsRepo = paidSubscriptions
	if notificationServices != nil {
		deps.NotifierService = notificationServices.Notifier
		deps.NotificationPermissionService = notificationServices.Permission
		deps.PwaPushService = notificationServices.PwaPush
		deps.FcmPushService = notificationServices.FcmPush
		deps.SMSSenderService = notificationServices.SMSSender
	}

	deps.StripeWebhookPath = strings.TrimSpace(c.Config.App.StripeWebhookPath)
	if deps.StripeWebhookPath == "" {
		deps.StripeWebhookPath = defaultStripeWebhookPath
	}

	return deps, nil
}

func RegisterStaticRoutes(c *frameworkbootstrap.Container) error {
	registerHealthRoutes(c)

	c.Web.Group("", webmiddleware.CacheControl(c.Config.Cache.Expiration.StaticFile), echomw.Gzip()).
		Static(config.StaticPrefix, config.StaticDir)

	if c.Config.Storage.Driver == config.StorageDriverLocal {
		c.Web.Static("/uploads", c.Config.Storage.LocalStoragePath)
	}

	c.Web.GET(pathAndroidAssetLinks, func(ctx echo.Context) error {
		ctx.Response().Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", c.Config.Cache.Expiration.StaticFile))
		return ctx.File("./pwabuilder-android-wrapper/assetlinks.json")
	})

	return nil
}

func registerHealthRoutes(c *frameworkbootstrap.Container) {
	healthcheck := frameworkcontrollers.NewHealthCheckRoute(ui.NewController(c))
	c.Web.GET("/up", healthcheck.GetLiveness).Name = "healthcheck"
	c.Web.GET("/health", healthcheck.GetLiveness).Name = "health.liveness"
	c.Web.GET("/health/ready", healthcheck.GetReadiness).Name = "health.readiness"
}

func ApplyTLSRedirect(groups ...*echo.Group) {
	for _, group := range groups {
		group.Use(echomw.HTTPSRedirect())
	}
}

func commonMiddleware(c *frameworkbootstrap.Container, deps *RouteDeps, sessionStore *sessions.CookieStore) []echo.MiddlewareFunc {
	return []echo.MiddlewareFunc{
		echomw.RemoveTrailingSlashWithConfig(echomw.TrailingSlashConfig{RedirectCode: http.StatusMovedPermanently}),
		echomw.Secure(),
		frameworkmiddleware.RequestID(),
		session.Middleware(sessionStore),
		webmiddleware.LoadAuthenticatedUser(c.Auth),
		i18nmodule.DetectLanguage(c.I18n, nil),
		echomw.CSRFWithConfig(echomw.CSRFConfig{
			TokenLookup:  "form:csrf,header:X-CSRF-Token,query:csrf",
			CookieMaxAge: 172800,
		}),
	}
}

func sentryEnabled(c *frameworkbootstrap.Container) bool {
	return c != nil && c.Config != nil && strings.TrimSpace(c.Config.App.SentryDsn) != ""
}

func ApplyMainMiddleware(c *frameworkbootstrap.Container, g *echo.Group, logger *slog.Logger, deps *RouteDeps, webFeatures runtimeplan.WebFeatures) {
	sessionStore := sessions.NewCookieStore([]byte(c.Config.App.EncryptionKey))
	base := commonMiddleware(c, deps, sessionStore)

	mw := []echo.MiddlewareFunc{
		webmiddleware.FilterSentryErrors,
		webmiddleware.RecoverPanics(c.Logger),
		echomw.Gzip(),
		slogecho.New(logger),
		frameworkmiddleware.SecurityHeaders(c.Config.Security, string(c.Config.App.Environment)),
		echomw.TimeoutWithConfig(echomw.TimeoutConfig{Timeout: c.Config.App.Timeout}),
	}
	if !sentryEnabled(c) {
		mw = mw[1:]
	}
	mw = append(mw, base...)
	mw = append(mw, webmiddleware.SetDeviceTypeToServe())
	g.Use(mw...)

	if webFeatures.EnablePageCache {
		g.Use(webmiddleware.ServeCachedPage(c.Cache))
	} else {
		logging.FromContext(context.Background()).Info("page cache middleware disabled (cache dependency unavailable or web process disabled)")
	}
}

func ApplyExternalMiddleware(c *frameworkbootstrap.Container, e *echo.Group, logger *slog.Logger, deps *RouteDeps) {
	sessionStore := sessions.NewCookieStore([]byte(c.Config.App.EncryptionKey))
	base := commonMiddleware(c, deps, sessionStore)
	mw := []echo.MiddlewareFunc{
		webmiddleware.FilterSentryErrors,
		webmiddleware.RecoverPanics(c.Logger),
		echomw.Gzip(),
		slogecho.New(logger),
		frameworkmiddleware.SecurityHeaders(c.Config.Security, string(c.Config.App.Environment)),
		echomw.TimeoutWithConfig(echomw.TimeoutConfig{Timeout: c.Config.App.Timeout}),
	}
	if !sentryEnabled(c) {
		mw = mw[1:]
	}
	mw = append(mw, base...)
	e.Use(mw...)
}
