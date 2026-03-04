package controllers

import (
	"github.com/leomorpho/goship/app/views"
	"github.com/leomorpho/goship/app/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/views/web/pages/gen"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/app/web/viewmodels"

	"github.com/labstack/echo/v4"
)

type (
	privacyPolicy struct {
		ctr ui.Controller
	}
)

func NewPrivacyPolicyRoute(ctr ui.Controller) privacyPolicy {
	return privacyPolicy{
		ctr: ctr,
	}
}

func (c *privacyPolicy) Get(ctx echo.Context) error {

	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Name = templates.PagePrivacyPolicy
	page.Component = pages.PrivacyPolicy(&page)
	page.Data = viewmodels.AboutData{
		SupportEmail: c.ctr.Container.Config.App.SupportEmail,
	}

	page.HTMX.Request.Boosted = true

	return c.ctr.RenderPage(ctx, page)
}
