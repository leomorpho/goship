package goship

import (
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/foundation"
	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/web/ui"
)

func TestRegisterMailPreviewRoutesDevelopmentOnly(t *testing.T) {
	t.Run("registers routes in development", func(t *testing.T) {
		e := echo.New()
		g := e.Group("")
		ctr := ui.NewController(&foundation.Container{Config: testConfigForDevelop()})

		registerMailPreviewRoutes(g, ctr)

		if countRoutesWithPrefix(e.Routes(), "/dev/mail") == 0 {
			t.Fatal("expected /dev/mail routes in development")
		}
	})

	t.Run("does not register routes in production", func(t *testing.T) {
		e := echo.New()
		g := e.Group("")
		ctr := ui.NewController(&foundation.Container{Config: testConfigForProduction()})

		registerMailPreviewRoutes(g, ctr)

		if countRoutesWithPrefix(e.Routes(), "/dev/mail") != 0 {
			t.Fatal("did not expect /dev/mail routes in production")
		}
	})
}

func testConfigForDevelop() *config.Config {
	return &config.Config{
		App: config.AppConfig{
			Name:         "GoShip",
			SupportEmail: "support@example.com",
			Environment:  config.EnvDevelop,
		},
		HTTP: config.HTTPConfig{
			Domain: "https://example.test",
		},
	}
}

func testConfigForProduction() *config.Config {
	cfg := testConfigForDevelop()
	cfg.App.Environment = config.EnvProduction
	return cfg
}

func countRoutesWithPrefix(routes []*echo.Route, prefix string) int {
	count := 0
	for _, route := range routes {
		if strings.HasPrefix(route.Path, prefix) {
			count++
		}
	}
	return count
}
