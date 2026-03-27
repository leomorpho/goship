package profiles

import (
	"testing"

	dbgen "github.com/leomorpho/goship/db/gen"
)

func TestMapProfilePhotoSizeRecordsToStorageImages(t *testing.T) {
	input := []dbgen.ProfilePhotoSizeRecord{
		{ImageID: 9, Size: "thumbnail", Width: 80, Height: 80, ObjectKey: "thumb.jpg"},
		{ImageID: 9, Size: "preview", Width: 400, Height: 300, ObjectKey: "preview.jpg"},
		{ImageID: 4, Size: "full", Width: 1200, Height: 900, ObjectKey: "full.jpg"},
	}

	got := mapProfilePhotoSizeRecordsToStorageImages(input)
	if len(got) != 2 {
		t.Fatalf("expected 2 grouped images, got %d", len(got))
	}
	if got[0].ID != 9 {
		t.Fatalf("expected first image ID 9, got %d", got[0].ID)
	}
	if len(got[0].Sizes) != 2 {
		t.Fatalf("expected first image to have 2 sizes, got %d", len(got[0].Sizes))
	}
	if got[1].ID != 4 {
		t.Fatalf("expected second image ID 4, got %d", got[1].ID)
	}
	if len(got[1].Sizes) != 1 {
		t.Fatalf("expected second image to have 1 size, got %d", len(got[1].Sizes))
	}
}
