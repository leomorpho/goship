package viewmodels

import "github.com/leomorpho/goship/apps/site/web/ui"

type (
	RegisterForm struct {
		RelationshipStatus string `form:"relationship_status" validate:"required"`
		Name               string `form:"name" validate:"required"`
		Email              string `form:"email" validate:"required,email"`
		Password           string `form:"password" validate:"required"`
		Birthdate          string `form:"birthdate" validate:"required"`
		Submission         ui.FormSubmission
	}

	RegisterData struct {
		RelationshipStatus string
		UserSignupEnabled  bool
		MinDate            string
	}
)
