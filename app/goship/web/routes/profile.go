package routes

import (
	"fmt"
	"strconv"

	"github.com/leomorpho/goship/app/goship/views"
	"github.com/leomorpho/goship/app/goship/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/goship/views/web/pages/gen"
	"github.com/leomorpho/goship/ent"
	"github.com/leomorpho/goship/pkg/context"
	"github.com/leomorpho/goship/pkg/controller"
	"github.com/leomorpho/goship/pkg/domain"
	"github.com/leomorpho/goship/pkg/repos/profilerepo"
	"github.com/leomorpho/goship/pkg/types"
	"github.com/nyaruka/phonenumbers"
	"github.com/rs/zerolog/log"

	"github.com/labstack/echo/v4"
)

// TODO: currProfilePage and otherProfilePage should really be one. Return self if profile_id is not present in current otherProfilePage
type (
	singleProfile struct {
		ctr         controller.Controller
		profileRepo *profilerepo.ProfileRepo
	}
)

func NewProfileRoutes(
	ctr controller.Controller, profileRepo *profilerepo.ProfileRepo,
) singleProfile {

	return singleProfile{
		ctr:         ctr,
		profileRepo: profileRepo,
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
		profileData, err = c.profileRepo.GetProfileByID(
			ctx.Request().Context(), otherProfileID, &selfProfileID,
		)
		isSelf = false
	} else {
		profileData, err = c.profileRepo.GetProfileByID(
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

	page := controller.NewPage(ctx)

	uploadProfilePicUrl := GetFullSecureUrlForRoute(
		ctx, c.ctr.Container.Config.HTTP.Domain, "currProfilePhoto.post", page.CSRF)
	uploadGalleryPicUrl := GetFullSecureUrlForRoute(
		ctx, c.ctr.Container.Config.HTTP.Domain, "uploadPhoto.post", page.CSRF)

	// Setting to 3 pics at most for now
	galleryPicsMaxCount := 3

	page.Layout = layouts.Main
	page.Name = templates.PageProfile

	data := types.ProfilePageData{
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
