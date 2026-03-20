package contracts

import (
	"github.com/leomorpho/goship/framework/domain"
)

// Route: GET /profile
type ProfilePage struct {
	Profile                     domain.Profile
	PhotosJSON                  string
	IsSelf                      bool
	ShowGenderAndAge            bool
	UploadProfilePicUrl         string
	UploadGalleryPicUrl         string
	CanUploadMoreGalleryPics    bool
	GalleryPicsMaxCount         int
	JustAcceptedCommitedRequest bool
}

// Route: POST /uploadPhoto
type UploadPhotoRequest struct {
	// Handled via multipart form: "file"
}

// Route: POST /currProfilePhoto
type SetProfilePhotoRequest struct {
	// Handled via multipart form: "file"
}
