package controllers

import (
	"github.com/leomorpho/goship/app/views"
	"github.com/leomorpho/goship/app/views/web/layouts/gen"
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
	page.Layout = layouts.LandingPage

	if page.AuthUser != nil {
		return c.ctr.Redirect(ctx, routenames.RouteNameHomeFeed)
	}

	page.Metatags.Description = "Opinionated Go + HTMX framework for shipping production apps fast."
	page.Metatags.Keywords = []string{"Go", "HTMX", "Templ", "Starter", "Framework", "SaaS", "CLI", "LLM"}
	page.Name = templates.PageLanding
	page.Component = pages.LandingPage(&page)

	return c.ctr.RenderPage(ctx, page)
}
