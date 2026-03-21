package controllers

import (
	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/framework/web/pages/gen"
	"github.com/leomorpho/goship/framework/web/routenames"
	"github.com/leomorpho/goship/framework/web/templates"
	"github.com/leomorpho/goship/framework/web/ui"
)

type LandingPageRoute struct {
	Controller ui.Controller
	Component  func(page *ui.Page) templ.Component
}

func NewLandingPageRoute(ctr ui.Controller) LandingPageRoute {
	return LandingPageRoute{
		Controller: ctr,
		Component:  pages.LandingPage,
	}
}

func (c *LandingPageRoute) Get(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.AppName = string(c.Controller.Container.Config.App.Name)

	if page.AuthUser != nil {
		return c.Controller.Redirect(ctx, routenames.RouteNameHomeFeed)
	}

	page.Name = templates.PageLanding
	page.Component = c.Component(&page)

	return c.Controller.RenderPage(ctx, page)
}
