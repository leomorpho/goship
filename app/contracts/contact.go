package contracts

import "github.com/leomorpho/goship/app/web/ui"

// Route: POST /contact
type ContactRequest struct {
	Email      string `form:"email" validate:"required,email"`
	Message    string `form:"message" validate:"required"`
	Submission ui.FormSubmission
}
