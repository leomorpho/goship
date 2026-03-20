package controllers

import (
	"github.com/labstack/echo/v4"
	modemailsubscriptions "github.com/leomorpho/goship-modules/emailsubscriptions"
	"github.com/leomorpho/goship/app/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/web/ui"
)

type verifyEmailSubscription struct {
	ctr                ui.Controller
	emailSubscriptions modemailsubscriptions.Service
}

func NewVerifyEmailSubscriptionRoute(
	ctr ui.Controller, emailSubscriptions modemailsubscriptions.Service,
) verifyEmailSubscription {

	return verifyEmailSubscription{
		ctr:                ctr,
		emailSubscriptions: emailSubscriptions,
	}
}

type SubscriptionData struct {
	Suceeded      bool
	SignupEnabled bool
}

func (c *verifyEmailSubscription) Get(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Name = "subscribe-confirmation"

	// Validate the token
	token := ctx.Param("token")

	err := c.emailSubscriptions.Confirm(ctx.Request().Context(), token)
	if err != nil {
		page.Data = SubscriptionData{Suceeded: false, SignupEnabled: false}
	} else {
		page.Data = SubscriptionData{Suceeded: true, SignupEnabled: false}
	}

	return c.ctr.RenderPage(ctx, page)
}
