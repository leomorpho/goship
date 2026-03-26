package gen

import (
	"context"
	"io"

	"github.com/a-h/templ"
)

func stubComponent() templ.Component {
	return templ.ComponentFunc(func(context.Context, io.Writer) error {
		return nil
	})
}

func TestEmail(...any) templ.Component                 { return stubComponent() }
func SubscriptionConfirmation(...any) templ.Component { return stubComponent() }
func RegistrationConfirmation(...any) templ.Component { return stubComponent() }
func PasswordReset(...any) templ.Component            { return stubComponent() }
func EmailUpdate(...any) templ.Component              { return stubComponent() }
func WelcomeDigest(...any) templ.Component            { return stubComponent() }
