package routes

import (
	"net/http"

	"github.com/mikestefanello/pagoda/pkg/context"
	"github.com/mikestefanello/pagoda/pkg/controller"
	"github.com/mikestefanello/pagoda/templates"
	"github.com/mikestefanello/pagoda/templates/layouts"
	"github.com/mikestefanello/pagoda/templates/pages"

	"github.com/labstack/echo/v4"
)

type errorHandler struct {
	ctr controller.Controller
}

func NewErrorHandler(ctr controller.Controller) errorHandler {
	return errorHandler{
		ctr: ctr,
	}
}

func (e *errorHandler) Get(err error, ctx echo.Context) {
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

	page := controller.NewPage(ctx)
	// page.Title = http.StatusText(code)
	page.Layout = layouts.Main
	page.Name = templates.PageError
	page.StatusCode = code
	page.HTMX.Request.Enabled = false
	page.HTMX.Request.Boosted = true

	page.Component = pages.Error(&page)

	if err = e.ctr.RenderPage(ctx, page); err != nil {
		ctx.Logger().Error(err)
	}
}

func (e *errorHandler) GetHttp400BadRequest(ctx echo.Context) error {
	e.Get(echo.NewHTTPError(http.StatusBadRequest, "Bad Request"), ctx)
	return nil
}

func (e *errorHandler) GetHttp401Unauthorized(ctx echo.Context) error {
	e.Get(echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized"), ctx)
	return nil
}

func (e *errorHandler) GetHttp403Forbidden(ctx echo.Context) error {
	e.Get(echo.NewHTTPError(http.StatusForbidden, "Forbidden"), ctx)
	return nil
}

func (e *errorHandler) GetHttp404NotFound(ctx echo.Context) error {
	e.Get(echo.NewHTTPError(http.StatusNotFound, "Not Found"), ctx)
	return nil
}

func (e *errorHandler) GetHttp500InternalServerError(ctx echo.Context) error {
	e.Get(echo.NewHTTPError(http.StatusInternalServerError, "Internal Server Error"), ctx)
	return nil
}
