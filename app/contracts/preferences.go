package contracts

import (
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/app/web/viewmodels"
)

// Route: GET /preferences
type PreferencesPage struct {
	viewmodels.PreferencesData
}

// Route: GET /preferences/phone
type EditPhonePage struct {
	viewmodels.PhoneNumber
}

// Route: POST /preferences/phone/save
type UpdatePhoneRequest struct {
	PhoneNumberE164Format string `form:"phone_number_e164" validate:"required"`
	CountryCode           string `form:"country_code" validate:"required"`
	Submission            ui.FormSubmission
}

// Route: POST /preferences/phone/verification
type VerifyPhoneRequest struct {
	VerificationCode string `form:"verification_code" validate:"required"`
	Submission       ui.FormSubmission
}

// Route: POST /preferences/display-name/save
type UpdateDisplayNameRequest struct {
	DisplayName string `form:"name" validate:"required"`
	Submission  ui.FormSubmission
}

// Route: POST /profileBio/update
type UpdateBioRequest struct {
	Bio        string `form:"bio" validate:"required"`
	Submission ui.FormSubmission
}
