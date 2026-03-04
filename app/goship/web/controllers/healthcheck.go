package controllers

import (
	"github.com/labstack/echo/v4"

	"github.com/leomorpho/goship/app/goship/views"
	"github.com/leomorpho/goship/app/goship/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/goship/views/web/pages/gen"
	"github.com/leomorpho/goship/app/goship/webui"
)

type (
	healthcheck struct {
		ctr webui.Controller
	}
)

func NewHealthCheckRoute(ctr webui.Controller) healthcheck {
	return healthcheck{
		ctr: ctr,
	}
}

func (g *healthcheck) Get(ctx echo.Context) error {
	page := webui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Name = templates.PageHealthcheck
	page.Component = pages.HealthCheck(&page)
	page.Cache.Enabled = false

	return g.ctr.RenderPage(ctx, page)
}
