package middleware

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/foundation"
	"github.com/leomorpho/goship/framework/dberrors"
	"log/slog"
)

// LoadAuthenticatedUser loads the authenticated user, if one, and stores in context
func SetLastSeenOnline(authClient *foundation.AuthClient) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			u, err := authClient.GetAuthenticatedIdentity(c)
			switch {
			case errors.Is(err, foundation.NotAuthenticatedError{}):
			case err == nil:
				err = authClient.SetLastOnlineTimestamp(c, u.UserID)
				if err != nil {
					slog.Error("failed to set last seen online", "error", err)
				}
				c.Logger().Infof("last seen timestamp set for user: %d", u.UserID)
			default:
				if dberrors.IsNotFound(err) {
					c.Logger().Warn("auth user not found")
					return next(c)
				}
				return echo.NewHTTPError(
					http.StatusInternalServerError,
					fmt.Sprintf("error querying for authenticated user: %v", err),
				)
			}

			return next(c)
		}
	}
}
