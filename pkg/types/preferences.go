package types

import (
	"time"

	"github.com/mikestefanello/pagoda/pkg/controller"
	"github.com/mikestefanello/pagoda/pkg/domain"
)

type (
	PreferencesData struct {
		// Form data
		Bio                     string
		PhoneNumberInE164Format string
		CountryCode             string
		SelfBirthdate           string

		// Validation data
		IsProfileFullyOnboarded bool
		DefaultBio              string
		DefaultBirthdate        string

		IsPaymentsEnabled             bool
		ActiveSubscriptionPlan        domain.ProductType
		IsTrial                       bool
		MonthlySybscriptionExpiration *time.Time

		NotificationPermissionsData NotificationPermissionsData
	}

	DeleteAccountData struct {
		IsPaymentsEnabled          bool
		HasUncancelledSubscription bool
	}

	NotificationPermissionsData struct {
		// Permissions                    []domain.NotificationPermission
		PermissionDailyNotif          domain.NotificationPermission
		PermissionPartnerActivity     domain.NotificationPermission
		VapidPublicKey                string
		SubscribedEndpoints           []string
		PhoneSubscriptionEnabled      bool
		NotificationTypeQueryParamKey string

		AddPushSubscriptionEndpoint    string
		DeletePushSubscriptionEndpoint string

		AddFCMPushSubscriptionEndpoint    string
		DeleteFCMPushSubscriptionEndpoint string

		AddEmailSubscriptionEndpoint    string
		DeleteEmailSubscriptionEndpoint string

		AddSmsSubscriptionEndpoint    string
		DeleteSmsSubscriptionEndpoint string
	}

	PushNotificationSubscriptions struct {
		URLs []string `json:"urls"`
	}

	// TODO: deprecated. If we remove it now, we need to clean it up
	// from go templates too (errors triggered from it).
	PreferencesFormData struct {
		Bio                     string `form:"bio"`
		SelfBirthdate           string `form:"birthdate"`
		FinishOnboardingRequest bool   `form:"finish_onboarding"`
		Submission              controller.FormSubmission
	}

	ProfileBioFormData struct {
		Bio        string `form:"bio" validate:"required"`
		Submission controller.FormSubmission
	}

	PhoneNumber struct {
		CountryCode     string
		PhoneNumberE164 string
		PhoneVerified   bool
	}

	PhoneNumberVerification struct {
		VerificationCode string `form:"verification_code" validate:"required"`
		Submission       controller.FormSubmission
	}

	SmsVerificationCodeInfo struct {
		ExpirationInMinutes int
	}

	DisplayNameForm struct {
		DisplayName string `form:"name" validate:"required"`
		Submission  controller.FormSubmission
	}
)
