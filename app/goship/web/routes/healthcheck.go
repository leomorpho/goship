package routes

import (
	"github.com/labstack/echo/v4"

	"github.com/mikestefanello/pagoda/app/goship/views"
	"github.com/mikestefanello/pagoda/app/goship/views/layouts"
	"github.com/mikestefanello/pagoda/app/goship/views/pages"
	"github.com/mikestefanello/pagoda/pkg/controller"
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
