package controllers

import (
	"github.com/leomorpho/goship/apps/site/views"
	"github.com/leomorpho/goship/apps/site/views/web/layouts/gen"
	"github.com/leomorpho/goship/apps/site/views/web/pages/gen"
	"github.com/leomorpho/goship/apps/site/web/ui"
	"github.com/leomorpho/goship/apps/site/web/viewmodels"

	"github.com/labstack/echo/v4"
)

type (
	about struct {
		ctr ui.Controller
	}
)

func NewAboutUsRoute(ctr ui.Controller) about {
	return about{
		ctr: ctr,
	}
}

func (c *about) Get(ctx echo.Context) error {

	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Name = templates.PageAbout
	page.Component = pages.About(&page)
	page.Data = viewmodels.AboutData{
		SupportEmail: c.ctr.Container.Config.App.SupportEmail,
	}
	page.HTMX.Request.Boosted = true

	return c.ctr.RenderPage(ctx, page)
}
