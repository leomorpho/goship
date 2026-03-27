package authcontext

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
)

const (
	AuthenticatedUserIDKey         = "auth_user_id"
	AuthenticatedUserNameKey       = "auth_user_name"
	AuthenticatedUserEmailKey      = "auth_user_email"
	AuthenticatedUserIsAdminKey    = "auth_user_is_admin"
	AuthenticatedProfileIDKey      = "auth_profile_id"
	ProfileFullyOnboarded          = "profile_fully_onboarded"
)

var errMissingAuthenticatedProfileID = errors.New("authenticated profile id missing from context")
var errMissingAuthenticatedUserEmail = errors.New("authenticated user email missing from context")

func AuthenticatedProfileID(ctx echo.Context) (int, error) {
	v := ctx.Get(AuthenticatedProfileIDKey)
	if v == nil {
		return 0, echo.NewHTTPError(http.StatusUnauthorized, errMissingAuthenticatedProfileID.Error())
	}
	profileID, ok := v.(int)
	if !ok || profileID <= 0 {
		return 0, echo.NewHTTPError(http.StatusUnauthorized, errMissingAuthenticatedProfileID.Error())
	}
	return profileID, nil
}

func AuthenticatedUserEmail(ctx echo.Context) (string, error) {
	v := ctx.Get(AuthenticatedUserEmailKey)
	email, ok := v.(string)
	if !ok || email == "" {
		return "", echo.NewHTTPError(http.StatusUnauthorized, errMissingAuthenticatedUserEmail.Error())
	}
	return email, nil
}
