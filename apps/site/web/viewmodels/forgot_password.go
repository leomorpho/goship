package viewmodels

import "github.com/leomorpho/goship/apps/site/web/ui"

type (
	ForgotPasswordForm struct {
		Email      string `form:"email" validate:"required,email"`
		Submission ui.FormSubmission
	}
)
