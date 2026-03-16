package api

import (
	"context"
	"strings"
)

const (
	ErrorCodeNotFound      = "not_found"
	ErrorCodeUnauthorized  = "unauthorized"
	ErrorCodeValidation    = "validation_error"
	defaultFallbackMessage = "Unexpected error"
)

type Localizer interface {
	T(ctx context.Context, key string, templateData ...map[string]any) string
}

func NotFound(message string) APIError {
	return APIError{Code: ErrorCodeNotFound, Message: message}
}

func Unauthorized(message string) APIError {
	return APIError{Code: ErrorCodeUnauthorized, Message: message}
}

func Validation(field, message string) APIError {
	return APIError{Field: field, Code: ErrorCodeValidation, Message: message}
}

func NotFoundLocalized(ctx context.Context, localizer Localizer, key string, fallback string, templateData ...map[string]any) APIError {
	return APIError{
		Code:    ErrorCodeNotFound,
		Message: resolveLocalizedMessage(ctx, localizer, key, fallback, templateData...),
	}
}

func UnauthorizedLocalized(ctx context.Context, localizer Localizer, key string, fallback string, templateData ...map[string]any) APIError {
	return APIError{
		Code:    ErrorCodeUnauthorized,
		Message: resolveLocalizedMessage(ctx, localizer, key, fallback, templateData...),
	}
}

func ValidationLocalized(ctx context.Context, localizer Localizer, field string, key string, fallback string, templateData ...map[string]any) APIError {
	return APIError{
		Field:   field,
		Code:    ErrorCodeValidation,
		Message: resolveLocalizedMessage(ctx, localizer, key, fallback, templateData...),
	}
}

func resolveLocalizedMessage(ctx context.Context, localizer Localizer, key string, fallback string, templateData ...map[string]any) string {
	normalizedFallback := strings.TrimSpace(fallback)
	if normalizedFallback == "" {
		normalizedFallback = defaultFallbackMessage
	}
	normalizedKey := strings.TrimSpace(key)
	if normalizedKey == "" || localizer == nil {
		return normalizedFallback
	}

	value := strings.TrimSpace(localizer.T(ctx, normalizedKey, templateData...))
	if value == "" || value == normalizedKey {
		return normalizedFallback
	}
	return value
}
