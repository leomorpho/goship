package middleware

import (
	"log/slog"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/framework/logging"
)

const logKeyRequestID = "request_id"

// RequestID returns a middleware that adds a request ID to the request and response.
// It also adds the request ID to the context for slog.
func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			id := c.Request().Header.Get(echo.HeaderXRequestID)
			if id == "" {
				id = uuid.New().String()
			}
			c.Set(logKeyRequestID, id)
			c.Response().Header().Set(echo.HeaderXRequestID, id)

			// Add to context for slog
			ctx := logging.WithLogger(c.Request().Context(), logging.FromContext(c.Request().Context()).With(slog.String(logKeyRequestID, id)))
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}

// LogRequest is a middleware that logs each request using slog.
// This is often handled by slog-echo, but we can add custom logic here if needed.
func LogRequest(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Logic here if we don't want to use slog-echo
			return next(c)
		}
	}
}
