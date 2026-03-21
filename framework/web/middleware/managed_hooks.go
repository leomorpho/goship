package middleware

import (
	"bytes"
	"errors"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	frameworksecurity "github.com/leomorpho/goship/framework/security"
)

// RequireManagedHookSignature enforces HMAC signatures and replay protection for managed hooks.
func RequireManagedHookSignature(verifier *frameworksecurity.ManagedHookVerifier) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			body, err := io.ReadAll(c.Request().Body)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "failed to read request body")
			}
			c.Request().Body = io.NopCloser(bytes.NewReader(body))

			if err := verifier.VerifyRequest(c.Request(), body); err != nil {
				status := http.StatusUnauthorized
				switch {
				case errors.Is(err, frameworksecurity.ErrManagedReplayDetected):
					status = http.StatusConflict
				case errors.Is(err, frameworksecurity.ErrManagedSecretNotConfigured):
					status = http.StatusServiceUnavailable
				}

				return c.JSON(status, map[string]string{
					"error": err.Error(),
				})
			}

			return next(c)
		}
	}
}
