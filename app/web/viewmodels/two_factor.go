package viewmodels

import "github.com/leomorpho/goship/app/web/ui"

type TwoFactorSetupData struct {
	QRCodeDataURL string
	ManualKey     string
}

type TwoFactorSetupForm struct {
	Code       string `form:"code"`
	Submission ui.FormSubmission
}

type TwoFactorBackupCodesData struct {
	Codes []string
}

type TwoFactorVerifyData struct{}

type TwoFactorVerifyForm struct {
	Code       string `form:"code"`
	Submission ui.FormSubmission
}
