package i18n

import (
	"context"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	appctx "github.com/leomorpho/goship/framework/context"
	"golang.org/x/text/language"
)

type ProfileLanguageResolver interface {
	PreferredLanguage(ctx context.Context, userID int) (lang string, ok bool, err error)
	SetPreferredLanguage(ctx context.Context, userID int, lang string) error
}

type LanguageService interface {
	DefaultLanguage() string
	NormalizeLanguage(raw string) string
}

func DetectLanguage(service LanguageService, resolver ProfileLanguageResolver) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			lang, setCookie := detectLanguage(c, service, resolver)
			request := c.Request().WithContext(WithLanguage(c.Request().Context(), lang))
			c.SetRequest(request)
			c.Response().Header().Set("Content-Language", lang)
			if setCookie {
				http.SetCookie(c.Response(), &http.Cookie{
					Name:     "lang",
					Value:    lang,
					Path:     "/",
					MaxAge:   365 * 24 * 60 * 60,
					SameSite: http.SameSiteLaxMode,
				})
			}
			return next(c)
		}
	}
}

func detectLanguage(c echo.Context, service LanguageService, resolver ProfileLanguageResolver) (string, bool) {
	if service == nil {
		return defaultLanguage, false
	}

	langFromQuery := strings.TrimSpace(c.QueryParam("lang"))
	if langFromQuery != "" {
		normalized := service.NormalizeLanguage(langFromQuery)
		if resolver != nil {
			if userID, ok := c.Get(appctx.AuthenticatedUserIDKey).(int); ok && userID > 0 {
				_ = resolver.SetPreferredLanguage(c.Request().Context(), userID, normalized)
			}
		}
		return normalized, true
	}

	if resolver != nil {
		if userID, ok := c.Get(appctx.AuthenticatedUserIDKey).(int); ok && userID > 0 {
			if preferred, exists, err := resolver.PreferredLanguage(c.Request().Context(), userID); err == nil && exists {
				return service.NormalizeLanguage(preferred), false
			}
		}
	}

	if cookie, err := c.Cookie("lang"); err == nil && cookie != nil && strings.TrimSpace(cookie.Value) != "" {
		return service.NormalizeLanguage(cookie.Value), false
	}

	if value := strings.TrimSpace(c.Request().Header.Get("Accept-Language")); value != "" {
		if parsed := parseAcceptLanguage(value, service); parsed != "" {
			return parsed, false
		}
	}

	return service.DefaultLanguage(), false
}

func parseAcceptLanguage(value string, service LanguageService) string {
	tags, _, err := language.ParseAcceptLanguage(value)
	if err != nil {
		return service.NormalizeLanguage(value)
	}
	for _, tag := range tags {
		candidate := service.NormalizeLanguage(tag.String())
		if candidate != service.DefaultLanguage() || strings.HasPrefix(strings.ToLower(tag.String()), service.DefaultLanguage()) {
			return candidate
		}
	}
	return service.DefaultLanguage()
}
