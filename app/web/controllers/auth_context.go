package controllers

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	customctx "github.com/leomorpho/goship/framework/context"
)

var errMissingAuthenticatedProfileID = errors.New("authenticated profile id missing from context")
var errMissingAuthenticatedUserEmail = errors.New("authenticated user email missing from context")

func authenticatedProfileID(ctx echo.Context) (int, error) {
	v := ctx.Get(customctx.AuthenticatedProfileIDKey)
	if v == nil {
		return 0, echo.NewHTTPError(http.StatusUnauthorized, errMissingAuthenticatedProfileID.Error())
	}
	profileID, ok := v.(int)
	if !ok || profileID <= 0 {
		return 0, echo.NewHTTPError(http.StatusUnauthorized, errMissingAuthenticatedProfileID.Error())
	}
	return profileID, nil
}

func authenticatedUserEmail(ctx echo.Context) (string, error) {
	v := ctx.Get(customctx.AuthenticatedUserEmailKey)
	email, ok := v.(string)
	if !ok || email == "" {
		return "", echo.NewHTTPError(http.StatusUnauthorized, errMissingAuthenticatedUserEmail.Error())
	}
	return email, nil
}
