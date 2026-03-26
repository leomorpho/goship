package controllers

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/framework/context"
	"github.com/leomorpho/goship/framework/repos/uxflashmessages"
	frameworkauthcontext "github.com/leomorpho/goship/framework/web/authcontext"
	layouts "github.com/leomorpho/goship/framework/web/layouts/gen"
	pages "github.com/leomorpho/goship/framework/web/pages/gen"
	"github.com/leomorpho/goship/framework/web/templates"
	"github.com/leomorpho/goship/framework/web/ui"
	viewmodels "github.com/leomorpho/goship/framework/web/viewmodels"
)

func (p *preferences) GetPhoneComponent(ctx echo.Context) error {
	profileID, err := frameworkauthcontext.AuthenticatedProfileID(ctx)
	if err != nil {
		return err
	}
	profile, err := p.profileService.GetProfileSettingsByID(ctx.Request().Context(), profileID)
	if err != nil {
		return err
	}

	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Component = pages.EditPhonePage(&page)
	page.Name = templates.PagePhoneNumber
	page.HTMX.Request.Boosted = true

	data := viewmodels.NewPhoneNumber()
	data.CountryCode = profile.CountryCode
	data.PhoneNumberE164 = profile.PhoneNumberE164
	data.PhoneVerified = profile.PhoneVerified
	page.Data = data

	return p.ctr.RenderPage(ctx, page)
}

func (p *preferences) GetPhoneVerificationComponent(ctx echo.Context) error {
	profileID, err := frameworkauthcontext.AuthenticatedProfileID(ctx)
	if err != nil {
		return err
	}
	profile, err := p.profileService.GetProfileSettingsByID(ctx.Request().Context(), profileID)
	if err != nil {
		return err
	}

	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Name = templates.PagePhoneNumber
	page.Form = viewmodels.NewPhoneNumberVerification()
	page.Component = pages.PhoneVerificationField(&page)
	data := viewmodels.NewSmsVerificationCodeInfo()
	data.ExpirationInMinutes = p.ctr.Container.Config.Phone.ValidationCodeExpirationMinutes
	page.Data = data

	if form := ctx.Get(context.FormKey); form != nil {
		page.Form = form.(*viewmodels.PhoneNumberVerification)
	}

	_, err = p.smsSenderService.CreateConfirmationCode(ctx.Request().Context(), profile.ID, profile.PhoneNumberE164)
	if err != nil {
		slog.Error("failed to send verification code", "error", err)
		uxflashmessages.Danger(ctx, "Failed to send verification code 😨")
		return p.ctr.RenderPage(ctx, page)
	}

	return p.ctr.RenderPage(ctx, page)
}

func (p *preferences) SubmitPhoneVerificationCode(ctx echo.Context) error {
	req := verifyPhoneRequest{}
	form := viewmodels.NewPhoneNumberVerification()
	ctx.Set(context.FormKey, form)

	if err := ctx.Bind(&req); err != nil {
		return p.ctr.Fail(err, "unable to parse verification code form")
	}
	form.VerificationCode = req.VerificationCode

	if err := form.Submission.Process(ctx, *form); err != nil {
		return p.ctr.Fail(err, "unable to process form submission")
	}

	if form.Submission.HasErrors() {
		return p.GetPhoneVerificationComponent(ctx)
	}

	if form.VerificationCode == "" {
		form.Submission.SetFieldError("VerificationCode", "Invalid code")
		uxflashmessages.Danger(ctx, "Invalid code. Please try again.")
		return p.GetPhoneVerificationComponent(ctx)
	}

	profileID, err := frameworkauthcontext.AuthenticatedProfileID(ctx)
	if err != nil {
		return err
	}
	profile, err := p.profileService.GetProfileSettingsByID(ctx.Request().Context(), profileID)
	if err != nil {
		return err
	}

	valid, err := p.smsSenderService.VerifyConfirmationCode(ctx.Request().Context(), profile.ID, form.VerificationCode)
	if err != nil || !valid {
		form.Submission.SetFieldError("VerificationCode", "Invalid code")
		uxflashmessages.Danger(ctx, "Invalid code. Please try again.")
		return p.GetPhoneVerificationComponent(ctx)
	}

	uxflashmessages.Success(ctx, "Success! Your phone number was confirmed.")
	return p.GetPhoneVerificationComponent(ctx)
}

func (p *preferences) SavePhoneInfo(ctx echo.Context) error {
	req := updatePhoneRequest{}
	ctx.Set(context.FormKey, &req)

	if err := ctx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid bio data")
	}

	if err := req.Submission.Process(ctx, req); err != nil {
		return p.ctr.Fail(err, "unable to process form submission")
	}

	if req.Submission.HasErrors() {
		return p.ctr.Redirect(ctx, "preferences")
	}

	profileID, err := frameworkauthcontext.AuthenticatedProfileID(ctx)
	if err != nil {
		return err
	}

	return p.profileService.UpdateProfilePhone(
		ctx.Request().Context(),
		profileID,
		req.CountryCode,
		req.PhoneNumberE164Format,
	)
}
