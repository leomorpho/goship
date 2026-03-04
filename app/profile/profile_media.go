package profiles

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/jpeg"
	"io"
	"path/filepath"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"github.com/leomorpho/goship/db/ent"
	"github.com/leomorpho/goship/db/ent/filestorage"
	"github.com/leomorpho/goship/db/ent/image"
	"github.com/leomorpho/goship/db/ent/imagesize"
	"github.com/leomorpho/goship/db/ent/profile"
	"github.com/leomorpho/goship/db/ent/user"
	"github.com/leomorpho/goship/framework/domain"
	storagerepo "github.com/leomorpho/goship/framework/repos/storage"
	"github.com/rs/zerolog/log"
)

func (p *ProfileService) GetPhotosByProfileByID(
	ctx context.Context, profileID int,
) ([]domain.Photo, error) {
	photoObjects, err := p.orm.Profile.
		Query().
		Where(profile.IDEQ(profileID)).
		QueryPhotos().
		Order(ent.Desc(filestorage.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	photos, err := p.storageRepo.GetImageObjectsFromFiles(photoObjects)
	if err != nil {
		if customErr, ok := err.(*storagerepo.NoImagesInFiles); ok {
			log.Error().Str("Error", customErr.Message)
		} else {
			return nil, err
		}
	}

	return photos, nil
}

func (p *ProfileService) GetProfilePhotoThumbnailURL(userID int) string {
	defaultProfilePic := "https://www.gravatar.com/avatar/?d=mp&s=200"
	ctx := context.Background()
	image, err := p.orm.Profile.
		Query().
		Where(profile.HasUserWith(user.IDEQ(userID))).
		QueryProfileImage().
		WithSizes(func(s *ent.ImageSizeQuery) {
			s.WithFile()
		}).
		Only(ctx)
	if ent.IsNotFound(err) {
		return defaultProfilePic
	}
	photo, err := p.storageRepo.GetImageObjectFromFile(image)
	if err != nil {
		if customErr, ok := err.(*storagerepo.NoImagesInFiles); ok {
			log.Error().Str("Error", customErr.Message)
		} else {
			return defaultProfilePic
		}
	}
	return photo.ThumbnailURL
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

	profileImage, err := p.orm.Profile.
		Query().
		Where(
			profile.IDEQ(profileID),
		).
		QueryProfileImage().
		First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return err
	}
	if err == nil {
		err = p.DeletePhoto(ctx, profileImage.ID, nil)
		if err != nil {
			log.Err(err).Str("err", "failed to delete old photo when uploading new profile photo")
		}
	}

	imageID, err := p.UploadImageSizes(
		ctx, profileID, fileStream, domain.ImageCategoryProfilePhoto, fileName,
		[]domain.ImageSize{domain.ImageSizeThumbnail, domain.ImageSizePreview})
	if err != nil {
		return err
	}

	_, err = p.orm.Profile.UpdateOneID(profileID).SetProfileImageID(*imageID).Save(ctx)
	if err != nil {
		return err
	}
	return nil
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

	imageID, err := p.UploadImageSizes(
		ctx, profileID, fileStream, domain.ImageCategoryProfileGallery, fileName,
		[]domain.ImageSize{domain.ImageSizeFull, domain.ImageSizePreview})
	if err != nil {
		return err
	}
	_, err = p.orm.Profile.UpdateOneID(profileID).AddPhotoIDs(*imageID).Save(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (p *ProfileService) UploadImageSizes(
	ctx context.Context,
	profileID int,
	fileStream io.Reader,
	imageCategory domain.ImageCategory,
	fileName string,
	sizes []domain.ImageSize,
) (*int, error) {

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

	var imageSizeIDs []int

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

		log.Info().
			Str("imageCategory", imageCategory.Value).
			Str("sizeName", sizeName.Value).
			Int("profileID", profileID).
			Msg("Uploading image for profile")

		// Upload the resized image
		filestorageEntryID, err := p.storageRepo.UploadFile(storagerepo.BucketMainApp, objectName, bytes.NewReader(buf.Bytes()))
		if err != nil {
			log.Err(err).
				Str("imageCategory", imageCategory.Value).
				Str("sizeName", sizeName.Value).
				Int("profileID", profileID).
				Msg("failed to upload image for profile")
			return nil, fmt.Errorf("failed to upload %s image: %w", sizeName, err)
		}

		// Create an image size record
		imageSize, err := p.orm.ImageSize.
			Create().
			SetSize(imagesize.Size(sizeName.Value)).
			SetWidth(actualWidth).
			SetHeight(actualHeight).
			SetFileID(*filestorageEntryID).
			Save(ctx)
		if err != nil {
			log.Err(err).
				Str("imageCategory", imageCategory.Value).
				Str("sizeName", sizeName.Value).
				Int("profileID", profileID).
				Msg("failed to save image size objects for profile")
			return nil, fmt.Errorf("failed to save image size for %s: %w", sizeName, err)
		}

		imageSizeIDs = append(imageSizeIDs, imageSize.ID)
	}

	image, err := p.orm.Image.
		Create().
		SetType(image.Type(imageCategory.Value)).
		AddSizeIDs(imageSizeIDs...).
		Save(ctx)
	if err != nil {
		log.Err(err).
			Str("imageCategory", imageCategory.Value).
			Int("profileID", profileID).
			Msg("failed to create image object for profile")
		return nil, err
	}
	log.Info().
		Str("imageCategory", imageCategory.Value).
		Int("profileID", profileID).
		Msg("Successfully added Image for profile")

	return &image.ID, nil
}

func (p *ProfileService) DeletePhoto(
	ctx context.Context,
	imageID int,
	profileID *int,

) error {
	if p.storageRepo == nil {
		return errors.New("storage orm not initialized")
	}

	if profileID != nil {
		// Check that the photo belongs to the profile
		exists, err := p.orm.Profile.
			Query().
			Where(
				profile.HasPhotosWith(image.IDEQ(imageID)),
				profile.IDEQ(*profileID),
			).Exist(ctx)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("no photo with id %d belonging to user", imageID)
		}
	}

	im, err := p.orm.Image.
		Query().
		Where(
			image.IDEQ(imageID),
		).
		WithSizes(func(s *ent.ImageSizeQuery) {
			s.WithFile()
		}).
		Only(ctx)

	if err != nil {
		return err
	}

	for _, size := range im.Edges.Sizes {
		if size.Edges.File != nil {
			// Delete the file in S3
			err = p.storageRepo.DeleteFile(storagerepo.BucketMainApp, size.Edges.File.ObjectKey)
			if err != nil {
				return err
			}
			// NOTE: filestorage object gets deleted by storage repo
		}
	}

	// TODO: verify cascade delete works for image size and files linked to image
	return p.orm.Image.DeleteOneID(imageID).Exec(ctx)
}
