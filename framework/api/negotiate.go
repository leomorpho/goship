package api

import (
	"strings"

	"github.com/labstack/echo/v4"
)

func IsAPIRequest(c echo.Context) bool {
	if c == nil || c.Request() == nil {
		return false
	}
	path := strings.ToLower(strings.TrimSpace(c.Path()))
	if strings.HasPrefix(path, "/api/") {
		return true
	}
	accept := strings.ToLower(c.Request().Header.Get(echo.HeaderAccept))
	return strings.Contains(accept, "application/json")
}
