package profiles

import (
	"context"

	dbgen "github.com/leomorpho/goship/db/gen"
)

func (p *ProfileService) deleteProfileStorageFiles(ctx context.Context, profileID int) error {
	if p.storageRepo == nil {
		return ErrProfileDBNotConfigured
	}

	photoRows, err := dbgen.GetProfilePhotosByProfileID(ctx, p.db, p.dbDialect, profileID)
	if err != nil {
		return err
	}
	photoFiles := mapProfilePhotoSizeRecordsToStorageImages(photoRows)
	if err := deleteImageFiles(p.storageRepo, photoFiles); err != nil {
		return err
	}

	imageRows, err := dbgen.GetProfileImageByProfileID(ctx, p.db, p.dbDialect, profileID)
	if err != nil {
		return err
	}
	imageFiles := mapProfilePhotoSizeRecordsToStorageImages(imageRows)
	if len(imageFiles) > 0 {
		if err := deleteImageFiles(p.storageRepo, imageFiles); err != nil {
			return err
		}
	}
	return nil
}
