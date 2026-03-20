package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type Response[T any] struct {
	Data   T          `json:"data"`
	Meta   *Meta      `json:"meta,omitempty"`
	Errors []APIError `json:"errors,omitempty"`
}

type Meta struct {
	Page    int `json:"page,omitempty"`
	PerPage int `json:"per_page,omitempty"`
	Total   int `json:"total,omitempty"`
}

type APIError struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

func OK[T any](c echo.Context, data T) error {
	return c.JSON(http.StatusOK, Response[T]{Data: data})
}

func OKWithMeta[T any](c echo.Context, data T, meta *Meta) error {
	return c.JSON(http.StatusOK, Response[T]{Data: data, Meta: meta})
}

func Fail(c echo.Context, status int, errors ...APIError) error {
	return c.JSON(status, Response[struct{}]{Errors: errors})
}
