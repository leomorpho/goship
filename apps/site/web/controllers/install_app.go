package controllers

import (
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/apps/site/views"
	"github.com/leomorpho/goship/apps/site/views/web/layouts/gen"
	"github.com/leomorpho/goship/apps/site/views/web/pages/gen"
	"github.com/leomorpho/goship/apps/site/web/ui"
)

type (
	installApp struct {
		ctr ui.Controller
	}
)

func NewInstallAppRoute(
	ctr ui.Controller,
) installApp {
	return installApp{
		ctr: ctr,
	}
}

func (c *installApp) GetInstallPage(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Name = templates.PageInstallApp
	page.Component = pages.InstallApp(&page)
	page.HTMX.Request.Boosted = true

	return c.ctr.RenderPage(ctx, page)
}
