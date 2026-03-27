package controllers

import (
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/framework/mailer"
	emailviews "github.com/leomorpho/goship/framework/views/emails/gen"
	frameworkpage "github.com/leomorpho/goship/framework/web/page"
	"github.com/leomorpho/goship/framework/web/ui"
	"github.com/leomorpho/goship/framework/web/viewmodels"
)

type MailPreviewRoute struct {
	Controller ui.Controller
}

func NewMailPreviewRoute(ctr ui.Controller) MailPreviewRoute {
	return MailPreviewRoute{Controller: ctr}
}

func (r *MailPreviewRoute) Index(ctx echo.Context) error {
	links := []string{
		"/dev/mail/welcome",
		"/dev/mail/password-reset",
		"/dev/mail/verify-email",
	}

	var b strings.Builder
	b.WriteString("<html><body><h1>Email previews</h1><ul>")
	for _, link := range links {
		b.WriteString(`<li><a href="`)
		b.WriteString(link)
		b.WriteString(`">`)
		b.WriteString(link)
		b.WriteString("</a></li>")
	}
	b.WriteString("</ul></body></html>")
	return ctx.HTML(http.StatusOK, b.String())
}

func (r *MailPreviewRoute) Welcome(ctx echo.Context) error {
	data := viewmodels.NewEmailDefaultData()
	data.AppName = string(r.Controller.Container.Config.App.Name)
	data.SupportEmail = r.Controller.Container.Config.App.SupportEmail
	data.Domain = r.Controller.Container.Config.HTTP.Domain
	data.ConfirmationLink = "https://example.test/confirm-email"

	page := &ui.Page{
		Base: frameworkpage.Base{
			Data: data,
		},
	}
	return r.renderEmailPreview(ctx, emailviews.RegistrationConfirmation(page))
}

func (r *MailPreviewRoute) PasswordReset(ctx echo.Context) error {
	data := viewmodels.NewEmailPasswordResetData()
	data.AppName = string(r.Controller.Container.Config.App.Name)
	data.SupportEmail = r.Controller.Container.Config.App.SupportEmail
	data.Domain = r.Controller.Container.Config.HTTP.Domain
	data.ProfileName = "Preview User"
	data.PasswordResetLink = "https://example.test/reset-password"
	data.OperatingSystem = "macOS"
	data.BrowserName = "Firefox"

	page := &ui.Page{
		Base: frameworkpage.Base{
			Data: data,
		},
	}
	return r.renderEmailPreview(ctx, emailviews.PasswordReset(page))
}

func (r *MailPreviewRoute) VerifyEmail(ctx echo.Context) error {
	data := viewmodels.NewEmailDefaultData()
	data.AppName = string(r.Controller.Container.Config.App.Name)
	data.SupportEmail = r.Controller.Container.Config.App.SupportEmail
	data.Domain = r.Controller.Container.Config.HTTP.Domain
	data.ConfirmationLink = "https://example.test/verify-email"

	page := &ui.Page{
		Base: frameworkpage.Base{
			Data: data,
		},
	}
	return r.renderEmailPreview(ctx, emailviews.RegistrationConfirmation(page))
}

func (r *MailPreviewRoute) renderEmailPreview(ctx echo.Context, component templ.Component) error {
	html, _, err := mailer.RenderEmail(ctx.Request().Context(), component)
	if err != nil {
		return err
	}
	return ctx.HTML(http.StatusOK, html)
}
