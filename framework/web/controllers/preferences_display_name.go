package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/framework/context"
	"github.com/leomorpho/goship/framework/web/layouts/gen"
	"github.com/leomorpho/goship/framework/web/pages/gen"
	"github.com/leomorpho/goship/framework/web/templates"
	"github.com/leomorpho/goship/framework/web/ui"
	viewmodels "github.com/leomorpho/goship/framework/web/viewmodels"
)

func (p *preferences) GetDisplayName(ctx echo.Context) error {
	userIDRaw := ctx.Get(context.AuthenticatedUserIDKey)
	userID, ok := userIDRaw.(int)
	if !ok || userID <= 0 {
		return echo.NewHTTPError(http.StatusUnauthorized, "authenticated user id missing from context")
	}
	displayName, err := p.ctr.Container.Auth.GetUserDisplayNameByUserID(ctx, userID)
	if err != nil {
		return p.ctr.Fail(err, "unable to load display name")
	}

	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Component = pages.DisplayName(&page)
	page.Name = templates.PageDisplayName
	form := viewmodels.NewDisplayNameForm()
	form.DisplayName = displayName
	page.Form = form

	if form := ctx.Get(context.FormKey); form != nil {
		page.Form = form.(*viewmodels.DisplayNameForm)
	}

	return p.ctr.RenderPage(ctx, page)
}

func (p *preferences) SaveDisplayName(ctx echo.Context) error {
	req := updateDisplayNameRequest{}
	form := viewmodels.NewDisplayNameForm()
	ctx.Set(context.FormKey, form)

	if err := ctx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid display name data")
	}
	form.DisplayName = req.DisplayName

	if err := form.Submission.Process(ctx, *form); err != nil {
		return p.ctr.Fail(err, "unable to process form submission")
	}

	if form.Submission.HasErrors() {
		return p.GetDisplayName(ctx)
	}

	userIDRaw := ctx.Get(context.AuthenticatedUserIDKey)
	userID, ok := userIDRaw.(int)
	if !ok || userID <= 0 {
		return echo.NewHTTPError(http.StatusUnauthorized, "authenticated user id missing from context")
	}

	if err := p.ctr.Container.Auth.SetUserDisplayNameByUserID(ctx, userID, form.DisplayName); err != nil {
		return err
	}

	return p.GetDisplayName(ctx)
}
