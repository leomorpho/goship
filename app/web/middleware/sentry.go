package middleware

import (
	"net/http"

	"github.com/getsentry/sentry-go"
	"github.com/labstack/echo/v4"
)

func FilterSentryErrors(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		err := next(c)
		if err != nil {
			// Log error without forwarding to Sentry if it's a 404
			httpErr, ok := err.(*echo.HTTPError)
			if ok && httpErr.Code == http.StatusNotFound {
				// Log or handle the error as you wish, but don't forward to Sentry
				c.Logger().Error(err)
				return err
			}
			// Forward other errors to Sentry
			sentry.CaptureException(err)
		}
		return err
	}
}
