package api

import (
	"strings"

	"github.com/labstack/echo/v4"
)

func IsAPIRequest(c echo.Context) bool {
	accept := strings.ToLower(strings.TrimSpace(c.Request().Header.Get(echo.HeaderAccept)))
	path := strings.TrimSpace(c.Path())
	return strings.Contains(accept, "application/json") || strings.HasPrefix(path, "/api/")
}
