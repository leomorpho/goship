package viewmodels

import "github.com/leomorpho/goship/app/web/ui"

type (
	ResetPasswordForm struct {
		Password        string `form:"password" validate:"required"`
		ConfirmPassword string `form:"password-confirm" validate:"required,eqfield=Password"`
		Submission      ui.FormSubmission
	}
)
