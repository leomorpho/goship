package controllers

import (
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/goship/views"
	"github.com/leomorpho/goship/app/goship/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/goship/views/web/pages/gen"
	"github.com/leomorpho/goship/app/goship/webui"
)

type (
	installApp struct {
		ctr webui.Controller
	}
)

func NewInstallAppRoute(
	ctr webui.Controller,
) installApp {
	return installApp{
		ctr: ctr,
	}
}

func (c *installApp) GetInstallPage(ctx echo.Context) error {
	page := webui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Name = templates.PageInstallApp
	page.Component = pages.InstallApp(&page)
	page.HTMX.Request.Boosted = true

	return c.ctr.RenderPage(ctx, page)
}
