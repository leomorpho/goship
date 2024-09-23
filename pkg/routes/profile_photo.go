package routes

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/mikestefanello/pagoda/ent"
	"github.com/mikestefanello/pagoda/pkg/context"
	"github.com/mikestefanello/pagoda/pkg/controller"
	"github.com/mikestefanello/pagoda/pkg/repos/profilerepo"
	storagerepo "github.com/mikestefanello/pagoda/pkg/repos/storage"
	"github.com/mikestefanello/pagoda/templates/layouts"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type (
	currProfilePhoto struct {
		ctr           controller.Controller
		profileRepo   *profilerepo.ProfileRepo
		storageClient *storagerepo.StorageClient
		maxFileSizeMB int64 // MB
	}
)

func NewCurrProfilePhotoRoutes(
	ctr controller.Controller, profileRepo *profilerepo.ProfileRepo, storageClient *storagerepo.StorageClient, maxFileSizeMB int64,
) currProfilePhoto {

	return currProfilePhoto{
		ctr:           ctr,
		profileRepo:   profileRepo,
		storageClient: storageClient,
		maxFileSizeMB: maxFileSizeMB,
	}
}

func (p *currProfilePhoto) Get(ctx echo.Context) error {

	page := controller.NewPage(ctx)
	page.Layout = layouts.Main
	page.Title = "Set profile photo"

	// TODO: currently duplicating the upload photo template. Refactor later.
	page.Name = "profile-photo"

	page.Data = photoData{
		MaxFileSizeMB: p.maxFileSizeMB,
	}
	page.Pager = controller.NewPager(ctx, 4)

	return p.ctr.RenderPage(ctx, page)

}

func (p *currProfilePhoto) Post(ctx echo.Context) error {
	// Check if this is an HTMX submission

	// Define max file size
	maxFileSizeMB := p.maxFileSizeMB
	limitString := fmt.Sprintf("%dM", maxFileSizeMB)

	p.ctr.Container.Web.Use(middleware.BodyLimit(limitString))

	// TODO: need to do many other checks for file upload security: https://portswigger.net/web-security/file-upload

	file, err := ctx.FormFile("file")
	if err != nil {
		return p.Get(ctx)
	}

	// Validate and process the image
	err = ValidateAndProcessImage(file)
	ctx.Logger().Error(err)
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

	if err := p.profileRepo.SetProfilePhoto(
		ctx.Request().Context(), profile.ID, src, file.Filename,
	); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			"Failed to upload photo: "+err.Error())
	}
	// msg.Success(ctx, "Successfully uploaded photo.")

	return p.ctr.RenderJSON(ctx, nil)
}

var (
	ErrInvalidMimeType      = errors.New("invalid MIME type")
	ErrInvalidFileExtension = errors.New("invalid file extension")
	ErrImageProcessing      = errors.New("error processing image")
)

func ValidateAndProcessImage(fileHeader *multipart.FileHeader) error {
	// TODO: need to do many other checks for file upload security: https://portswigger.net/web-security/file-upload

	// Define allowed MIME types
	allowedMimeTypes := map[string]bool{
		"image/jpeg": true,
		"image/webp": true,
		"image/png":  true,
		"image/gif":  true,
	}

	// Define allowed file extensions
	allowedExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".webp": true,
		".png":  true,
		".gif":  true,
	}

	// Check MIME type
	contentType := fileHeader.Header.Get("Content-Type")
	if !allowedMimeTypes[contentType] {
		return ErrInvalidMimeType
	}

	// Check file extension
	extension := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !allowedExtensions[extension] {
		return ErrInvalidFileExtension
	}

	// Open the file
	file, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer file.Close()

	// TODO: verify that the file actually contains an image
	return nil
}

// daysAgo returns a time.Time object for x days ago.
func daysAgo(x int) time.Time {
	return time.Now().UTC().AddDate(0, 0, -x)
}
