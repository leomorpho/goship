package routes

import (
	"github.com/mikestefanello/pagoda/pkg/controller"
	"github.com/mikestefanello/pagoda/pkg/types"
	"github.com/mikestefanello/pagoda/templates"
	"github.com/mikestefanello/pagoda/templates/layouts"
	"github.com/mikestefanello/pagoda/templates/pages"

	"github.com/labstack/echo/v4"
)

type (
	about struct {
		ctr controller.Controller
	}
)

func NewAboutUsRoute(ctr controller.Controller) about {
	return about{
		ctr: ctr,
	}
}

func (c *about) Get(ctx echo.Context) error {

	page := controller.NewPage(ctx)
	page.Layout = layouts.Main
	page.Name = templates.PageAbout
	page.Component = pages.About(&page)
	page.Data = types.AboutData{
		SupportEmail: c.ctr.Container.Config.App.SupportEmail,
	}
	page.HTMX.Request.Boosted = true

	return c.ctr.RenderPage(ctx, page)
}
