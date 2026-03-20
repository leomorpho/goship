package contracts

import (
	"github.com/leomorpho/goship/app/web/viewmodels"
)

// Route: GET /
type LandingPage struct {
	viewmodels.LandingPage
}

// Route: GET /homeFeed
type HomeFeedPage struct {
	viewmodels.HomeFeedData
}

// Route: GET /homeFeed/buttons
type HomeFeedButtons struct {
	viewmodels.HomeFeedButtonsData
}
