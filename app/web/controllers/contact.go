package controllers

import (
	"fmt"

	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/framework/context"
	"github.com/leomorpho/goship/framework/repos/msg"

	"github.com/leomorpho/goship/app/views"
	"github.com/leomorpho/goship/app/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/views/web/pages/gen"
	"github.com/leomorpho/goship/app/web/viewmodels"

	"github.com/labstack/echo/v4"
)

type (
	contact struct {
		ui.Controller
	}
)

func (c *contact) Get(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Name = templates.PageContact
	page.Title = "Contact us"
	page.Form = &viewmodels.ContactForm{}
	page.Component = pages.Contact(&page)
	page.HTMX.Request.Boosted = true

	if form := ctx.Get(context.FormKey); form != nil {
		page.Form = form.(*viewmodels.ContactForm)
	}
	msg.Success(ctx, "Success!")
	msg.Warning(ctx, "Warning!")
	msg.Danger(ctx, "Danger!")
	msg.Info(ctx, "Info!")

	return c.RenderPage(ctx, page)
}

func (c *contact) Post(ctx echo.Context) error {
	var form viewmodels.ContactForm
	ctx.Set(context.FormKey, &form)

	// Parse the form values
	if err := ctx.Bind(&form); err != nil {
		return c.Fail(err, "unable to bind form")
	}

	if err := form.Submission.Process(ctx, form); err != nil {
		return c.Fail(err, "unable to process form submission")
	}

	if !form.Submission.HasErrors() {
		err := c.Container.Mail.
			Compose().
			To(form.Email).
			Subject("Contact form submitted").
			Body(fmt.Sprintf("The message is: %s", form.Message)).
			Send(ctx.Request().Context())

		if err != nil {
			return c.Fail(err, "unable to send email")
		}
	}

	return c.Get(ctx)
}
