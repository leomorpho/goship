package viewmodels

import "github.com/leomorpho/goship/apps/goship/web/ui"

type (
	ContactForm struct {
		Email      string `form:"email" validate:"required,email"`
		Message    string `form:"message" validate:"required"`
		Submission ui.FormSubmission
	}
)
