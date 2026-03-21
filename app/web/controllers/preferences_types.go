package controllers

import (
	"github.com/leomorpho/goship-modules/notifications"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	"github.com/leomorpho/goship/framework/web/ui"
	profilesvc "github.com/leomorpho/goship/modules/profile"
)

type profilePrefsRoute struct {
	ctr            ui.Controller
	profileService *profilesvc.ProfileService
}

type updateBioRequest struct {
	Bio        string `form:"bio" validate:"required"`
	Submission ui.FormSubmission
}

type verifyPhoneRequest struct {
	VerificationCode string `form:"verification_code" validate:"required"`
	Submission       ui.FormSubmission
}

type updatePhoneRequest struct {
	PhoneNumberE164Format string `form:"phone_number_e164" validate:"required"`
	CountryCode           string `form:"country_code" validate:"required"`
	Submission            ui.FormSubmission
}

type updateDisplayNameRequest struct {
	DisplayName string `form:"name" validate:"required"`
	Submission  ui.FormSubmission
}

type preferences struct {
	ctr                           ui.Controller
	profileService                profilesvc.ProfileService
	pushNotificationsRepo         *notifications.PwaPushService
	notificationPermissionService *notifications.NotificationPermissionService
	subscriptionsService          *paidsubscriptions.Service
	smsSenderService              *notifications.SMSSender
}

type onboarding struct {
	ctr            ui.Controller
	profileService *profilesvc.ProfileService
}
