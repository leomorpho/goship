package controllers

import (
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/goship/repos/emailsmanager"
	"github.com/leomorpho/goship/app/goship/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/goship/webui"
)

type verifyEmailSubscription struct {
	ctr                   webui.Controller
	emailSubscriptionRepo emailsmanager.EmailSubscriptionRepo
}

func NewVerifyEmailSubscriptionRoute(
	ctr webui.Controller, emailSubscriptionRepo emailsmanager.EmailSubscriptionRepo,
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
	page := webui.NewPage(ctx)
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
