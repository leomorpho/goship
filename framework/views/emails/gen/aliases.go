package emails

import appemails "github.com/leomorpho/goship/app/views/emails/gen"

var (
	EmailUpdate              = appemails.EmailUpdate
	PasswordReset            = appemails.PasswordReset
	RegistrationConfirmation = appemails.RegistrationConfirmation
	SubscriptionConfirmation = appemails.SubscriptionConfirmation
	TestEmail                = appemails.TestEmail
)
