package controllers

import (
	"github.com/labstack/echo/v4"
	routeNames "github.com/leomorpho/goship/apps/goship/web/routenames"
	"github.com/leomorpho/goship/apps/goship/web/ui"
	"github.com/leomorpho/goship/ent"
	"github.com/leomorpho/goship/pkg/context"
	"github.com/leomorpho/goship/pkg/domain"
	"github.com/leomorpho/goship/pkg/repos/msg"

	"github.com/leomorpho/goship/apps/goship/app/profiles"
	"github.com/leomorpho/goship/apps/goship/app/subscriptions"
	"github.com/leomorpho/goship/apps/goship/web/viewmodels"
	"github.com/leomorpho/goship/apps/goship/views"
	"github.com/leomorpho/goship/apps/goship/views/web/layouts/gen"
	"github.com/leomorpho/goship/apps/goship/views/web/pages/gen"
)

type (
	deleteAccount struct {
		ctr               ui.Controller
		profileRepo       *profiles.ProfileRepo
		subscriptionsRepo *subscriptions.SubscriptionsRepo
	}
)

func NewDeleteAccountRoute(
	ctr ui.Controller,
	profileRepo *profiles.ProfileRepo,
	subscriptionsRepo *subscriptions.SubscriptionsRepo,
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
		msg.Danger(ctx, "An error occurred. Please try again.")
	}
	return c.ctr.Redirect(ctx, routeNames.RouteNameLandingPage)
}
