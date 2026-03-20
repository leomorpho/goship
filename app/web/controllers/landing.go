package controllers

import (
	"github.com/leomorpho/goship/app/views"
	"github.com/leomorpho/goship/app/views/web/pages/gen"
	"github.com/leomorpho/goship/app/web/routenames"
	"github.com/leomorpho/goship/app/web/ui"

	"github.com/labstack/echo/v4"
)

type (
	landingPage struct {
		ctr ui.Controller
	}
)

func NewLandingPageRoute(ctr ui.Controller) landingPage {
	return landingPage{
		ctr: ctr,
	}
}

func (c *landingPage) Get(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.AppName = string(c.ctr.Container.Config.App.Name)

	if page.AuthUser != nil {
		return c.ctr.Redirect(ctx, routenames.RouteNameHomeFeed)
	}

	page.Name = templates.PageLanding
	page.Component = pages.LandingPage(&page)

	return c.ctr.RenderPage(ctx, page)
}
