package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/framework/dberrors"
	"github.com/leomorpho/goship/framework/flash"
	"github.com/leomorpho/goship/framework/http/authcontext"
	"github.com/leomorpho/goship/modules/authsupport"
	"log/slog"
)

// LoadAuthenticatedUser loads the authenticated user, if one, and stores in context
func LoadAuthenticatedUser(authClient *authsupport.AuthClient) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			u, err := authClient.GetAuthenticatedIdentity(c)
			switch {
			case err == nil:
				c.Set(authcontext.AuthenticatedUserIDKey, u.UserID)
				c.Set(authcontext.AuthenticatedUserNameKey, u.UserName)
				c.Set(authcontext.AuthenticatedUserEmailKey, u.UserEmail)
				c.Set(authcontext.AuthenticatedUserIsAdminKey, userIsAdmin(u.UserEmail))
				if u.HasProfile {
					c.Set(authcontext.AuthenticatedProfileIDKey, u.ProfileID)
					c.Set(authcontext.ProfileFullyOnboarded, u.ProfileFullyOnboarded)
				}
				c.Logger().Infof("auth user loaded in to context: %d", u.UserID)
			case dberrors.IsNotFound(err):
				c.Logger().Warn("auth user not found")
			case errors.Is(err, authsupport.NotAuthenticatedError{}):
			default:
				return echo.NewHTTPError(
					http.StatusInternalServerError,
					fmt.Sprintf("error querying for authenticated user: %v", err),
				)
			}

			return next(c)
		}
	}
}

// LoadValidPasswordToken loads a valid password token entity that matches the user and token
// provided in path parameters
// If the token is invalid, the user will be redirected to the forgot password route
// This requires that the user ID owning the token is loaded in context
// (e.g. by LoadUser middleware).
func LoadValidPasswordToken(authClient *authsupport.AuthClient) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userIDRaw := c.Get(authcontext.AuthenticatedUserIDKey)
			userID, ok := userIDRaw.(int)
			if !ok || userID <= 0 {
				return echo.NewHTTPError(http.StatusInternalServerError)
			}

			// Extract the token ID
			tokenID, err := strconv.Atoi(c.Param("password_token"))
			if err != nil {
				return echo.NewHTTPError(http.StatusNotFound)
			}

			// Attempt to load a valid password token
			err = authClient.GetValidPasswordToken(
				c,
				userID,
				tokenID,
				c.Param("token"),
			)

			switch err.(type) {
			case nil:
				return next(c)
			case authsupport.InvalidPasswordTokenError:
				uxflashmessages.Warning(c, "The link is either invalid or has expired. Please request a new one.")
				// TODO use the const for route name
				return c.Redirect(http.StatusFound, c.Echo().Reverse("forgot_password"))
			default:
				return echo.NewHTTPError(
					http.StatusInternalServerError,
					fmt.Sprintf("error loading password token: %v", err),
				)
			}
		}
	}
}

// RequireAuthentication requires that the user be authenticated in order to proceed
func RequireAuthentication() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Get(authcontext.AuthenticatedUserIDKey) == nil {
				// Get the session
				sess, err := session.Get("session", c)
				if err != nil {
					slog.Error("failed to open session to save redirectAfterLogin URL to it", "error", err)
				} else {
					// Store the original URL they were trying to access
					currentURL := c.Request().RequestURI
					sess.Values["redirectAfterLogin"] = currentURL
					sess.Save(c.Request(), c.Response())

				}

				// Redirect to login page
				url := c.Echo().Reverse("login")
				return c.Redirect(http.StatusSeeOther, url)
			}

			return next(c)
		}
	}
}

// RequireNoAuthentication requires that the user not be authenticated in order to proceed
func RequireNoAuthentication() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if u := c.Get(authcontext.AuthenticatedUserIDKey); u != nil {
				url := c.Echo().Reverse("landing_page")
				return c.Redirect(http.StatusSeeOther, url)
			}

			return next(c)
		}
	}
}

// RequireAdmin requires an authenticated admin user.
func RequireAdmin() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Get(authcontext.AuthenticatedUserIDKey) == nil {
				return echo.NewHTTPError(http.StatusUnauthorized)
			}
			isAdmin, _ := c.Get(authcontext.AuthenticatedUserIsAdminKey).(bool)
			if !isAdmin {
				return echo.NewHTTPError(http.StatusForbidden, "admin access required")
			}
			return next(c)
		}
	}
}

func userIsAdmin(email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return false
	}
	raw := strings.TrimSpace(os.Getenv("PAGODA_ADMIN_EMAILS"))
	if raw == "" {
		return false
	}
	for _, candidate := range strings.Split(raw, ",") {
		if strings.ToLower(strings.TrimSpace(candidate)) == email {
			return true
		}
	}
	return false
}
