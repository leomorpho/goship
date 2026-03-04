package viewmodels

import "github.com/leomorpho/goship/apps/site/web/ui"

type (
	LoginForm struct {
		Email      string `form:"email" validate:"required,email"`
		Password   string `form:"password" validate:"required"`
		Submission ui.FormSubmission
	}
)
