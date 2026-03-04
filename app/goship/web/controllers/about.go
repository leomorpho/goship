package controllers

import (
	"github.com/leomorpho/goship/app/goship/types"
	"github.com/leomorpho/goship/app/goship/views"
	"github.com/leomorpho/goship/app/goship/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/goship/views/web/pages/gen"
	"github.com/leomorpho/goship/app/goship/webui"

	"github.com/labstack/echo/v4"
)

type (
	about struct {
		ctr webui.Controller
	}
)

func NewAboutUsRoute(ctr webui.Controller) about {
	return about{
		ctr: ctr,
	}
}

func (c *about) Get(ctx echo.Context) error {

	page := webui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Name = templates.PageAbout
	page.Component = pages.About(&page)
	page.Data = types.AboutData{
		SupportEmail: c.ctr.Container.Config.App.SupportEmail,
	}
	page.HTMX.Request.Boosted = true

	return c.ctr.RenderPage(ctx, page)
}
