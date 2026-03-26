package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/framework/appcontext"
	"github.com/leomorpho/goship/framework/web/routenames"
)

func RedirectToOnboardingIfNotComplete() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Get(appcontext.ProfileFullyOnboarded) == nil {
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
			isFullyOnboarded := c.Get(appcontext.ProfileFullyOnboarded).(bool)
			if !isFullyOnboarded {
				url := c.Echo().Reverse(routenames.RouteNamePreferences)
				return c.Redirect(303, url)
			}
			return next(c)
		}
	}
}
