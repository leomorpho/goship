package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/views"
	"github.com/leomorpho/goship/app/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/views/web/pages/gen"
	"github.com/leomorpho/goship/framework/context"
	frameworkauthcontext "github.com/leomorpho/goship/framework/web/authcontext"
	"github.com/leomorpho/goship/framework/web/ui"
	viewmodels "github.com/leomorpho/goship/framework/web/viewmodels"
	profilesvc "github.com/leomorpho/goship/modules/profile"
)

func NewProfilePrefsRoute(ctr ui.Controller, profileService *profilesvc.ProfileService) profilePrefsRoute {
	return profilePrefsRoute{
		ctr:            ctr,
		profileService: profileService,
	}
}

func (p *profilePrefsRoute) GetBio(ctx echo.Context) error {
	profileID, err := frameworkauthcontext.AuthenticatedProfileID(ctx)
	if err != nil {
		return err
	}
	prof, err := p.profileService.GetProfileSettingsByID(ctx.Request().Context(), profileID)
	if err != nil {
		return err
	}

	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Component = pages.AboutMe(&page)
	page.Name = templates.PagePreferences

	form := viewmodels.NewProfileBioFormData()
	form.Bio = prof.Bio
	page.Form = form

	if form := ctx.Get(context.FormKey); form != nil {
		page.Form = form.(*viewmodels.ProfileBioFormData)
	}

	return p.ctr.RenderPage(ctx, page)
}

func (p *profilePrefsRoute) UpdateBio(ctx echo.Context) error {
	req := updateBioRequest{}
	form := viewmodels.NewProfileBioFormData()
	ctx.Set(context.FormKey, form)

	if err := ctx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid bio data")
	}
	form.Bio = req.Bio

	if err := form.Submission.Process(ctx, *form); err != nil {
		return p.ctr.Fail(err, "unable to process form submission")
	}
	if form.Submission.HasErrors() {
		return p.GetBio(ctx)
	}

	profileID, err := frameworkauthcontext.AuthenticatedProfileID(ctx)
	if err != nil {
		return err
	}

	if err := p.profileService.UpdateProfileBio(ctx.Request().Context(), profileID, form.Bio); err != nil {
		return err
	}

	return p.GetBio(ctx)
}
