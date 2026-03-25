package controllers

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/framework/health"
	"github.com/leomorpho/goship/framework/web/ui"
)

type HealthcheckRoute struct {
	startedAt time.Time
	version   string
	registry  *health.Registry
}

var processStartedAt = time.Now()

func NewHealthCheckRoute(ctr ui.Controller) HealthcheckRoute {
	if ctr.Container == nil || ctr.Container.Health == nil {
		panic("startup configuration failure: health registry is not configured")
	}
	registry := ctr.Container.Health
	if err := registry.ValidateStartupContract(); err != nil {
		panic(fmt.Sprintf("startup configuration failure: %v", err))
	}

	return HealthcheckRoute{
		startedAt: processStartedAt,
		version:   buildVersion(),
		registry:  registry,
	}
}

type healthResponse struct {
	Status  string                        `json:"status"`
	Version string                        `json:"version,omitempty"`
	Uptime  string                        `json:"uptime,omitempty"`
	Checks  map[string]health.CheckResult `json:"checks,omitempty"`
}

func (g *HealthcheckRoute) GetLiveness(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, healthResponse{Status: health.StatusOK})
}

func (g *HealthcheckRoute) GetReadiness(ctx echo.Context) error {
	results, allOK := g.registry.Run(ctx.Request().Context())

	statusCode := http.StatusOK
	status := health.StatusOK
	if !allOK {
		statusCode = http.StatusServiceUnavailable
		status = health.StatusError
	}

	return ctx.JSON(statusCode, healthResponse{
		Status:  status,
		Version: g.version,
		Uptime:  time.Since(g.startedAt).Round(time.Second).String(),
		Checks:  results,
	})
}

func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" || info.Main.Version == "(devel)" {
		return "dev"
	}
	return info.Main.Version
}
