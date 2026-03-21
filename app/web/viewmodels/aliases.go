package viewmodels

import (
	"github.com/leomorpho/goship/framework/domain"
	frameworkviewmodels "github.com/leomorpho/goship/framework/web/viewmodels"
)

type (
	CommittedModePageData         = frameworkviewmodels.CommittedModePageData
	CountByDay                    = frameworkviewmodels.CountByDay
	CreateCheckoutSessionForm     = frameworkviewmodels.CreateCheckoutSessionForm
	DeleteAccountData             = frameworkviewmodels.DeleteAccountData
	DisplayNameForm               = frameworkviewmodels.DisplayNameForm
	DropdownIterable              = frameworkviewmodels.DropdownIterable
	EmailDefaultData              = frameworkviewmodels.EmailDefaultData
	EmailPasswordResetData        = frameworkviewmodels.EmailPasswordResetData
	EmailUpdate                   = frameworkviewmodels.EmailUpdate
	ForgotPasswordForm            = frameworkviewmodels.ForgotPasswordForm
	HomeFeedButtonsData           = frameworkviewmodels.HomeFeedButtonsData
	HomeFeedData                  = frameworkviewmodels.HomeFeedData
	HomeFeedStatsData             = frameworkviewmodels.HomeFeedStatsData
	LandingPage                   = frameworkviewmodels.LandingPage
	LocalizationPageData          = frameworkviewmodels.LocalizationPageData
	LoginForm                     = frameworkviewmodels.LoginForm
	LoginOAuthData                = frameworkviewmodels.LoginOAuthData
	LoginOAuthProvider            = frameworkviewmodels.LoginOAuthProvider
	ManagedSettingControl         = frameworkviewmodels.ManagedSettingControl
	NormalNotificationsPageData   = frameworkviewmodels.NormalNotificationsPageData
	NotificationItem              = frameworkviewmodels.NotificationItem
	NotificationPermissionsData   = frameworkviewmodels.NotificationPermissionsData
	PaymentProcessorPublicKey     = frameworkviewmodels.PaymentProcessorPublicKey
	PhoneNumber                   = frameworkviewmodels.PhoneNumber
	PhoneNumberVerification       = frameworkviewmodels.PhoneNumberVerification
	Post                          = frameworkviewmodels.Post
	PreferencesData               = frameworkviewmodels.PreferencesData
	PreferencesFormData           = frameworkviewmodels.PreferencesFormData
	PricingPageData               = frameworkviewmodels.PricingPageData
	ProductDescription            = frameworkviewmodels.ProductDescription
	ProfileBioFormData            = frameworkviewmodels.ProfileBioFormData
	ProfileCalendarHeatmap        = frameworkviewmodels.ProfileCalendarHeatmap
	ProfilePageData               = frameworkviewmodels.ProfilePageData
	PushNotificationSubscriptions = frameworkviewmodels.PushNotificationSubscriptions
	QAItem                        = frameworkviewmodels.QAItem
	QuestionInEmail               = frameworkviewmodels.QuestionInEmail
	RegisterData                  = frameworkviewmodels.RegisterData
	RegisterForm                  = frameworkviewmodels.RegisterForm
	ResetPasswordForm             = frameworkviewmodels.ResetPasswordForm
	SearchResult                  = frameworkviewmodels.SearchResult
	SmsVerificationCodeInfo       = frameworkviewmodels.SmsVerificationCodeInfo
	TwoFactorBackupCodesData      = frameworkviewmodels.TwoFactorBackupCodesData
	TwoFactorSetupData            = frameworkviewmodels.TwoFactorSetupData
	TwoFactorSetupForm            = frameworkviewmodels.TwoFactorSetupForm
	TwoFactorVerifyData           = frameworkviewmodels.TwoFactorVerifyData
	TwoFactorVerifyForm           = frameworkviewmodels.TwoFactorVerifyForm
	UpdateInAppModeForm           = frameworkviewmodels.UpdateInAppModeForm
)

var (
	NewCommittedModePageData         = frameworkviewmodels.NewCommittedModePageData
	NewCountByDay                    = frameworkviewmodels.NewCountByDay
	NewCreateCheckoutSessionForm     = frameworkviewmodels.NewCreateCheckoutSessionForm
	NewDeleteAccountData             = frameworkviewmodels.NewDeleteAccountData
	NewDisplayNameForm               = frameworkviewmodels.NewDisplayNameForm
	NewDropdownIterable              = frameworkviewmodels.NewDropdownIterable
	NewEmailDefaultData              = frameworkviewmodels.NewEmailDefaultData
	NewEmailPasswordResetData        = frameworkviewmodels.NewEmailPasswordResetData
	NewEmailUpdate                   = frameworkviewmodels.NewEmailUpdate
	NewForgotPasswordForm            = frameworkviewmodels.NewForgotPasswordForm
	NewHomeFeedButtonsData           = frameworkviewmodels.NewHomeFeedButtonsData
	NewHomeFeedData                  = frameworkviewmodels.NewHomeFeedData
	NewHomeFeedStatsData             = frameworkviewmodels.NewHomeFeedStatsData
	NewLandingPage                   = frameworkviewmodels.NewLandingPage
	NewLocalizationPageData          = frameworkviewmodels.NewLocalizationPageData
	NewLoginForm                     = frameworkviewmodels.NewLoginForm
	NewLoginOAuthData                = frameworkviewmodels.NewLoginOAuthData
	NewLoginOAuthProvider            = frameworkviewmodels.NewLoginOAuthProvider
	NewManagedSettingControl         = frameworkviewmodels.NewManagedSettingControl
	NewNormalNotificationsPageData   = frameworkviewmodels.NewNormalNotificationsPageData
	NewNotificationItem              = frameworkviewmodels.NewNotificationItem
	NewNotificationPermissionsData   = frameworkviewmodels.NewNotificationPermissionsData
	NewPaymentProcessorPublicKey     = frameworkviewmodels.NewPaymentProcessorPublicKey
	NewPhoneNumber                   = frameworkviewmodels.NewPhoneNumber
	NewPhoneNumberVerification       = frameworkviewmodels.NewPhoneNumberVerification
	NewPost                          = frameworkviewmodels.NewPost
	NewPreferencesData               = frameworkviewmodels.NewPreferencesData
	NewPreferencesFormData           = frameworkviewmodels.NewPreferencesFormData
	NewPricingPageData               = frameworkviewmodels.NewPricingPageData
	NewProductDescription            = frameworkviewmodels.NewProductDescription
	NewProfileBioFormData            = frameworkviewmodels.NewProfileBioFormData
	NewProfileCalendarHeatmap        = frameworkviewmodels.NewProfileCalendarHeatmap
	NewProfilePageData               = frameworkviewmodels.NewProfilePageData
	NewPushNotificationSubscriptions = frameworkviewmodels.NewPushNotificationSubscriptions
	NewQAItem                        = frameworkviewmodels.NewQAItem
	NewQuestionInEmail               = frameworkviewmodels.NewQuestionInEmail
	NewRegisterData                  = frameworkviewmodels.NewRegisterData
	NewRegisterForm                  = frameworkviewmodels.NewRegisterForm
	NewResetPasswordForm             = frameworkviewmodels.NewResetPasswordForm
	NewSearchResult                  = frameworkviewmodels.NewSearchResult
	NewSmsVerificationCodeInfo       = frameworkviewmodels.NewSmsVerificationCodeInfo
	NewTwoFactorBackupCodesData      = frameworkviewmodels.NewTwoFactorBackupCodesData
	NewTwoFactorSetupData            = frameworkviewmodels.NewTwoFactorSetupData
	NewTwoFactorSetupForm            = frameworkviewmodels.NewTwoFactorSetupForm
	NewTwoFactorVerifyData           = frameworkviewmodels.NewTwoFactorVerifyData
	NewTwoFactorVerifyForm           = frameworkviewmodels.NewTwoFactorVerifyForm
	NewUpdateInAppModeForm           = frameworkviewmodels.NewUpdateInAppModeForm
)

func NotificationItemsFromDomain(items []*domain.Notification) []NotificationItem {
	return frameworkviewmodels.NotificationItemsFromDomain(items)
}
