package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/goship/views"
	"github.com/leomorpho/goship/app/goship/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/goship/views/web/pages/gen"
	"github.com/leomorpho/goship/pkg/controller"
)

type (
	installApp struct {
		ctr controller.Controller
	}
)

func NewInstallAppRoute(
	ctr controller.Controller,
) installApp {
	return installApp{
		ctr: ctr,
	}
}

func (c *installApp) GetInstallPage(ctx echo.Context) error {
	page := controller.NewPage(ctx)
	page.Layout = layouts.Main
	page.Name = templates.PageInstallApp
	page.Component = pages.InstallApp(&page)
	page.HTMX.Request.Boosted = true

	return c.ctr.RenderPage(ctx, page)
}
