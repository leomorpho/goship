package contracts

import (
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/app/web/viewmodels"
)

// Route: GET /emailSubscribe
type EmailSubscribePage struct {
	viewmodels.EmailSubscriptionData
}

// Route: POST /emailSubscribe
type EmailSubscribeRequest struct {
	Email      string            `form:"email" validate:"required,email"`
	Latitude   float64           `form:"lat"`
	Longitude  float64           `form:"lon"`
	Submission ui.FormSubmission
}

// Route: GET /about
type AboutPage struct {
	viewmodels.AboutData
}

// Route: GET /privacy-policy
type PrivacyPolicyPage struct {
}
