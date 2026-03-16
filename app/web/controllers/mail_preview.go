package controllers

import (
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/controller"
	emailviews "github.com/leomorpho/goship/app/views/emails/gen"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/app/web/viewmodels"
	"github.com/leomorpho/goship/framework/repos/mailer"
)

type mailPreview struct {
	ctr ui.Controller
}

func NewMailPreviewRoute(ctr ui.Controller) mailPreview {
	return mailPreview{ctr: ctr}
}

func (r *mailPreview) Index(ctx echo.Context) error {
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

func (r *mailPreview) Welcome(ctx echo.Context) error {
	page := &controller.Page{
		Data: viewmodels.EmailDefaultData{
			AppName:          string(r.ctr.Container.Config.App.Name),
			SupportEmail:     r.ctr.Container.Config.App.SupportEmail,
			Domain:           r.ctr.Container.Config.HTTP.Domain,
			ConfirmationLink: "https://example.test/confirm-email",
		},
	}
	return r.renderEmailPreview(ctx, emailviews.RegistrationConfirmation(page))
}

func (r *mailPreview) PasswordReset(ctx echo.Context) error {
	page := &controller.Page{
		Data: viewmodels.EmailPasswordResetData{
			AppName:           string(r.ctr.Container.Config.App.Name),
			SupportEmail:      r.ctr.Container.Config.App.SupportEmail,
			Domain:            r.ctr.Container.Config.HTTP.Domain,
			ProfileName:       "Preview User",
			PasswordResetLink: "https://example.test/reset-password",
			OperatingSystem:   "macOS",
			BrowserName:       "Firefox",
		},
	}
	return r.renderEmailPreview(ctx, emailviews.PasswordReset(page))
}

func (r *mailPreview) VerifyEmail(ctx echo.Context) error {
	page := &controller.Page{
		Data: viewmodels.EmailDefaultData{
			AppName:          string(r.ctr.Container.Config.App.Name),
			SupportEmail:     r.ctr.Container.Config.App.SupportEmail,
			Domain:           r.ctr.Container.Config.HTTP.Domain,
			ConfirmationLink: "https://example.test/verify-email",
		},
	}
	return r.renderEmailPreview(ctx, emailviews.RegistrationConfirmation(page))
}

func (r *mailPreview) renderEmailPreview(ctx echo.Context, component templ.Component) error {
	html, _, err := mailer.RenderEmail(ctx.Request().Context(), component)
	if err != nil {
		return err
	}
	return ctx.HTML(http.StatusOK, html)
}
