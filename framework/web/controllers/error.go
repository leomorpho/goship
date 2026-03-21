package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/framework/context"
	"github.com/leomorpho/goship/framework/web/layouts/gen"
	"github.com/leomorpho/goship/framework/web/pages/gen"
	"github.com/leomorpho/goship/framework/web/templates"
	"github.com/leomorpho/goship/framework/web/ui"
)

type ErrorHandler struct {
	Controller ui.Controller
}

func NewErrorHandler(ctr ui.Controller) ErrorHandler {
	return ErrorHandler{Controller: ctr}
}

func (e *ErrorHandler) Get(err error, ctx echo.Context) {
	if ctx.Response().Committed || context.IsCanceledError(err) {
		return
	}

	code := http.StatusInternalServerError
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
	}

	if code >= 500 {
		ctx.Logger().Error(err)
	} else {
		ctx.Logger().Info(err)
	}

	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Name = templates.PageError
	page.StatusCode = code
	page.HTMX.Request.Enabled = false
	page.HTMX.Request.Boosted = true
	page.Component = pages.Error(&page)

	if err = e.Controller.RenderPage(ctx, page); err != nil {
		ctx.Logger().Error(err)
	}
}

func (e *ErrorHandler) GetHttp400BadRequest(ctx echo.Context) error {
	e.Get(echo.NewHTTPError(http.StatusBadRequest, "Bad Request"), ctx)
	return nil
}

func (e *ErrorHandler) GetHttp401Unauthorized(ctx echo.Context) error {
	e.Get(echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized"), ctx)
	return nil
}

func (e *ErrorHandler) GetHttp403Forbidden(ctx echo.Context) error {
	e.Get(echo.NewHTTPError(http.StatusForbidden, "Forbidden"), ctx)
	return nil
}

func (e *ErrorHandler) GetHttp404NotFound(ctx echo.Context) error {
	e.Get(echo.NewHTTPError(http.StatusNotFound, "Not Found"), ctx)
	return nil
}

func (e *ErrorHandler) GetHttp500InternalServerError(ctx echo.Context) error {
	e.Get(echo.NewHTTPError(http.StatusInternalServerError, "Internal Server Error"), ctx)
	return nil
}
