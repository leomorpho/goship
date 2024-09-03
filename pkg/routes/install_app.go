package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/mikestefanello/pagoda/pkg/controller"
	"github.com/mikestefanello/pagoda/templates"
	"github.com/mikestefanello/pagoda/templates/layouts"
	"github.com/mikestefanello/pagoda/templates/pages"
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
