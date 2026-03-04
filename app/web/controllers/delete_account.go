package controllers

import (
	"github.com/labstack/echo/v4"
	routeNames "github.com/leomorpho/goship/app/web/routenames"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/db/ent"
	"github.com/leomorpho/goship/framework/context"
	"github.com/leomorpho/goship/framework/domain"
	"github.com/leomorpho/goship/framework/repos/uxflashmessages"

	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	"github.com/leomorpho/goship/app/profiles"
	"github.com/leomorpho/goship/app/views"
	"github.com/leomorpho/goship/app/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/views/web/pages/gen"
	"github.com/leomorpho/goship/app/web/viewmodels"
)

type (
	deleteAccount struct {
		ctr               ui.Controller
		profileRepo       *profiles.ProfileRepo
		subscriptionsRepo *paidsubscriptions.Service
	}
)

func NewDeleteAccountRoute(
	ctr ui.Controller,
	profileRepo *profiles.ProfileRepo,
	subscriptionsRepo *paidsubscriptions.Service,
) deleteAccount {

	return deleteAccount{
		ctr:               ctr,
		profileRepo:       profileRepo,
		subscriptionsRepo: subscriptionsRepo,
	}
}

func (c *deleteAccount) DeleteAccountPage(ctx echo.Context) error {
	page := ui.NewPage(ctx)

	usr := ctx.Get(context.AuthenticatedUserKey).(*ent.User)
	profile := usr.QueryProfile().FirstX(ctx.Request().Context())

	activePlan, subscriptionExpiredOn, isTrial, err := c.subscriptionsRepo.GetCurrentlyActiveProduct(
		ctx.Request().Context(), profile.ID,
	)

	if err != nil {
		return err
	}
	uncancelledSubscription := *activePlan == domain.ProductTypePro && !isTrial && subscriptionExpiredOn == nil

	page.Layout = layouts.Main
	page.Name = templates.PageDeleteAccount
	page.Component = pages.DeleteAccountPage(&page)
	page.Data = &viewmodels.DeleteAccountData{
		IsPaymentsEnabled:          c.ctr.Container.Config.App.OperationalConstants.PaymentsEnabled,
		HasUncancelledSubscription: uncancelledSubscription,
	}
	page.HTMX.Request.Boosted = true

	return c.ctr.RenderPage(ctx, page)
}

func (c *deleteAccount) DeleteAccountRequest(ctx echo.Context) error {
	usr := ctx.Get(context.AuthenticatedUserKey).(*ent.User)
	profileId := usr.QueryProfile().FirstX(ctx.Request().Context()).ID

	err := c.profileRepo.DeleteUserData(ctx.Request().Context(), profileId)
	if err != nil {
		return err
	}

	if err := c.ctr.Container.Auth.Logout(ctx); err == nil {

	} else {
		uxflashmessages.Danger(ctx, "An error occurred. Please try again.")
	}
	return c.ctr.Redirect(ctx, routeNames.RouteNameLandingPage)
}
