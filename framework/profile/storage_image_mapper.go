package profiles

import (
	"log/slog"

	dbgen "github.com/leomorpho/goship/db/gen"
	storagerepo "github.com/leomorpho/goship/framework/storage"
)

func mapProfilePhotoSizeRecordsToStorageImages(records []dbgen.ProfilePhotoSizeRecord) []*storagerepo.ImageFile {
	if len(records) == 0 {
		return nil
	}

	byID := make(map[int]*storagerepo.ImageFile, len(records))
	order := make([]int, 0, len(records))

	for _, rec := range records {
		img, exists := byID[rec.ImageID]
		if !exists {
			img = &storagerepo.ImageFile{
				ID:    rec.ImageID,
				Sizes: make([]storagerepo.ImageFileSize, 0, 3),
			}
			byID[rec.ImageID] = img
			order = append(order, rec.ImageID)
		}
		img.Sizes = append(img.Sizes, storagerepo.ImageFileSize{
			Size:      rec.Size,
			Height:    rec.Height,
			Width:     rec.Width,
			ObjectKey: rec.ObjectKey,
		})
	}

	out := make([]*storagerepo.ImageFile, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}
	return out
}

func deleteImageFiles(storage storagerepo.StorageClientInterface, files []*storagerepo.ImageFile) error {
	for _, file := range files {
		for _, size := range file.Sizes {
			if size.ObjectKey == "" {
				continue
			}
			if err := storage.DeleteFile(storagerepo.BucketMainApp, size.ObjectKey); err != nil {
				slog.Error("failed to delete storage artifact while deleting user data",
					"error", err,
					"objectKey", size.ObjectKey,
					"imageID", file.ID,
				)
			}
		}
	}
	return nil
}
