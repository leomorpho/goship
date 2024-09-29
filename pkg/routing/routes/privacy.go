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
	privacyPolicy struct {
		ctr controller.Controller
	}
)

func NewPrivacyPolicyRoute(ctr controller.Controller) privacyPolicy {
	return privacyPolicy{
		ctr: ctr,
	}
}

func (c *privacyPolicy) Get(ctx echo.Context) error {

	page := controller.NewPage(ctx)
	page.Layout = layouts.Main
	page.Name = templates.PagePrivacyPolicy
	page.Component = pages.PrivacyPolicy(&page)
	page.Data = types.AboutData{
		SupportEmail: c.ctr.Container.Config.App.SupportEmail,
	}

	page.HTMX.Request.Boosted = true

	return c.ctr.RenderPage(ctx, page)
}
