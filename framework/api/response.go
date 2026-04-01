package api

import "github.com/labstack/echo/v4"

type successEnvelope struct {
	Data any            `json:"data"`
	Meta map[string]any `json:"meta,omitempty"`
}

type errorEnvelope struct {
	Data   map[string]any `json:"data"`
	Errors []Error        `json:"errors"`
}

func OK(c echo.Context, data any, meta ...map[string]any) error {
	payload := successEnvelope{Data: data}
	if len(meta) > 0 {
		payload.Meta = meta[0]
	}
	return c.JSON(200, payload)
}

func Fail(c echo.Context, status int, errs ...Error) error {
	payload := errorEnvelope{
		Data:   map[string]any{},
		Errors: errs,
	}
	return c.JSON(status, payload)
}
