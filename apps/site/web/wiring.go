package web

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	modemailsubscriptions "github.com/leomorpho/goship-modules/emailsubscriptions"
	appemailsubscriptions "github.com/leomorpho/goship/apps/site/app/emailsubscriptions"
	"github.com/leomorpho/goship/apps/site/app/notifications"
	"github.com/leomorpho/goship/apps/site/app/profiles"
	"github.com/leomorpho/goship/apps/site/app/subscriptions"
	"github.com/leomorpho/goship/apps/site/foundation"
	"github.com/leomorpho/goship/apps/site/web/middleware"
	"github.com/leomorpho/goship/config"
	storagerepo "github.com/leomorpho/goship/pkg/repos/storage"
	"github.com/leomorpho/goship/pkg/runtimeplan"
	"github.com/rs/zerolog/log"
	slogecho "github.com/samber/slog-echo"
	"github.com/ziflex/lecho/v3"
)

const (
	defaultStripeWebhookPath = "/Q2HBfAY7iid59J1SUN8h1Y3WxJcPWA/payments/webhooks"
	pathServiceWorker        = "/service-worker.js"
	pathAndroidAssetLinks    = "/.well-known/assetlinks.json"
)

type RouteDeps struct {
	EmailSubscriptions             *modemailsubscriptions.Service
	StorageRepo                    *storagerepo.StorageClient
	ProfileRepo                    *profiles.ProfileRepo
	SubscriptionsRepo              *subscriptions.SubscriptionsRepo
	NotificationSendPermissionRepo *notifications.NotificationSendPermissionRepo
	StripeWebhookPath              string
}

func sseSkipper(c echo.Context) bool {
	// Skip timeout middleware for SSE endpoint.
	return c.Path() == "/auth/realtime"
}

func NewRouteDeps(c *foundation.Container) (*RouteDeps, error) {
	deps := &RouteDeps{}
	deps.EmailSubscriptions = modemailsubscriptions.New(appemailsubscriptions.NewEntStore(c.ORM))
	deps.StorageRepo = storagerepo.NewStorageClient(c.Config, c.ORM)
	deps.SubscriptionsRepo = subscriptions.NewSubscriptionsRepo(
		c.ORM,
		c.Config.App.OperationalConstants.ProTrialTimespanInDays,
		c.Config.App.OperationalConstants.PaymentFailedGracePeriodInDays,
	)
	deps.ProfileRepo = profiles.NewProfileRepo(c.ORM, deps.StorageRepo, deps.SubscriptionsRepo)
	deps.NotificationSendPermissionRepo = notifications.NewNotificationSendPermissionRepo(c.ORM)

	deps.StripeWebhookPath = strings.TrimSpace(c.Config.App.StripeWebhookPath)
	if deps.StripeWebhookPath == "" {
		deps.StripeWebhookPath = defaultStripeWebhookPath
	}

	return deps, nil
}

func RegisterStaticRoutes(c *foundation.Container) {
	// Static files with proper cache control.
	c.Web.Group("", middleware.CacheControl(c.Config.Cache.Expiration.StaticFile), echomw.Gzip()).
		Static(config.StaticPrefix, config.StaticDir)

	// Custom handler for serving the service worker script with specific headers.
	c.Web.GET(pathServiceWorker, func(ctx echo.Context) error {
		ctx.Response().Header().Set(echo.HeaderContentType, "application/javascript")
		ctx.Response().Header().Set("Service-Worker-Allowed", "/")
		ctx.Response().Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", c.Config.Cache.Expiration.StaticFile))
		return ctx.File("./apps/site/static/service-worker.js")
	})

	// Custom handler for serving Android asset links.
	c.Web.GET(pathAndroidAssetLinks, func(ctx echo.Context) error {
		ctx.Response().Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", c.Config.Cache.Expiration.StaticFile))
		return ctx.File("./pwabuilder-android-wrapper/assetlinks.json")
	})
}

func ApplyTLSRedirect(groups ...*echo.Group) {
	for _, group := range groups {
		group.Use(echomw.HTTPSRedirect())
	}
}

func commonMiddleware(c *foundation.Container, deps *RouteDeps, sessionStore *sessions.CookieStore) []echo.MiddlewareFunc {
	return []echo.MiddlewareFunc{
		echomw.RemoveTrailingSlashWithConfig(echomw.TrailingSlashConfig{RedirectCode: http.StatusMovedPermanently}),
		echomw.Recover(),
		echomw.Secure(),
		echomw.RequestID(),
		middleware.LogRequestID(),
		session.Middleware(sessionStore),
		middleware.LoadAuthenticatedUser(c.Auth, deps.ProfileRepo, deps.SubscriptionsRepo),
		echomw.CSRFWithConfig(echomw.CSRFConfig{
			TokenLookup:  "form:csrf,header:X-CSRF-Token,query:csrf",
			CookieMaxAge: 172800, // 48h
		}),
		lecho.Middleware(lecho.Config{Logger: c.Logger}),
	}
}

func ApplyMainMiddleware(c *foundation.Container, g *echo.Group, logger *slog.Logger, deps *RouteDeps, webFeatures runtimeplan.WebFeatures) {
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

func ApplyRealtimeMiddleware(c *foundation.Container, s *echo.Group, deps *RouteDeps) {
	sessionStore := sessions.NewCookieStore([]byte(c.Config.App.EncryptionKey))
	base := commonMiddleware(c, deps, sessionStore)
	mw := []echo.MiddlewareFunc{echomw.Logger()}
	mw = append(mw, base...)
	s.Use(mw...)
}

func ApplyExternalMiddleware(c *foundation.Container, e *echo.Group, deps *RouteDeps) {
	sessionStore := sessions.NewCookieStore([]byte(c.Config.App.EncryptionKey))
	base := commonMiddleware(c, deps, sessionStore)
	mw := []echo.MiddlewareFunc{
		echomw.Gzip(),
		echomw.TimeoutWithConfig(echomw.TimeoutConfig{Skipper: sseSkipper, Timeout: c.Config.App.Timeout}),
	}
	mw = append(mw, base...)
	e.Use(mw...)
}
