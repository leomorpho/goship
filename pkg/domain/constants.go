package domain

import "time"

var DefaultBirthdate = time.Date(1898, time.January, 6, 0, 0, 0, 0, time.UTC)

// ImageSizeEnumToSizeMap maps an image size name to its actual size
var ImageSizeEnumToSizeMap = map[ImageSize]int{
	ImageSizeThumbnail: 150,  // max 150 pixels wide or tall
	ImageSizePreview:   800,  // max 800 pixels wide or tall
	ImageSizeFull:      1600, // max 1600 pixels wide or tall
}

const (
	DefaultBio                 = "Hello, there! 🌟"
	FreePlanNumAnswersPerDay   = 1
	PermissionNotificationType = "permission"
)
