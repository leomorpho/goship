package controllers

import (
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/views"
	"github.com/leomorpho/goship/app/views/web/pages/gen"
	"github.com/leomorpho/goship/app/web/ui"
)

type islandsDemo struct {
	ctr ui.Controller
}

func NewIslandsDemoRoute(ctr ui.Controller) islandsDemo {
	return islandsDemo{ctr: ctr}
}

func (r *islandsDemo) Get(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.AppName = string(r.ctr.Container.Config.App.Name)
	page.Name = templates.PageIslandsDemo
	page.Component = pages.IslandsDemoPage(&page)

	return r.ctr.RenderPage(ctx, page)
}
