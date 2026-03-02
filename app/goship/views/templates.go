package templates

type (
	Page string
)

const (
	PageAbout                  Page = "about"
	PageLanding                Page = "landing"
	PageContact                Page = "contact"
	PageError                  Page = "error"
	PageForgotPassword         Page = "forgot-password"
	PageHome                   Page = "home"
	PageLogin                  Page = "login"
	PageRegister               Page = "register"
	PageResetPassword          Page = "reset-password"
	PageEmailSubscribe         Page = "email-subscribe"
	PagePreferences            Page = "preferences"
	PagePhoneNumber            Page = "preferences.phone"
	PageDisplayName            Page = "preferences.display_name"
	PageHomeFeed               Page = "home_feed"
	PageInstallApp             Page = "install_app"
	PageProfile                Page = "profile"
	PageNotifications          Page = "notifications"
	PageHealthcheck            Page = "healthcheck"
	PagePricing                Page = "pricing"
	PageSuccessfullySubscribed Page = "successfully_subscribed"
	PageDeleteAccount          Page = "delete_account.page"
	PagePrivacyPolicy          Page = "privacy_policy"
	PageWiki                   Page = "wiki"

	SSEAnsweredByFriend Page = "sse_answered_by_friend"
)
