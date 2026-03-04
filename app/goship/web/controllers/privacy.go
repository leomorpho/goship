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
	privacyPolicy struct {
		ctr webui.Controller
	}
)

func NewPrivacyPolicyRoute(ctr webui.Controller) privacyPolicy {
	return privacyPolicy{
		ctr: ctr,
	}
}

func (c *privacyPolicy) Get(ctx echo.Context) error {

	page := webui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Name = templates.PagePrivacyPolicy
	page.Component = pages.PrivacyPolicy(&page)
	page.Data = types.AboutData{
		SupportEmail: c.ctr.Container.Config.App.SupportEmail,
	}

	page.HTMX.Request.Boosted = true

	return c.ctr.RenderPage(ctx, page)
}
