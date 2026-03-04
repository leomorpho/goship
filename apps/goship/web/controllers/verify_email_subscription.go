package controllers

import (
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/apps/goship/app/emailsubscriptions"
	"github.com/leomorpho/goship/apps/goship/views/web/layouts/gen"
	"github.com/leomorpho/goship/apps/goship/web/ui"
)

type verifyEmailSubscription struct {
	ctr                   ui.Controller
	emailSubscriptionRepo emailsubscriptions.EmailSubscriptionRepo
}

func NewVerifyEmailSubscriptionRoute(
	ctr ui.Controller, emailSubscriptionRepo emailsubscriptions.EmailSubscriptionRepo,
) verifyEmailSubscription {

	return verifyEmailSubscription{
		ctr:                   ctr,
		emailSubscriptionRepo: emailSubscriptionRepo,
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

	err := c.emailSubscriptionRepo.ConfirmSubscription(ctx.Request().Context(), token)
	if err != nil {
		page.Data = SubscriptionData{Suceeded: false, SignupEnabled: false}
	} else {
		page.Data = SubscriptionData{Suceeded: true, SignupEnabled: false}
	}

	return c.ctr.RenderPage(ctx, page)
}
