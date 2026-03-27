package middleware

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/leomorpho/goship/framework/dberrors"
	"github.com/leomorpho/goship/framework/http/authcontext"
	"github.com/leomorpho/goship/modules/authsupport"

	"github.com/labstack/echo/v4"
)

// LoadUser loads the user based on the ID provided as a path parameter
func LoadUser(authClient *authsupport.AuthClient) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID, err := strconv.Atoi(c.Param("user"))
			if err != nil {
				return echo.NewHTTPError(http.StatusNotFound)
			}

			identity, err := authClient.GetIdentityByUserID(c.Request().Context(), userID)
			switch {
			case err == nil:
				c.Set(authcontext.AuthenticatedUserIDKey, identity.UserID)
				c.Set(authcontext.AuthenticatedUserEmailKey, identity.UserEmail)
				return next(c)
			case dberrors.IsNotFound(err):
				return echo.NewHTTPError(http.StatusNotFound)
			default:
				return echo.NewHTTPError(
					http.StatusInternalServerError,
					fmt.Sprintf("error querying user: %v", err),
				)
			}
		}
	}
}
