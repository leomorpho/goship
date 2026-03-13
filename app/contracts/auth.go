package contracts

import "github.com/leomorpho/goship/app/web/ui"

// Route: GET /user/login
type LoginPage struct {
	Email string
}

// Route: POST /user/login
type LoginRequest struct {
	Email      string `form:"email" validate:"required,email"`
	Password   string `form:"password" validate:"required"`
	Submission ui.FormSubmission
}

// Route: GET /auth/oauth/:provider
type OAuthStartRequest struct {
	Provider string `param:"provider" validate:"required"`
}

// Route: GET /auth/oauth/:provider/callback
type OAuthCallbackRequest struct {
	Provider string `param:"provider" validate:"required"`
	Code     string `query:"code" validate:"required"`
	State    string `query:"state" validate:"required"`
}

// Route: GET /user/register
type RegisterPage struct {
	UserSignupEnabled  bool
	RelationshipStatus string
	MinDate            string
}

// Route: POST /user/register
type RegisterRequest struct {
	Name       string `form:"name" validate:"required"`
	Email      string `form:"email" validate:"required,email"`
	Password   string `form:"password" validate:"required,min=8"`
	Birthdate  string `form:"birthdate" validate:"required"`
	Submission ui.FormSubmission
}

// Route: GET /user/password
type ForgotPasswordPage struct {
}

// Route: POST /user/password
type ForgotPasswordRequest struct {
	Email      string `form:"email" validate:"required,email"`
	Submission ui.FormSubmission
}

// Route: GET /user/password/reset/:user/:password_token/:token
type ResetPasswordPage struct {
}

// Route: POST /user/password/reset/:user/:password_token/:token
type ResetPasswordRequest struct {
	Password   string `form:"password" validate:"required,min=8"`
	Submission ui.FormSubmission
}
