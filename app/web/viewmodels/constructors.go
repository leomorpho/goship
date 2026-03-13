package viewmodels

func NewAboutData() AboutData {
	return AboutData{}
}

func NewCapabilityDocLink() CapabilityDocLink {
	return CapabilityDocLink{}
}

func NewCapabilitySection() CapabilitySection {
	return CapabilitySection{
		Docs: []CapabilityDocLink{},
	}
}

func NewDropdownIterable() DropdownIterable {
	return DropdownIterable{}
}

func NewCommittedModePageData() CommittedModePageData {
	return CommittedModePageData{
		Friends: []DropdownIterable{},
	}
}

func NewUpdateInAppModeForm() *UpdateInAppModeForm {
	return &UpdateInAppModeForm{}
}

func NewContactForm() *ContactForm {
	return &ContactForm{}
}

func NewEmailSubscriptionData() EmailSubscriptionData {
	return EmailSubscriptionData{}
}

func NewEmailSubscriptionForm() *EmailSubscriptionForm {
	return &EmailSubscriptionForm{}
}

func NewEmailDefaultData() EmailDefaultData {
	return EmailDefaultData{}
}

func NewEmailPasswordResetData() EmailPasswordResetData {
	return EmailPasswordResetData{}
}

func NewQuestionInEmail() QuestionInEmail {
	return QuestionInEmail{}
}

func NewEmailUpdate() EmailUpdate {
	return EmailUpdate{
		QuestionsAnsweredByFriendButNotSelf: []QuestionInEmail{},
		QuestionsNotAnsweredInSocialCircle:  []QuestionInEmail{},
	}
}

func NewForgotPasswordForm() *ForgotPasswordForm {
	return &ForgotPasswordForm{}
}

func NewPost() Post {
	return Post{}
}

func NewHomeFeedData() HomeFeedData {
	return HomeFeedData{}
}

func NewHomeFeedButtonsData() HomeFeedButtonsData {
	return HomeFeedButtonsData{}
}

func NewHomeFeedStatsData() HomeFeedStatsData {
	return HomeFeedStatsData{}
}

func NewInvitationsData() InvitationsData {
	return InvitationsData{}
}

func NewLandingPage() LandingPage {
	return LandingPage{
		QAItems: []QAItem{},
	}
}

func NewQAItem() QAItem {
	return QAItem{}
}

func NewLoginForm() *LoginForm {
	return &LoginForm{}
}

func NewLoginOAuthProvider() LoginOAuthProvider {
	return LoginOAuthProvider{}
}

func NewLoginOAuthData() LoginOAuthData {
	return LoginOAuthData{
		Providers: []LoginOAuthProvider{},
	}
}

func NewNotificationItem() NotificationItem {
	return NotificationItem{}
}

func NewNormalNotificationsPageData() NormalNotificationsPageData {
	return NormalNotificationsPageData{
		Notifications: []NotificationItem{},
	}
}

func NewPaymentProcessorPublicKey() PaymentProcessorPublicKey {
	return PaymentProcessorPublicKey{}
}

func NewCreateCheckoutSessionForm() *CreateCheckoutSessionForm {
	return &CreateCheckoutSessionForm{}
}

func NewProductDescription() ProductDescription {
	return ProductDescription{
		Points: []string{},
	}
}

func NewPricingPageData() PricingPageData {
	return PricingPageData{
		ProductDescriptions: []ProductDescription{},
	}
}

func NewPreferencesData() PreferencesData {
	return PreferencesData{}
}

func NewDeleteAccountData() *DeleteAccountData {
	return &DeleteAccountData{}
}

func NewNotificationPermissionsData() NotificationPermissionsData {
	return NotificationPermissionsData{
		SubscribedEndpoints: []string{},
	}
}

func NewPushNotificationSubscriptions() PushNotificationSubscriptions {
	return PushNotificationSubscriptions{
		URLs: []string{},
	}
}

func NewPreferencesFormData() *PreferencesFormData {
	return &PreferencesFormData{}
}

func NewProfileBioFormData() *ProfileBioFormData {
	return &ProfileBioFormData{}
}

func NewPhoneNumber() *PhoneNumber {
	return &PhoneNumber{}
}

func NewPhoneNumberVerification() *PhoneNumberVerification {
	return &PhoneNumberVerification{}
}

func NewSmsVerificationCodeInfo() *SmsVerificationCodeInfo {
	return &SmsVerificationCodeInfo{}
}

func NewDisplayNameForm() *DisplayNameForm {
	return &DisplayNameForm{}
}

func NewProfilePageData() ProfilePageData {
	return ProfilePageData{}
}

func NewProfileCalendarHeatmap() ProfileCalendarHeatmap {
	return ProfileCalendarHeatmap{
		Counts: []CountByDay{},
	}
}

func NewCountByDay() CountByDay {
	return CountByDay{}
}

func NewLocalizationPageData() LocalizationPageData {
	return LocalizationPageData{}
}

func NewRegisterForm() *RegisterForm {
	return &RegisterForm{}
}

func NewRegisterData() RegisterData {
	return RegisterData{}
}

func NewResetPasswordForm() *ResetPasswordForm {
	return &ResetPasswordForm{}
}

func NewSearchResult() SearchResult {
	return SearchResult{}
}
