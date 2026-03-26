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
	"github.com/leomorpho/goship/framework/domain"
	frameworkauthcontext "github.com/leomorpho/goship/framework/web/authcontext"
	layouts "github.com/leomorpho/goship/framework/web/layouts/gen"
	routeNames "github.com/leomorpho/goship/framework/web/routenames"
	templates "github.com/leomorpho/goship/framework/web/templates"
	"github.com/leomorpho/goship/framework/web/ui"
	"github.com/leomorpho/goship/framework/web/viewmodels"
	pages "github.com/leomorpho/goship/modules/profile/views/web/pages/gen"
	"github.com/nyaruka/phonenumbers"
	"log/slog"
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
		slog.Error("failed to parse phone number to international format", "error", err, "profileID", profileData.ID)
	} else {
		internationalFormat := phonenumbers.Format(phoneNumber, phonenumbers.INTERNATIONAL)
		profileData.PhoneNumberInternational = &internationalFormat
	}

	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Name = templates.PageProfile
	data := viewmodels.NewProfilePageData()
	data.Profile = *profileData
	data.IsSelf = isSelf
	data.UploadGalleryPicUrl = fullSecureURLForRoute(ctx, s.ctr.Container.Config.HTTP.Domain, routeNames.RouteNameUploadPhotoSubmit, page.CSRF)
	data.UploadProfilePicUrl = fullSecureURLForRoute(ctx, s.ctr.Container.Config.HTTP.Domain, routeNames.RouteNameCurrentProfilePhotoSubmit, page.CSRF)
	data.GalleryPicsMaxCount = 3
	page.Data = data
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
	return frameworkauthcontext.AuthenticatedProfileID(ctx)
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
