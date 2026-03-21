package controllers

import (
	"github.com/labstack/echo/v4"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	"github.com/leomorpho/goship/framework/repos/uxflashmessages"
	frameworkauthcontext "github.com/leomorpho/goship/framework/web/authcontext"
	"github.com/leomorpho/goship/framework/web/layouts/gen"
	"github.com/leomorpho/goship/framework/web/pages/gen"
	routeNames "github.com/leomorpho/goship/framework/web/routenames"
	"github.com/leomorpho/goship/framework/web/templates"
	"github.com/leomorpho/goship/framework/web/ui"
	"github.com/leomorpho/goship/framework/web/viewmodels"
	profilesvc "github.com/leomorpho/goship/modules/profile"
)

type deleteAccount struct {
	ctr                  ui.Controller
	profileService       *profilesvc.ProfileService
	subscriptionsService *paidsubscriptions.Service
}

func NewDeleteAccountRoute(
	ctr ui.Controller,
	profileService *profilesvc.ProfileService,
	subscriptionsService *paidsubscriptions.Service,
) deleteAccount {
	return deleteAccount{
		ctr:                  ctr,
		profileService:       profileService,
		subscriptionsService: subscriptionsService,
	}
}

func (c *deleteAccount) DeleteAccountPage(ctx echo.Context) error {
	page := ui.NewPage(ctx)

	profileID, err := frameworkauthcontext.AuthenticatedProfileID(ctx)
	if err != nil {
		return err
	}

	activePlan, subscriptionExpiredOn, isTrial, err := c.subscriptionsService.GetCurrentlyActiveProduct(
		ctx.Request().Context(), profileID,
	)
	if err != nil {
		return err
	}
	planKey := c.subscriptionsService.ActivePlanKey(activePlan)
	uncancelledSubscription := c.subscriptionsService.IsPaidPlanKey(planKey) && !isTrial && subscriptionExpiredOn == nil

	page.Layout = layouts.Main
	page.Name = templates.PageDeleteAccount
	page.Component = pages.DeleteAccountPage(&page)
	data := viewmodels.NewDeleteAccountData()
	data.IsPaymentsEnabled = c.ctr.Container.Config.App.OperationalConstants.PaymentsEnabled
	data.HasUncancelledSubscription = uncancelledSubscription
	page.Data = data
	page.HTMX.Request.Boosted = true

	return c.ctr.RenderPage(ctx, page)
}

func (c *deleteAccount) DeleteAccountRequest(ctx echo.Context) error {
	profileID, err := frameworkauthcontext.AuthenticatedProfileID(ctx)
	if err != nil {
		return err
	}

	err = c.profileService.DeleteUserData(ctx.Request().Context(), profileID)
	if err != nil {
		return err
	}

	if err := c.ctr.Container.Auth.Logout(ctx); err != nil {
		uxflashmessages.Danger(ctx, "An error occurred. Please try again.")
	}
	return c.ctr.Redirect(ctx, routeNames.RouteNameLandingPage)
}
