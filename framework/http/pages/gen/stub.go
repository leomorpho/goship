package gen

import (
	"context"
	"io"

	"github.com/a-h/templ"
	"github.com/leomorpho/goship/v2/framework/http/ui"
)

func stubComponent() templ.Component {
	return templ.ComponentFunc(func(context.Context, io.Writer) error {
		return nil
	})
}

func PricingPage(*ui.Page) templ.Component {
	return stubComponent()
}

func PaymentSuccess(*ui.Page) templ.Component {
	return stubComponent()
}

func NotificationsPage(*ui.Page) templ.Component {
	return stubComponent()
}

func NotificationPermissions(*ui.Page, ...any) templ.Component {
	return stubComponent()
}

func Error(*ui.Page) templ.Component {
	return stubComponent()
}
