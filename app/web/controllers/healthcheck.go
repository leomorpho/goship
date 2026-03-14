package controllers

import (
	"net/http"
	"runtime/debug"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/framework/health"
)

type (
	healthcheck struct {
		startedAt time.Time
		version   string
		registry  *health.Registry
	}
)

var processStartedAt = time.Now()

func NewHealthCheckRoute(ctr ui.Controller) healthcheck {
	registry := health.NewRegistry()
	if ctr.Container != nil {
		if ctr.Container.Database != nil {
			registry.Register(health.NewDBChecker(ctr.Container.Database, 2*time.Second))
		}
		if ctr.Container.CoreCache != nil {
			registry.Register(health.NewCacheChecker(ctr.Container.CoreCache, 2*time.Second))
		}
		if ctr.Container.CoreJobsInspector != nil {
			registry.Register(health.NewJobsChecker(ctr.Container.CoreJobsInspector, 2*time.Second))
		}
	}

	return healthcheck{
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

func (g *healthcheck) GetLiveness(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, healthResponse{Status: health.StatusOK})
}

func (g *healthcheck) GetReadiness(ctx echo.Context) error {
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
