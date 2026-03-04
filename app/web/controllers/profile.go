package controllers

import (
	"fmt"
	"strconv"

	profilesvc "github.com/leomorpho/goship/app/profile"
	"github.com/leomorpho/goship/app/views"
	"github.com/leomorpho/goship/app/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/views/web/pages/gen"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/app/web/viewmodels"
	"github.com/leomorpho/goship/db/ent"
	"github.com/leomorpho/goship/framework/context"
	"github.com/leomorpho/goship/framework/domain"
	"github.com/nyaruka/phonenumbers"
	"github.com/rs/zerolog/log"

	"github.com/labstack/echo/v4"
)

// TODO: currProfilePage and otherProfilePage should really be one. Return self if profile_id is not present in current otherProfilePage
type (
	singleProfile struct {
		ctr            ui.Controller
		profileService *profilesvc.ProfileService
	}
)

func NewProfileRoutes(
	ctr ui.Controller, profileService *profilesvc.ProfileService,
) singleProfile {

	return singleProfile{
		ctr:            ctr,
		profileService: profileService,
	}
}

const PROFILE_ID_QUERY_PARAM = "profile_id"

func (c *singleProfile) Get(ctx echo.Context) error {
	var otherProfileID int
	var selfProfileID int
	var err error
	isSelf := true

	usr := ctx.Get(context.AuthenticatedUserKey).(*ent.User)

	selfProfileID = usr.QueryProfile().
		FirstX(ctx.Request().Context()).ID

	var profileData *domain.Profile
	otherProfileIdStr := ctx.QueryParam(PROFILE_ID_QUERY_PARAM)
	if otherProfileIdStr != "" {
		otherProfileID, err = strconv.Atoi(otherProfileIdStr)
		if err != nil {
			return err
		}
		profileData, err = c.profileService.GetProfileByID(
			ctx.Request().Context(), otherProfileID, &selfProfileID,
		)
		isSelf = false
	} else {
		profileData, err = c.profileService.GetProfileByID(
			ctx.Request().Context(), selfProfileID, nil,
		)
		isSelf = true
	}
	if err != nil {
		return err
	}

	// Parse the phone number
	phoneNumber, err := phonenumbers.Parse(profileData.PhoneNumberE164, "")
	if err != nil {
		log.Err(err).Int("profileID", profileData.ID).Msg("Failed to parse phone number to international format")
	}
	// Format the number in international format
	internationalFormat := phonenumbers.Format(phoneNumber, phonenumbers.INTERNATIONAL)
	profileData.PhoneNumberInternational = &internationalFormat

	page := ui.NewPage(ctx)

	uploadProfilePicUrl := GetFullSecureUrlForRoute(
		ctx, c.ctr.Container.Config.HTTP.Domain, "currProfilePhoto.post", page.CSRF)
	uploadGalleryPicUrl := GetFullSecureUrlForRoute(
		ctx, c.ctr.Container.Config.HTTP.Domain, "uploadPhoto.post", page.CSRF)

	// Setting to 3 pics at most for now
	galleryPicsMaxCount := 3

	page.Layout = layouts.Main
	page.Name = templates.PageProfile

	data := viewmodels.ProfilePageData{
		Profile:             *profileData,
		IsSelf:              isSelf,
		UploadGalleryPicUrl: uploadGalleryPicUrl,
		UploadProfilePicUrl: uploadProfilePicUrl,
		GalleryPicsMaxCount: galleryPicsMaxCount,
	}

	page.Data = data
	page.Component = pages.ProfilePage(&page)
	page.HTMX.Request.Boosted = true

	if isSelf {
		page.SelectedBottomNavbarItem = domain.BottomNavbarItemProfile
	}
	page.ShowBottomNavbar = true

	return c.ctr.RenderPage(ctx, page)
}

func GetFullSecureUrlForRoute(ctx echo.Context, domain, routeName, csrf string) string {
	url := ctx.Echo().Reverse(routeName)
	return fmt.Sprintf("%s%s?csrf=%s", domain, url, csrf)
}
