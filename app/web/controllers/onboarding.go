package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/framework/web/routenames"
	"github.com/leomorpho/goship/framework/web/ui"
	profilesvc "github.com/leomorpho/goship/modules/profile"
)

func NewOnboardingRoute(ctr ui.Controller, profileService *profilesvc.ProfileService) onboarding {
	return onboarding{
		ctr:            ctr,
		profileService: profileService,
	}
}

func (p *onboarding) Get(ctx echo.Context) error {
	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}

	if err := p.profileService.MarkProfileFullyOnboarded(ctx.Request().Context(), profileID); err != nil {
		return err
	}

	return p.ctr.RedirectWithDetails(ctx, routenames.RouteNameHomeFeed, "?just_finished_onboarding=true", http.StatusFound)
}
