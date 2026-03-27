package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	frameworkauthcontext "github.com/leomorpho/goship/framework/web/authcontext"
	"github.com/leomorpho/goship/framework/web/routenames"
	"github.com/leomorpho/goship/framework/web/ui"
	profilesvc "github.com/leomorpho/goship/framework/account"
)

func NewOnboardingRoute(ctr ui.Controller, profileService *profilesvc.ProfileService) onboarding {
	return onboarding{
		ctr:            ctr,
		profileService: profileService,
	}
}

func (p *onboarding) Get(ctx echo.Context) error {
	profileID, err := frameworkauthcontext.AuthenticatedProfileID(ctx)
	if err != nil {
		return err
	}

	if err := p.profileService.MarkProfileFullyOnboarded(ctx.Request().Context(), profileID); err != nil {
		return err
	}

	return p.ctr.RedirectWithDetails(ctx, routenames.RouteNameHomeFeed, "?just_finished_onboarding=true", http.StatusFound)
}
