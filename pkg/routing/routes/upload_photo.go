package routes

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/mikestefanello/pagoda/ent"
	"github.com/mikestefanello/pagoda/pkg/context"
	"github.com/mikestefanello/pagoda/pkg/controller"
	"github.com/mikestefanello/pagoda/pkg/repos/profilerepo"
	storagerepo "github.com/mikestefanello/pagoda/pkg/repos/storage"
	"github.com/mikestefanello/pagoda/pkg/routing/routenames"
	"github.com/mikestefanello/pagoda/templates/layouts"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type (
	uploadPhoto struct {
		ctr           controller.Controller
		profileRepo   *profilerepo.ProfileRepo
		storageClient *storagerepo.StorageClient
		maxFileSizeMB int64 // MB
	}

	photoData struct {
		MaxFileSizeMB int64 // MB
	}
)

func NewUploadPhotoRoutes(
	ctr controller.Controller, profileRepo *profilerepo.ProfileRepo, storageClient *storagerepo.StorageClient, maxFileSizeMB int64,
) uploadPhoto {

	return uploadPhoto{
		ctr:           ctr,
		profileRepo:   profileRepo,
		storageClient: storageClient,
		maxFileSizeMB: maxFileSizeMB,
	}
}

func (p *uploadPhoto) Get(ctx echo.Context) error {

	page := controller.NewPage(ctx)
	page.Layout = layouts.Main
	page.Title = "Add Photo"
	page.Name = "upload-photo"

	page.Data = photoData{
		MaxFileSizeMB: p.maxFileSizeMB,
	}
	page.Pager = controller.NewPager(ctx, 4)

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

	usr := ctx.Get(context.AuthenticatedUserKey).(*ent.User)

	profile, err := usr.QueryProfile().First(ctx.Request().Context())
	if err != nil {
		return err
	}

	if err := p.profileRepo.UploadPhoto(
		ctx.Request().Context(), profile.ID, src, file.Filename,
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

	usr := ctx.Get(context.AuthenticatedUserKey).(*ent.User)
	profileID := usr.QueryProfile().FirstX(ctx.Request().Context()).ID

	err = p.profileRepo.DeletePhoto(ctx.Request().Context(), imageID, &profileID)
	if err != nil {
		// Handle error, e.g., photo not found, database error, etc.
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return p.ctr.RedirectWithDetails(ctx, routenames.RouteNameProfile, "", http.StatusSeeOther)
}
