package profiles

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"image/jpeg"
	"io"
	"path/filepath"
	"time"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	dbgen "github.com/leomorpho/goship/db/gen"
	"github.com/leomorpho/goship/framework/domain"
	storagerepo "github.com/leomorpho/goship/framework/storage"
	"log/slog"
)

func (p *ProfileService) GetPhotosByProfileByID(
	ctx context.Context, profileID int,
) ([]domain.Photo, error) {
	if p.db == nil {
		return nil, ErrProfileDBNotConfigured
	}
	rows, err := dbgen.GetProfilePhotosByProfileID(ctx, p.db, p.dbDialect, profileID)
	if err != nil {
		return nil, err
	}
	photos, err := p.storageRepo.GetImageObjectsFromFiles(mapProfilePhotoSizeRecordsToStorageImages(rows))
	if err != nil {
		if customErr, ok := err.(*storagerepo.NoImagesInFiles); ok {
			slog.Error("Error in GetPhotosByProfileByID", "error", customErr.Message)
		} else {
			return nil, err
		}
	}

	return photos, nil
}

func (p *ProfileService) GetProfilePhotoThumbnailURL(userID int) string {
	defaultProfilePic := "https://www.gravatar.com/avatar/?d=mp&s=200"
	if p.db == nil || p.storageRepo == nil {
		slog.Warn("profile thumbnail lookup unavailable: missing db or storage dependency", "userID", userID)
		return defaultProfilePic
	}

	ctx := context.Background()
	objectKey, err := dbgen.GetProfileThumbnailObjectKeyByUserID(ctx, p.db, p.dbDialect, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return defaultProfilePic
	}
	if err != nil {
		slog.Error("failed to fetch profile thumbnail object key", "error", err, "userID", userID)
		return defaultProfilePic
	}
	if objectKey == "" {
		return defaultProfilePic
	}
	url, urlErr := p.storageRepo.GetPresignedURL(storagerepo.BucketMainApp, objectKey, 2*24*time.Hour)
	if urlErr != nil {
		slog.Error("failed to sign profile thumbnail URL", "error", urlErr, "userID", userID, "objectKey", objectKey)
		return defaultProfilePic
	}
	return url
}

func (p *ProfileService) SetProfilePhoto(
	ctx context.Context,
	profileID int,
	fileStream io.Reader,
	fileName string,
) error {
	if p.storageRepo == nil {
		return errors.New("storage orm not initialized")
	}
	if p.db == nil {
		return ErrProfileDBNotConfigured
	}

	profileImageID, err := dbgen.GetProfileImageIDByProfileID(ctx, p.db, p.dbDialect, profileID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil && profileImageID.Valid {
		err = p.DeletePhoto(ctx, int(profileImageID.Int64), nil)
		if err != nil {
			slog.Error("failed to delete old photo when uploading new profile photo", "error", err)
		}
	}

	imageID, err := p.UploadImageSizes(
		ctx, profileID, fileStream, domain.ImageCategoryProfilePhoto, fileName,
		[]domain.ImageSize{domain.ImageSizeThumbnail, domain.ImageSizePreview})
	if err != nil {
		return err
	}
	return dbgen.SetProfileImageID(ctx, p.db, p.dbDialect, profileID, *imageID)
}

func (p *ProfileService) UploadPhoto(
	ctx context.Context,
	profileID int,
	fileStream io.Reader,
	fileName string,
) error {
	if p.storageRepo == nil {
		return errors.New("storage orm not initialized")
	}
	if p.db == nil {
		return ErrProfileDBNotConfigured
	}

	imageID, err := p.UploadImageSizes(
		ctx, profileID, fileStream, domain.ImageCategoryProfileGallery, fileName,
		[]domain.ImageSize{domain.ImageSizeFull, domain.ImageSizePreview})
	if err != nil {
		return err
	}
	return dbgen.AttachGalleryImageToProfile(ctx, p.db, p.dbDialect, *imageID, profileID)
}

func (p *ProfileService) UploadImageSizes(
	ctx context.Context,
	profileID int,
	fileStream io.Reader,
	imageCategory domain.ImageCategory,
	fileName string,
	sizes []domain.ImageSize,
) (*int, error) {
	if p.db == nil {
		return nil, ErrProfileDBNotConfigured
	}

	// Before decoding, try to seek to the beginning of the stream
	if seeker, ok := fileStream.(io.Seeker); ok {
		_, err := seeker.Seek(0, io.SeekStart)
		if err != nil {
			return nil, fmt.Errorf("failed to seek file stream: %w", err)
		}
	}

	// Decode the image from the stream
	img, err := imaging.Decode(fileStream)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	type uploadedImageSize struct {
		size      domain.ImageSize
		width     int
		height    int
		fileID    int
		objectKey string
	}
	var uploaded []uploadedImageSize

	for _, sizeName := range sizes {

		maxSize, ok := domain.ImageSizeEnumToSizeMap[sizeName]
		if !ok {
			return nil, err
		}
		// Resize image maintaining the aspect ratio
		resizedImage := imaging.Fit(img, maxSize, maxSize, imaging.Lanczos)

		// Get actual dimensions of the resized image
		actualWidth := resizedImage.Bounds().Dx()
		actualHeight := resizedImage.Bounds().Dy()

		// Create a buffer to write the resized image for upload
		buf := new(bytes.Buffer)
		if err := jpeg.Encode(buf, resizedImage, &jpeg.Options{Quality: 90}); err != nil {
			return nil, fmt.Errorf("failed to encode resized image for %s: %w", sizeName, err)
		}

		// Generate a unique object name
		ext := filepath.Ext(fileName)
		if ext == "" {
			return nil, fmt.Errorf("invalid file extension for %s", fileName)
		}
		objectName := fmt.Sprintf("profile_%d_%s_%s_%s%s",
			profileID, imageCategory.Value, uuid.New().String(), sizeName.Value, ext)

		slog.Info("Uploading image for profile",
			"imageCategory", imageCategory.Value,
			"sizeName", sizeName.Value,
			"profileID", profileID,
		)

		// Upload the resized image
		filestorageEntryID, err := p.storageRepo.UploadFile(storagerepo.BucketMainApp, objectName, bytes.NewReader(buf.Bytes()))
		if err != nil {
			slog.Error("failed to upload image for profile",
				"error", err,
				"imageCategory", imageCategory.Value,
				"sizeName", sizeName.Value,
				"profileID", profileID,
			)
			return nil, fmt.Errorf("failed to upload %s image: %w", sizeName, err)
		}

		// Create an image size record
		uploaded = append(uploaded, uploadedImageSize{
			size:      sizeName,
			width:     actualWidth,
			height:    actualHeight,
			fileID:    *filestorageEntryID,
			objectKey: objectName,
		})
	}

	now := time.Now().UTC()
	imageID, err := dbgen.InsertImage(ctx, p.db, p.dbDialect, imageCategory.Value, now, now)
	if err != nil {
		slog.Error("failed to create image object for profile",
			"error", err,
			"imageCategory", imageCategory.Value,
			"profileID", profileID,
		)
		return nil, err
	}

	for _, item := range uploaded {
		if err := dbgen.InsertImageSize(
			ctx,
			p.db,
			p.dbDialect,
			item.size.Value,
			item.width,
			item.height,
			imageID,
			item.fileID,
			now,
			now,
		); err != nil {
			slog.Error("failed to save image size objects for profile",
				"error", err,
				"imageCategory", imageCategory.Value,
				"sizeName", item.size.Value,
				"profileID", profileID,
			)
			return nil, fmt.Errorf("failed to save image size for %s: %w", item.size.Value, err)
		}
	}

	slog.Info("Successfully added Image for profile",
		"imageCategory", imageCategory.Value,
		"profileID", profileID,
	)

	return &imageID, nil
}

func (p *ProfileService) DeletePhoto(
	ctx context.Context,
	imageID int,
	profileID *int,

) error {
	if p.storageRepo == nil {
		return errors.New("storage orm not initialized")
	}
	if p.db == nil {
		return ErrProfileDBNotConfigured
	}

	if profileID != nil {
		exists, err := dbgen.ImageBelongsToProfileGallery(ctx, p.db, p.dbDialect, imageID, *profileID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("no photo with id %d belonging to user", imageID)
		}
	}

	rows, err := dbgen.GetImageStorageObjectsByImageID(ctx, p.db, p.dbDialect, imageID)
	if err != nil {
		return err
	}
	imageFiles := mapProfilePhotoSizeRecordsToStorageImages(rows)
	for _, file := range imageFiles {
		for _, size := range file.Sizes {
			if size.ObjectKey == "" {
				continue
			}
			if err := p.storageRepo.DeleteFile(storagerepo.BucketMainApp, size.ObjectKey); err != nil {
				return err
			}
		}
	}
	// Ensure profile FK is detached before deleting the image row.
	if err := dbgen.ClearProfileImageByImageID(ctx, p.db, p.dbDialect, imageID); err != nil {
		return err
	}
	return dbgen.DeleteImageByID(ctx, p.db, p.dbDialect, imageID)
}
