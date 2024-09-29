package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/mikestefanello/pagoda/ent"
	"github.com/mikestefanello/pagoda/pkg/context"
	"github.com/mikestefanello/pagoda/pkg/controller"
	"github.com/mikestefanello/pagoda/pkg/domain"
	"github.com/mikestefanello/pagoda/pkg/repos/msg"
	routeNames "github.com/mikestefanello/pagoda/pkg/routing/routenames"

	"github.com/mikestefanello/pagoda/pkg/repos/profilerepo"
	"github.com/mikestefanello/pagoda/pkg/repos/subscriptions"
	"github.com/mikestefanello/pagoda/pkg/types"
	"github.com/mikestefanello/pagoda/templates"
	"github.com/mikestefanello/pagoda/templates/layouts"
	"github.com/mikestefanello/pagoda/templates/pages"
)

type (
	deleteAccount struct {
		ctr               controller.Controller
		profileRepo       *profilerepo.ProfileRepo
		subscriptionsRepo *subscriptions.SubscriptionsRepo
	}
)

func NewDeleteAccountRoute(
	ctr controller.Controller,
	profileRepo *profilerepo.ProfileRepo,
	subscriptionsRepo *subscriptions.SubscriptionsRepo,
) deleteAccount {

	return deleteAccount{
		ctr:               ctr,
		profileRepo:       profileRepo,
		subscriptionsRepo: subscriptionsRepo,
	}
}

func (c *deleteAccount) DeleteAccountPage(ctx echo.Context) error {
	page := controller.NewPage(ctx)

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
	page.Data = &types.DeleteAccountData{
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
