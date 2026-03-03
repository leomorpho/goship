package routes

import (
	"github.com/labstack/echo/v4"

	"github.com/leomorpho/goship/app/goship/controller"
	"github.com/leomorpho/goship/app/goship/views"
	"github.com/leomorpho/goship/app/goship/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/goship/views/web/pages/gen"
)

type (
	healthcheck struct {
		ctr controller.Controller
	}
)

func NewHealthCheckRoute(ctr controller.Controller) healthcheck {
	return healthcheck{
		ctr: ctr,
	}
}

func (g *healthcheck) Get(ctx echo.Context) error {
	page := controller.NewPage(ctx)
	page.Layout = layouts.Main
	page.Name = templates.PageHealthcheck
	page.Component = pages.HealthCheck(&page)
	page.Cache.Enabled = false

	return g.ctr.RenderPage(ctx, page)
}
