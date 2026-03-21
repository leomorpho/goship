package viewmodels

import (
	"github.com/leomorpho/goship/framework/domain"
	"github.com/leomorpho/goship/framework/web/ui"
)

type (
	ManagedSettingControl struct {
		Key    string
		Label  string
		Value  string
		Source string
		Access string
	}

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
		ActiveSubscriptionPlanKey     string
		ActiveSubscriptionPlanIsPaid  bool
		IsTrial                       bool
		HasMonthlySubscriptionExpiry  bool
		MonthlySybscriptionExpiration string
		TwoFactorEnabled              bool
		ManagedMode                   bool
		ManagedAuthority              string
		ManagedSettings               []ManagedSettingControl

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

	// PreferencesFormData is retained for legacy template bindings until the remaining
	// preferences templates are fully migrated to narrower forms.
	PreferencesFormData struct {
		Bio                     string `form:"bio"`
		SelfBirthdate           string `form:"birthdate"`
		FinishOnboardingRequest bool   `form:"finish_onboarding"`
		Submission              ui.FormSubmission
	}

	ProfileBioFormData struct {
		Bio        string `form:"bio" validate:"required"`
		Submission ui.FormSubmission
	}

	PhoneNumber struct {
		CountryCode     string
		PhoneNumberE164 string
		PhoneVerified   bool
	}

	PhoneNumberVerification struct {
		VerificationCode string `form:"verification_code" validate:"required"`
		Submission       ui.FormSubmission
	}

	SmsVerificationCodeInfo struct {
		ExpirationInMinutes int
	}

	DisplayNameForm struct {
		DisplayName string `form:"name" validate:"required"`
		Submission  ui.FormSubmission
	}
)
