package middleware

import (
	"github.com/mikestefanello/pagoda/pkg/context"

	"github.com/labstack/echo/v4"
)

// LoadAuthenticatedUser loads the authenticated user, if one, and stores in context
func SetDeviceTypeToServe() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {

			// Check for `app-platform` cookie
			appPlatformCookie, err := c.Cookie("app-platform")
			var isiOSApp bool
			if err == nil && appPlatformCookie != nil {
				isiOSApp = appPlatformCookie.Value == "iOS App Store"
			}
			c.Set(context.IsFromIOSApp, isiOSApp)

			return next(c)
		}
	}
}
