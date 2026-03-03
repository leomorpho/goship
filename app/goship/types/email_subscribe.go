package types

import "github.com/leomorpho/goship/app/goship/controller"

type (
	EmailSubscriptionData struct {
		Description string
		Placeholder string
		Latitude    float64
		Longitude   float64
	}

	EmailSubscriptionForm struct {
		Email      string  `form:"email" validate:"required"`
		Latitude   float64 `form:"latitude" validate:"required"`
		Longitude  float64 `form:"longitude" validate:"required"`
		Submission controller.FormSubmission
	}
)
