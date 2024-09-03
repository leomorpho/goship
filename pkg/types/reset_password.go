package types

import "github.com/mikestefanello/pagoda/pkg/controller"

type (
	ResetPasswordForm struct {
		Password        string `form:"password" validate:"required"`
		ConfirmPassword string `form:"password-confirm" validate:"required,eqfield=Password"`
		Submission      controller.FormSubmission
	}
)
