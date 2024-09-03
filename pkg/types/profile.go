package types

import (
	"github.com/mikestefanello/pagoda/pkg/domain"
)

type ProfilePageData struct {
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

type ProfileCalendarHeatmap struct {
	Counts []CountByDay
}
type CountByDay struct {
	Date  string `json:"date"`
	Value int    `json:"value"`
}

type LocalizationPageData struct {
	Latitude       float64
	Longitude      float64
	Radius         int
	PostGeoDataURL string
}
