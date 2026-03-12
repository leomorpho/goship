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
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	"github.com/leomorpho/goship/app/foundation"
	"github.com/leomorpho/goship/app/web/routenames"
	"github.com/leomorpho/goship/framework/context"
	"github.com/leomorpho/goship/framework/dberrors"
	"github.com/leomorpho/goship/framework/repos/uxflashmessages"
	"log/slog"
)

type profileThumbnailReader interface {
	GetProfilePhotoThumbnailURL(userID int) string
}

// LoadAuthenticatedUser loads the authenticated user, if one, and stores in context
func LoadAuthenticatedUser(
	authClient *foundation.AuthClient, profileService profileThumbnailReader, subscriptionsService *paidsubscriptions.Service,
) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			u, err := authClient.GetAuthenticatedIdentity(c)
			switch {
			case err == nil:
				c.Set(context.AuthenticatedUserIDKey, u.UserID)
				c.Set(context.AuthenticatedUserNameKey, u.UserName)
				c.Set(context.AuthenticatedUserEmailKey, u.UserEmail)
				c.Set(context.AuthenticatedUserIsAdminKey, userIsAdmin(u.UserEmail))
				if u.HasProfile {
					c.Set(context.AuthenticatedProfileIDKey, u.ProfileID)
					c.Set(context.ProfileFullyOnboarded, u.ProfileFullyOnboarded)
				}
				// if subscriptionsService != nil {
				// 	activeProduct, _, err := subscriptionsService.GetCurrentlyActiveProduct(c.Request().Context(), u.Edges.Profile.ID)
				// 	if err != nil {
				// 		log.Error().Err(err).Int("userID", u.ID).Int("profileID", u.Edges.Profile.ID).Msg("failed to get active product in middleware for user")
				// 	}
				// 	c.Set(context.ActiveProductPlan, activeProduct)

				// }
				if profileService != nil {
					// TODO: cache profile photo URL somewhere in the stack
					c.Set(context.AuthenticatedUserProfilePicURL, profileService.GetProfilePhotoThumbnailURL(u.UserID))
				}
				c.Logger().Infof("auth user loaded in to context: %d", u.UserID)
			case dberrors.IsNotFound(err):
				c.Logger().Warn("auth user not found")
			case errors.Is(err, foundation.NotAuthenticatedError{}):
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
func LoadValidPasswordToken(authClient *foundation.AuthClient) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userIDRaw := c.Get(context.AuthenticatedUserIDKey)
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
			case foundation.InvalidPasswordTokenError:
				uxflashmessages.Warning(c, "The link is either invalid or has expired. Please request a new one.")
				// TODO use the const for route name
				return c.Redirect(http.StatusFound, c.Echo().Reverse(routenames.RouteNameForgotPassword))
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
			if c.Get(context.AuthenticatedUserIDKey) == nil {
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
				url := c.Echo().Reverse(routenames.RouteNameLogin)
				return c.Redirect(http.StatusSeeOther, url)
				// Note: leaving original code commented out in case there are unforeseen consequences...so I remember this change which may have caused it...
				// return echo.NewHTTPError(http.StatusUnauthorized)
			}

			return next(c)
		}
	}
}

// RequireNoAuthentication requires that the user not be authenticated in order to proceed
func RequireNoAuthentication() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if u := c.Get(context.AuthenticatedUserIDKey); u != nil {
				url := c.Echo().Reverse("home_feed")
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
			if c.Get(context.AuthenticatedUserIDKey) == nil {
				return echo.NewHTTPError(http.StatusUnauthorized)
			}
			isAdmin, _ := c.Get(context.AuthenticatedUserIsAdminKey).(bool)
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
