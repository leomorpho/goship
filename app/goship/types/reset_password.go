package types

import "github.com/leomorpho/goship/app/goship/webui"

type (
	ResetPasswordForm struct {
		Password        string `form:"password" validate:"required"`
		ConfirmPassword string `form:"password-confirm" validate:"required,eqfield=Password"`
		Submission      webui.FormSubmission
	}
)
