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
	Email      string  `form:"email" validate:"required,email"`
	Latitude   float64 `form:"latitude"`
	Longitude  float64 `form:"longitude"`
	Submission ui.FormSubmission
}
