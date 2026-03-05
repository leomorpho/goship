package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	profilesvc "github.com/leomorpho/goship/app/profile"
	"github.com/leomorpho/goship/app/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/web/routenames"
	"github.com/leomorpho/goship/app/web/ui"
	storagerepo "github.com/leomorpho/goship/framework/repos/storage"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type (
	uploadPhoto struct {
		ctr            ui.Controller
		profileService *profilesvc.ProfileService
		storageClient  *storagerepo.StorageClient
		maxFileSizeMB  int64 // MB
	}

	photoData struct {
		MaxFileSizeMB int64 // MB
	}
)

func NewUploadPhotoRoutes(
	ctr ui.Controller, profileService *profilesvc.ProfileService, storageClient *storagerepo.StorageClient, maxFileSizeMB int64,
) uploadPhoto {

	return uploadPhoto{
		ctr:            ctr,
		profileService: profileService,
		storageClient:  storageClient,
		maxFileSizeMB:  maxFileSizeMB,
	}
}

func (p *uploadPhoto) Get(ctx echo.Context) error {

	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Title = "Add Photo"
	page.Name = "upload-photo"

	page.Data = photoData{
		MaxFileSizeMB: p.maxFileSizeMB,
	}
	page.Pager = ui.NewPager(ctx, 4)

	return p.ctr.RenderPage(ctx, page)

}

func (p *uploadPhoto) Post(ctx echo.Context) error {
	// Check if this is an HTMX submission

	// Define max file size
	maxFileSizeMB := p.maxFileSizeMB
	limitString := fmt.Sprintf("%dM", maxFileSizeMB)

	p.ctr.Container.Web.Use(middleware.BodyLimit(limitString))

	file, err := ctx.FormFile("file")
	if err != nil {
		return p.Get(ctx)
	}

	// Validate and process the image
	err = ValidateAndProcessImage(file)
	if err != nil {
		// Handle specific errors returned by ValidateAndProcessImage
		if errors.Is(err, ErrInvalidMimeType) || errors.Is(err, ErrInvalidFileExtension) {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid file type")
		} else if errors.Is(err, ErrImageProcessing) {
			return echo.NewHTTPError(http.StatusInternalServerError, "Error processing image")
		} else {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	src, err := file.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			"Failed to open file: "+err.Error())
	}
	defer src.Close()

	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}

	if err := p.profileService.UploadPhoto(
		ctx.Request().Context(), profileID, src, file.Filename,
	); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			"Failed to upload photo: "+err.Error())
	}
	// msg.Success(ctx, "Successfully uploaded photo.")

	return p.ctr.RenderJSON(ctx, nil)
}

func (p *uploadPhoto) Delete(ctx echo.Context) error {

	idStr := ctx.Param("image_id")
	imageID, err := strconv.Atoi(idStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid photo ID")
	}

	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}

	err = p.profileService.DeletePhoto(ctx.Request().Context(), imageID, &profileID)
	if err != nil {
		// Handle error, e.g., photo not found, database error, etc.
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return p.ctr.RedirectWithDetails(ctx, routenames.RouteNameProfile, "", http.StatusSeeOther)
}
