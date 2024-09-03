package routes

import (
	"github.com/labstack/echo/v4"

	"github.com/mikestefanello/pagoda/pkg/controller"
	"github.com/mikestefanello/pagoda/templates"
	"github.com/mikestefanello/pagoda/templates/layouts"
	"github.com/mikestefanello/pagoda/templates/pages"
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
