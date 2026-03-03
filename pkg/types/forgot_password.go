package types

import "github.com/leomorpho/goship/pkg/controller"

type (
	ForgotPasswordForm struct {
		Email      string `form:"email" validate:"required,email"`
		Submission controller.FormSubmission
	}
)
