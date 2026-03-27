package middleware

import (
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	gommonlog "github.com/labstack/gommon/log"
)

// RecoverPanics wraps Echo's recovery middleware and emits structured panic logs.
func RecoverPanics(logger echo.Logger) echo.MiddlewareFunc {
	return echomw.RecoverWithConfig(echomw.RecoverConfig{
		LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
			fields := gommonlog.JSON{
				"error":  err.Error(),
				"stack":  string(stack),
				"method": c.Request().Method,
				"path":   requestPath(c),
			}

			target := c.Logger()
			if logger != nil {
				logger.Errorj(fields)
			} else {
				target.Errorj(fields)
			}
			return err
		},
	})
}

func requestPath(c echo.Context) string {
	if c == nil || c.Request() == nil || c.Request().URL == nil {
		return ""
	}
	if path := c.Path(); path != "" {
		return path
	}
	return c.Request().URL.Path
}
