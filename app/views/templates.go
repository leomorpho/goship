package templates

type (
	Page string
)

const (
	PageLanding                Page = "landing"
	PageError                  Page = "error"
	PageForgotPassword         Page = "forgot-password"
	PageHome                   Page = "home"
	PageLogin                  Page = "login"
	PageRegister               Page = "register"
	PageResetPassword          Page = "reset-password"
	PagePreferences            Page = "preferences"
	PagePhoneNumber            Page = "preferences.phone"
	PageDisplayName            Page = "preferences.display_name"
	PageHomeFeed               Page = "home_feed"
	PageInstallApp             Page = "install_app"
	PageProfile                Page = "profile"
	PageNotifications          Page = "notifications"
	PagePricing                Page = "pricing"
	PageSuccessfullySubscribed Page = "successfully_subscribed"
	PageDeleteAccount          Page = "delete_account.page"

	SSEAnsweredByFriend Page = "sse_answered_by_friend"
)
