package pwa

import (
	"embed"
	"fmt"
	"io/fs"
	"time"

	"github.com/labstack/echo/v4"
)

const (
	pathServiceWorker = "/service-worker.js"
	pathManifest      = "/files/manifest.json"
)

//go:embed static/*
var staticFS embed.FS

type staticRegistrar interface {
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
}

type assetService struct{}

func newAssetService() *assetService {
	return &assetService{}
}

func (m *Module) RegisterStaticRoutes(r staticRegistrar, cacheMaxAge time.Duration) error {
	return m.assets.registerRoutes(r, cacheMaxAge)
}

func (s *assetService) registerRoutes(r staticRegistrar, cacheMaxAge time.Duration) error {
	serviceWorker, err := fs.ReadFile(staticFS, "static/service-worker.js")
	if err != nil {
		return fmt.Errorf("read pwa service worker: %w", err)
	}
	manifest, err := fs.ReadFile(staticFS, "static/manifest.json")
	if err != nil {
		return fmt.Errorf("read pwa manifest: %w", err)
	}

	r.GET(pathServiceWorker, func(ctx echo.Context) error {
		ctx.Response().Header().Set(echo.HeaderContentType, "application/javascript")
		ctx.Response().Header().Set("Service-Worker-Allowed", "/")
		ctx.Response().Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", cacheMaxAge))
		return ctx.Blob(200, "application/javascript", serviceWorker)
	})

	r.GET(pathManifest, func(ctx echo.Context) error {
		ctx.Response().Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", cacheMaxAge))
		return ctx.Blob(200, "application/manifest+json", manifest)
	})

	return nil
}
