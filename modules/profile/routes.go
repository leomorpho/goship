package profiles

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	templates "github.com/leomorpho/goship/app/views"
	"github.com/leomorpho/goship/app/views/web/layouts/gen"
	routeNames "github.com/leomorpho/goship/app/web/routenames"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/app/web/viewmodels"
	customctx "github.com/leomorpho/goship/framework/context"
	"github.com/leomorpho/goship/framework/domain"
	pages "github.com/leomorpho/goship/modules/profile/views/web/pages/gen"
	"github.com/nyaruka/phonenumbers"
	"github.com/rs/zerolog/log"
)

type routeService struct {
	ctr            ui.Controller
	profileService *ProfileService
	maxFileSizeMB  int64
}

type routeRegistrar interface {
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
}

type photoData struct {
	MaxFileSizeMB int64
}

var (
	ErrInvalidMimeType      = errors.New("invalid MIME type")
	ErrInvalidFileExtension = errors.New("invalid file extension")
	ErrImageProcessing      = errors.New("error processing image")
)

func newRouteService(deps ModuleDeps) *routeService {
	return &routeService{
		ctr:            deps.Controller,
		profileService: deps.ProfileService,
		maxFileSizeMB:  deps.MaxFileSizeMB,
	}
}

func registerRoutes(r routeRegistrar, service *routeService) {
	r.GET("/profile", service.getProfile).Name = routeNames.RouteNameProfile
	r.GET("/uploadPhoto", service.getUploadPhoto).Name = routeNames.RouteNameUploadPhoto
	r.POST("/uploadPhoto", service.postUploadPhoto).Name = routeNames.RouteNameUploadPhotoSubmit
	r.DELETE("/uploadPhoto/:image_id", service.deleteUploadPhoto).Name = routeNames.RouteNameUploadPhotoDelete
	r.GET("/currProfilePhoto", service.getCurrentProfilePhoto).Name = routeNames.RouteNameCurrentProfilePhoto
	r.POST("/currProfilePhoto", service.postCurrentProfilePhoto).Name = routeNames.RouteNameCurrentProfilePhotoSubmit
}

func (s *routeService) getProfile(ctx echo.Context) error {
	var otherProfileID int
	var err error
	isSelf := true

	selfProfileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}

	var profileData *domain.Profile
	otherProfileIDStr := ctx.QueryParam(profileIDQueryParam)
	if otherProfileIDStr != "" {
		otherProfileID, err = strconv.Atoi(otherProfileIDStr)
		if err != nil {
			return err
		}
		profileData, err = s.profileService.GetProfileByID(ctx.Request().Context(), otherProfileID, &selfProfileID)
		isSelf = false
	} else {
		profileData, err = s.profileService.GetProfileByID(ctx.Request().Context(), selfProfileID, nil)
	}
	if err != nil {
		return err
	}

	phoneNumber, err := phonenumbers.Parse(profileData.PhoneNumberE164, "")
	if err != nil {
		log.Err(err).Int("profileID", profileData.ID).Msg("failed to parse phone number to international format")
	} else {
		internationalFormat := phonenumbers.Format(phoneNumber, phonenumbers.INTERNATIONAL)
		profileData.PhoneNumberInternational = &internationalFormat
	}

	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Name = templates.PageProfile
	page.Data = viewmodels.ProfilePageData{
		Profile:             *profileData,
		IsSelf:              isSelf,
		UploadGalleryPicUrl: fullSecureURLForRoute(ctx, s.ctr.Container.Config.HTTP.Domain, routeNames.RouteNameUploadPhotoSubmit, page.CSRF),
		UploadProfilePicUrl: fullSecureURLForRoute(ctx, s.ctr.Container.Config.HTTP.Domain, routeNames.RouteNameCurrentProfilePhotoSubmit, page.CSRF),
		GalleryPicsMaxCount: 3,
	}
	page.Component = pages.ProfilePage(&page)
	page.HTMX.Request.Boosted = true
	if isSelf {
		page.SelectedBottomNavbarItem = domain.BottomNavbarItemProfile
	}
	page.ShowBottomNavbar = true
	return s.ctr.RenderPage(ctx, page)
}

func (s *routeService) getUploadPhoto(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Title = "Add Photo"
	page.Name = "upload-photo"
	page.Data = photoData{MaxFileSizeMB: s.maxFileSizeMB}
	page.Pager = ui.NewPager(ctx, 4)
	return s.ctr.RenderPage(ctx, page)
}

func (s *routeService) postUploadPhoto(ctx echo.Context) error {
	s.ctr.Container.Web.Use(echomiddleware.BodyLimit(fmt.Sprintf("%dM", s.maxFileSizeMB)))

	file, err := ctx.FormFile("file")
	if err != nil {
		return s.getUploadPhoto(ctx)
	}
	if err := validateAndProcessImage(file); err != nil {
		switch {
		case errors.Is(err, ErrInvalidMimeType), errors.Is(err, ErrInvalidFileExtension):
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid file type")
		case errors.Is(err, ErrImageProcessing):
			return echo.NewHTTPError(http.StatusInternalServerError, "Error processing image")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	src, err := file.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to open file: "+err.Error())
	}
	defer src.Close()

	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}
	if err := s.profileService.UploadPhoto(ctx.Request().Context(), profileID, src, file.Filename); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to upload photo: "+err.Error())
	}
	return s.ctr.RenderJSON(ctx, nil)
}

func (s *routeService) deleteUploadPhoto(ctx echo.Context) error {
	imageID, err := strconv.Atoi(ctx.Param("image_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid photo ID")
	}
	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}
	if err := s.profileService.DeletePhoto(ctx.Request().Context(), imageID, &profileID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return s.ctr.RedirectWithDetails(ctx, routeNames.RouteNameProfile, "", http.StatusSeeOther)
}

func (s *routeService) getCurrentProfilePhoto(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Title = "Set profile photo"
	page.Name = "profile-photo"
	page.Data = photoData{MaxFileSizeMB: s.maxFileSizeMB}
	page.Pager = ui.NewPager(ctx, 4)
	return s.ctr.RenderPage(ctx, page)
}

func (s *routeService) postCurrentProfilePhoto(ctx echo.Context) error {
	s.ctr.Container.Web.Use(echomiddleware.BodyLimit(fmt.Sprintf("%dM", s.maxFileSizeMB)))

	file, err := ctx.FormFile("file")
	if err != nil {
		return s.getCurrentProfilePhoto(ctx)
	}
	if err := validateAndProcessImage(file); err != nil {
		switch {
		case errors.Is(err, ErrInvalidMimeType), errors.Is(err, ErrInvalidFileExtension):
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid file type")
		case errors.Is(err, ErrImageProcessing):
			return echo.NewHTTPError(http.StatusInternalServerError, "Error processing image")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	src, err := file.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to open file: "+err.Error())
	}
	defer src.Close()

	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}
	if err := s.profileService.SetProfilePhoto(ctx.Request().Context(), profileID, src, file.Filename); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to upload photo: "+err.Error())
	}
	return s.ctr.RenderJSON(ctx, nil)
}

const profileIDQueryParam = "profile_id"

func authenticatedProfileID(ctx echo.Context) (int, error) {
	v := ctx.Get(customctx.AuthenticatedProfileIDKey)
	if v == nil {
		return 0, echo.NewHTTPError(http.StatusUnauthorized, "authenticated profile id missing from context")
	}
	profileID, ok := v.(int)
	if !ok || profileID <= 0 {
		return 0, echo.NewHTTPError(http.StatusUnauthorized, "authenticated profile id missing from context")
	}
	return profileID, nil
}

func fullSecureURLForRoute(ctx echo.Context, domain, routeName, csrf string) string {
	return fmt.Sprintf("%s%s?csrf=%s", domain, ctx.Echo().Reverse(routeName), csrf)
}

func validateAndProcessImage(fileHeader *multipart.FileHeader) error {
	allowedMimeTypes := map[string]bool{
		"image/jpeg": true,
		"image/webp": true,
		"image/png":  true,
		"image/gif":  true,
	}
	allowedExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".webp": true,
		".png":  true,
		".gif":  true,
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if !allowedMimeTypes[contentType] {
		return ErrInvalidMimeType
	}

	extension := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !allowedExtensions[extension] {
		return ErrInvalidFileExtension
	}

	file, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer file.Close()

	return nil
}
