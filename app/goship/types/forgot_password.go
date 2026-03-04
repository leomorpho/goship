package types

import "github.com/leomorpho/goship/app/goship/webui"

type (
	ForgotPasswordForm struct {
		Email      string `form:"email" validate:"required,email"`
		Submission webui.FormSubmission
	}
)
