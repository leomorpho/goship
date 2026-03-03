package types

import "github.com/leomorpho/goship/app/goship/controller"

type (
	LoginForm struct {
		Email      string `form:"email" validate:"required,email"`
		Password   string `form:"password" validate:"required"`
		Submission controller.FormSubmission
	}
)
