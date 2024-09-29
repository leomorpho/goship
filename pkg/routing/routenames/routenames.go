package routeNames

const (
	RouteNameForgotPassword          = "forgot_password"
	RouteNameForgotPasswordSubmit    = "forgot_password.submit"
	RouteNameLogin                   = "login"
	RouteNameLoginSubmit             = "login.submit"
	RouteNameLogout                  = "logout"
	RouteNameRegister                = "register"
	RouteNameRegisterSubmit          = "register.post" // TODO: rename to "register.submit" as was done in pagoda
	RouteNameResetPassword           = "reset_password"
	RouteNameResetPasswordSubmit     = "reset_password.post" // TODO: rename to "register.submit" as was done in pagoda
	RouteNameVerifyEmail             = "verify_email"
	RouteNameContact                 = "contact"
	RouteNameContactSubmit           = "contact.submit"
	RouteNameAboutUs                 = "about"
	RouteNameLandingPage             = "landing_page"
	RouteNameSearch                  = "search"
	RouteNamePreferences             = "preferences"
	RouteNameGetPhone                = "phone.get"
	RouteNameUpdatePhoneNum          = "phone.save"
	RouteNameGetDisplayName          = "display_name.get"
	RouteNameUpdateDisplayName       = "display_name.save"
	RouteNameGetPhoneVerification    = "phone.verification"
	RouteNameSubmitPhoneVerification = "phone.verification.submit"
	RouteNameDeleteAccountPage       = "delete_account.page"
	RouteNameDeleteAccountRequest    = "delete_account.request"
	RouteNamePrivacyPolicy           = "privacy_policy"

	RouteNameHomeFeed           = "home_feed"
	RouteNameGetHomeFeedButtons = "home_feed.buttons"
	RouteNameGetHomeFeedStats   = "home_feed.stats"
	RouteNameSearchQuestion     = "home_feed.search_question"

	RouteNameGetMatch               = "meet.get_match"
	RouteNameRateMatch              = "meet.rate_match"
	RouteNameMatchMadeInHeaven      = "meet.match_made_in_heaven"
	RouteNameProfile                = "profile"
	RouteNameProfileCalendarHeatmap = "profile.calendar_heatmap"
	RouteNameInstallApp             = "install_app"

	RouteNameMarkNotificationsAsRead    = "markNormalNotificationRead"
	RouteNameMarkAllNotificationsAsRead = "normalNotificationsMarkAllAsRead"

	RouteNameRealtime = "realtime"

	RouteNameFinishOnboarding          = "finish_onboarding"
	RouteNameGetGeolocationComponent   = "geolocation"
	RouteNameUpdateGeolocation         = "geolocation.post"
	RouteNameGetBio                    = "profileBio.get"
	RouteNameUpdateBio                 = "profileBio.post"
	RouteNameUpdateSelfGender          = "profileGender.post"
	RouteNameUpdateInterestedInGenders = "profileInterestedGenders.post"
	RouteNameUpdateMinAgeInterestedIn  = "profileMinAgeInterestedIn.post"
	RouteNameUpdateMaxAgeInterestedIn  = "profileMaxAgeInterestedIn.post"

	RouteNameGetPushSubscriptions             = "push_subscriptions.get"
	RouteNameRegisterSubscription             = "notification_subscriptions.register"
	RouteNameDeleteSubscription               = "notification_subscriptions.delete"
	RouteNameDeleteEmailSubscriptionWithToken = "email_subscriptions.delete_with_token"

	RouteNamePaymentProcessorGetPublicKey = "payment_processor.get_public_key"
	RouteNameCreateCheckoutSession        = "stripe.create_checkout_session"
	RouteNameCreatePortalSession          = "stripe.create_portal_session"
	RouteNamePaymentProcessorWebhook      = "stripe.webhook"
	RouteNamePricingPage                  = "pricing_page"
	RouteNamePaymentProcessorSuccess      = "stripe.success"

	RouteNameDocs               = "docs"
	RouteNameDocsGettingStarted = "docs.getting_started"
	RouteNameDocsGuidedTour     = "docs.guided_tour"
	RouteNameDocsArchitecture   = "docs.architecture"
)
