package routes

import (
	"fmt"

	"github.com/leomorpho/goship/app/goship/controller"
	"github.com/leomorpho/goship/pkg/context"
	"github.com/leomorpho/goship/pkg/repos/msg"

	"github.com/leomorpho/goship/app/goship/views"
	"github.com/leomorpho/goship/app/goship/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/goship/views/web/pages/gen"
	"github.com/leomorpho/goship/app/goship/types"

	"github.com/labstack/echo/v4"
)

type (
	contact struct {
		controller.Controller
	}
)

func (c *contact) Get(ctx echo.Context) error {
	page := controller.NewPage(ctx)
	page.Layout = layouts.Main
	page.Name = templates.PageContact
	page.Title = "Contact us"
	page.Form = &types.ContactForm{}
	page.Component = pages.Contact(&page)
	page.HTMX.Request.Boosted = true

	if form := ctx.Get(context.FormKey); form != nil {
		page.Form = form.(*types.ContactForm)
	}
	msg.Success(ctx, "Success!")
	msg.Warning(ctx, "Warning!")
	msg.Danger(ctx, "Danger!")
	msg.Info(ctx, "Info!")

	return c.RenderPage(ctx, page)
}

func (c *contact) Post(ctx echo.Context) error {
	var form types.ContactForm
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
