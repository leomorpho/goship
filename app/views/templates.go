package templates

type (
	Page string
)

const (
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
	PageAIDemo                 Page = "ai_demo"
	PageIslandsDemo            Page = "islands_demo"

	SSEAnsweredByFriend Page = "sse_answered_by_friend"
)
