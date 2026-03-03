package types

import "github.com/leomorpho/goship/app/goship/controller"

type (
	ContactForm struct {
		Email      string `form:"email" validate:"required,email"`
		Message    string `form:"message" validate:"required"`
		Submission controller.FormSubmission
	}
)
