package controllers

import (
	"github.com/labstack/echo/v4"

	"github.com/leomorpho/goship/apps/site/views"
	"github.com/leomorpho/goship/apps/site/views/web/layouts/gen"
	"github.com/leomorpho/goship/apps/site/views/web/pages/gen"
	"github.com/leomorpho/goship/apps/site/web/ui"
)

type (
	healthcheck struct {
		ctr ui.Controller
	}
)

func NewHealthCheckRoute(ctr ui.Controller) healthcheck {
	return healthcheck{
		ctr: ctr,
	}
}

func (g *healthcheck) Get(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Name = templates.PageHealthcheck
	page.Component = pages.HealthCheck(&page)
	page.Cache.Enabled = false

	return g.ctr.RenderPage(ctx, page)
}
