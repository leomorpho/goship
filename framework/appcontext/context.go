package appcontext

import (
	"context"
	"errors"
)

const (
	AuthenticatedUserIDKey         = "auth_user_id"
	AuthenticatedUserNameKey       = "auth_user_name"
	AuthenticatedUserEmailKey      = "auth_user_email"
	AuthenticatedUserIsAdminKey    = "auth_user_is_admin"
	AuthenticatedProfileIDKey      = "auth_profile_id"
	AuthenticatedUserProfilePicURL = "profile_pic_url"
	ProfileFullyOnboarded          = "profile_fully_onboarded"
	ActiveProductPlan              = "product_plan"

	// FormKey is the key value used to store a form in context
	FormKey = "form"

	IsFromIOSApp = "is_from_ios_app"
)

// IsCanceledError determines if an error is due to a context cancelation
func IsCanceledError(err error) bool {
	return errors.Is(err, context.Canceled)
}
