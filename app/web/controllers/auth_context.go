package controllers

import (
	"github.com/labstack/echo/v4"
	frameworkauthcontext "github.com/leomorpho/goship/framework/web/authcontext"
)

func authenticatedProfileID(ctx echo.Context) (int, error) {
	return frameworkauthcontext.AuthenticatedProfileID(ctx)
}

func authenticatedUserEmail(ctx echo.Context) (string, error) {
	return frameworkauthcontext.AuthenticatedUserEmail(ctx)
}
