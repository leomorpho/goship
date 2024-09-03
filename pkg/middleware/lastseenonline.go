package middleware

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/mikestefanello/pagoda/ent"
	"github.com/mikestefanello/pagoda/pkg/services"
	"github.com/rs/zerolog/log"
)

// LoadAuthenticatedUser loads the authenticated user, if one, and stores in context
func SetLastSeenOnline(authClient *services.AuthClient) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			u, err := authClient.GetAuthenticatedUser(c)
			switch err.(type) {
			case *ent.NotFoundError:
				c.Logger().Warn("auth user not found")
			case services.NotAuthenticatedError:
			case nil:
				err = authClient.SetLastOnlineTimestamp(c, u.ID)
				if err != nil {
					log.Error().Err(err).Msg("failed to set last seen online")
				}
				c.Logger().Infof("last seen timestamp set for user: %d", u.ID)
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
