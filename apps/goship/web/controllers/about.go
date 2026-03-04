package controllers

import (
	"github.com/leomorpho/goship/apps/goship/web/viewmodels"
	"github.com/leomorpho/goship/apps/goship/views"
	"github.com/leomorpho/goship/apps/goship/views/web/layouts/gen"
	"github.com/leomorpho/goship/apps/goship/views/web/pages/gen"
	"github.com/leomorpho/goship/apps/goship/web/ui"

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
