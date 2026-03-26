package gen

import (
	"context"
	"io"

	"github.com/a-h/templ"
	"github.com/leomorpho/goship/framework/web/ui"
)

func stubComponent() templ.Component {
	return templ.ComponentFunc(func(context.Context, io.Writer) error {
		return nil
	})
}

func PricingPage(page *ui.Page) templ.Component {
	_ = page
	return stubComponent()
}

func PaymentSuccess(page *ui.Page) templ.Component {
	_ = page
	return stubComponent()
}

func DeleteAccount(page *ui.Page) templ.Component {
	_ = page
	return stubComponent()
}

func DeleteAccountPage(page *ui.Page) templ.Component {
	_ = page
	return stubComponent()
}

func Error(page *ui.Page) templ.Component {
	_ = page
	return stubComponent()
}

func ErrorPage(page *ui.Page) templ.Component {
	_ = page
	return stubComponent()
}

func HomeFeed(page *ui.Page) templ.Component {
	_ = page
	return stubComponent()
}

func HomeFeedButtons(page *ui.Page) templ.Component {
	_ = page
	return stubComponent()
}

func HomeFeedPage(page *ui.Page) templ.Component {
	_ = page
	return stubComponent()
}

func LandingPage(page *ui.Page) templ.Component {
	_ = page
	return stubComponent()
}

func NotificationsPage(page *ui.Page) templ.Component {
	_ = page
	return stubComponent()
}

func NotificationPermissions(page *ui.Page, _ ...any) templ.Component {
	_ = page
	return stubComponent()
}

func PaymentsPage(page *ui.Page) templ.Component {
	_ = page
	return stubComponent()
}

func DisplayName(page *ui.Page) templ.Component {
	_ = page
	return stubComponent()
}

func EditPhonePage(page *ui.Page) templ.Component {
	_ = page
	return stubComponent()
}

func Phone(page *ui.Page) templ.Component {
	_ = page
	return stubComponent()
}

func PhoneVerificationField(page *ui.Page) templ.Component {
	_ = page
	return stubComponent()
}

func Settings(page *ui.Page) templ.Component {
	_ = page
	return stubComponent()
}

func AboutMe(page *ui.Page) templ.Component {
	_ = page
	return stubComponent()
}

func Setup(page *ui.Page) templ.Component {
	_ = page
	return stubComponent()
}

func BackupCodes(page *ui.Page) templ.Component {
	_ = page
	return stubComponent()
}

func Verify(page *ui.Page) templ.Component {
	_ = page
	return stubComponent()
}

func Register(page *ui.Page) templ.Component {
	_ = page
	return stubComponent()
}

func ForgotPassword(page *ui.Page) templ.Component {
	_ = page
	return stubComponent()
}

func ResetPassword(page *ui.Page) templ.Component {
	_ = page
	return stubComponent()
}

func Login(page *ui.Page) templ.Component {
	_ = page
	return stubComponent()
}

func ProfilePage(page *ui.Page) templ.Component {
	_ = page
	return stubComponent()
}

func InstallApp(page *ui.Page) templ.Component {
	_ = page
	return stubComponent()
}

func PreferencesPage(page *ui.Page) templ.Component {
	_ = page
	return stubComponent()
}
