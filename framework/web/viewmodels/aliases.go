package viewmodels

import (
	appviewmodels "github.com/leomorpho/goship/app/web/viewmodels"
	"github.com/leomorpho/goship/framework/domain"
)

type (
	CommittedModePageData         = appviewmodels.CommittedModePageData
	CountByDay                    = appviewmodels.CountByDay
	CreateCheckoutSessionForm     = appviewmodels.CreateCheckoutSessionForm
	DeleteAccountData             = appviewmodels.DeleteAccountData
	DisplayNameForm               = appviewmodels.DisplayNameForm
	DropdownIterable              = appviewmodels.DropdownIterable
	EmailDefaultData              = appviewmodels.EmailDefaultData
	EmailPasswordResetData        = appviewmodels.EmailPasswordResetData
	EmailUpdate                   = appviewmodels.EmailUpdate
	ForgotPasswordForm            = appviewmodels.ForgotPasswordForm
	HomeFeedButtonsData           = appviewmodels.HomeFeedButtonsData
	HomeFeedData                  = appviewmodels.HomeFeedData
	HomeFeedStatsData             = appviewmodels.HomeFeedStatsData
	LandingPage                   = appviewmodels.LandingPage
	LocalizationPageData          = appviewmodels.LocalizationPageData
	LoginForm                     = appviewmodels.LoginForm
	LoginOAuthData                = appviewmodels.LoginOAuthData
	LoginOAuthProvider            = appviewmodels.LoginOAuthProvider
	ManagedSettingControl         = appviewmodels.ManagedSettingControl
	NormalNotificationsPageData   = appviewmodels.NormalNotificationsPageData
	NotificationItem              = appviewmodels.NotificationItem
	NotificationPermissionsData   = appviewmodels.NotificationPermissionsData
	PaymentProcessorPublicKey     = appviewmodels.PaymentProcessorPublicKey
	PhoneNumber                   = appviewmodels.PhoneNumber
	PhoneNumberVerification       = appviewmodels.PhoneNumberVerification
	Post                          = appviewmodels.Post
	PreferencesData               = appviewmodels.PreferencesData
	PreferencesFormData           = appviewmodels.PreferencesFormData
	PricingPageData               = appviewmodels.PricingPageData
	ProductDescription            = appviewmodels.ProductDescription
	ProfileBioFormData            = appviewmodels.ProfileBioFormData
	ProfileCalendarHeatmap        = appviewmodels.ProfileCalendarHeatmap
	ProfilePageData               = appviewmodels.ProfilePageData
	PushNotificationSubscriptions = appviewmodels.PushNotificationSubscriptions
	QAItem                        = appviewmodels.QAItem
	QuestionInEmail               = appviewmodels.QuestionInEmail
	RegisterData                  = appviewmodels.RegisterData
	RegisterForm                  = appviewmodels.RegisterForm
	ResetPasswordForm             = appviewmodels.ResetPasswordForm
	SearchResult                  = appviewmodels.SearchResult
	SmsVerificationCodeInfo       = appviewmodels.SmsVerificationCodeInfo
	TwoFactorBackupCodesData      = appviewmodels.TwoFactorBackupCodesData
	TwoFactorSetupData            = appviewmodels.TwoFactorSetupData
	TwoFactorSetupForm            = appviewmodels.TwoFactorSetupForm
	TwoFactorVerifyData           = appviewmodels.TwoFactorVerifyData
	TwoFactorVerifyForm           = appviewmodels.TwoFactorVerifyForm
	UpdateInAppModeForm           = appviewmodels.UpdateInAppModeForm
)

var (
	NewCommittedModePageData         = appviewmodels.NewCommittedModePageData
	NewCountByDay                    = appviewmodels.NewCountByDay
	NewCreateCheckoutSessionForm     = appviewmodels.NewCreateCheckoutSessionForm
	NewDeleteAccountData             = appviewmodels.NewDeleteAccountData
	NewDisplayNameForm               = appviewmodels.NewDisplayNameForm
	NewDropdownIterable              = appviewmodels.NewDropdownIterable
	NewEmailDefaultData              = appviewmodels.NewEmailDefaultData
	NewEmailPasswordResetData        = appviewmodels.NewEmailPasswordResetData
	NewEmailUpdate                   = appviewmodels.NewEmailUpdate
	NewForgotPasswordForm            = appviewmodels.NewForgotPasswordForm
	NewHomeFeedButtonsData           = appviewmodels.NewHomeFeedButtonsData
	NewHomeFeedData                  = appviewmodels.NewHomeFeedData
	NewHomeFeedStatsData             = appviewmodels.NewHomeFeedStatsData
	NewLandingPage                   = appviewmodels.NewLandingPage
	NewLocalizationPageData          = appviewmodels.NewLocalizationPageData
	NewLoginForm                     = appviewmodels.NewLoginForm
	NewLoginOAuthData                = appviewmodels.NewLoginOAuthData
	NewLoginOAuthProvider            = appviewmodels.NewLoginOAuthProvider
	NewManagedSettingControl         = appviewmodels.NewManagedSettingControl
	NewNormalNotificationsPageData   = appviewmodels.NewNormalNotificationsPageData
	NewNotificationItem              = appviewmodels.NewNotificationItem
	NewNotificationPermissionsData   = appviewmodels.NewNotificationPermissionsData
	NewPaymentProcessorPublicKey     = appviewmodels.NewPaymentProcessorPublicKey
	NewPhoneNumber                   = appviewmodels.NewPhoneNumber
	NewPhoneNumberVerification       = appviewmodels.NewPhoneNumberVerification
	NewPost                          = appviewmodels.NewPost
	NewPreferencesData               = appviewmodels.NewPreferencesData
	NewPreferencesFormData           = appviewmodels.NewPreferencesFormData
	NewPricingPageData               = appviewmodels.NewPricingPageData
	NewProductDescription            = appviewmodels.NewProductDescription
	NewProfileBioFormData            = appviewmodels.NewProfileBioFormData
	NewProfileCalendarHeatmap        = appviewmodels.NewProfileCalendarHeatmap
	NewProfilePageData               = appviewmodels.NewProfilePageData
	NewPushNotificationSubscriptions = appviewmodels.NewPushNotificationSubscriptions
	NewQAItem                        = appviewmodels.NewQAItem
	NewQuestionInEmail               = appviewmodels.NewQuestionInEmail
	NewRegisterData                  = appviewmodels.NewRegisterData
	NewRegisterForm                  = appviewmodels.NewRegisterForm
	NewResetPasswordForm             = appviewmodels.NewResetPasswordForm
	NewSearchResult                  = appviewmodels.NewSearchResult
	NewSmsVerificationCodeInfo       = appviewmodels.NewSmsVerificationCodeInfo
	NewTwoFactorBackupCodesData      = appviewmodels.NewTwoFactorBackupCodesData
	NewTwoFactorSetupData            = appviewmodels.NewTwoFactorSetupData
	NewTwoFactorSetupForm            = appviewmodels.NewTwoFactorSetupForm
	NewTwoFactorVerifyData           = appviewmodels.NewTwoFactorVerifyData
	NewTwoFactorVerifyForm           = appviewmodels.NewTwoFactorVerifyForm
	NewUpdateInAppModeForm           = appviewmodels.NewUpdateInAppModeForm
)

func NotificationItemsFromDomain(items []*domain.Notification) []NotificationItem {
	return appviewmodels.NotificationItemsFromDomain(items)
}
