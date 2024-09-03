package types

import "github.com/mikestefanello/pagoda/pkg/controller"

type (
	ForgotPasswordForm struct {
		Email      string `form:"email" validate:"required,email"`
		Submission controller.FormSubmission
	}
)
