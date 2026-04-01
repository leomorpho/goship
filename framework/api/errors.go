package api

import (
	"context"

	"github.com/leomorpho/goship/framework/core"
)

type Error struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

func NotFound(message string) Error {
	return Error{Message: message, Code: "not_found"}
}

func Unauthorized(message string) Error {
	return Error{Message: message, Code: "unauthorized"}
}

func Validation(field, message string) Error {
	return Error{Field: field, Message: message, Code: "validation_error"}
}

func NotFoundLocalized(ctx context.Context, i18n core.I18n, key, fallback string) Error {
	return Error{Message: localizedMessage(ctx, i18n, key, fallback), Code: "not_found"}
}

func UnauthorizedLocalized(ctx context.Context, i18n core.I18n, key, fallback string) Error {
	return Error{Message: localizedMessage(ctx, i18n, key, fallback), Code: "unauthorized"}
}

func ValidationLocalized(ctx context.Context, i18n core.I18n, key, fallback string) Error {
	return Error{Message: localizedMessage(ctx, i18n, key, fallback), Code: "validation_error"}
}

func localizedMessage(ctx context.Context, i18n core.I18n, key, fallback string) string {
	if i18n == nil || key == "" {
		return fallback
	}
	message := i18n.T(ctx, key)
	if message == "" {
		return fallback
	}
	return message
}
