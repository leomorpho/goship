package context // TODO: rename this package, it conflicts with the std lib

import (
	"context"
	"errors"
)

const (
	// AuthenticatedUserKey is the key value used to store the authenticated user in context
	AuthenticatedUserKey           = "auth_user"
	AuthenticatedUserProfilePicURL = "profile_pic_url"
	ProfileFullyOnboarded          = "profile_fully_onboarded"
	ActiveProductPlan              = "product_plan"

	// UserKey is the key value used to store a user in context
	UserKey = "user"

	// FormKey is the key value used to store a form in context
	FormKey = "form"

	// PasswordTokenKey is the key value used to store a password token in context
	PasswordTokenKey = "password_token"

	IsFromIOSApp = "is_from_ios_app"
)

// IsCanceledError determines if an error is due to a context cancelation
func IsCanceledError(err error) bool {
	return errors.Is(err, context.Canceled)
}
