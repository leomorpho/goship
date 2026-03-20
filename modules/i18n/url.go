package i18n

import (
	"net/url"
	"strings"
)

// LanguageSwitcherURL preserves current route/query while switching the `lang` query value.
func LanguageSwitcherURL(currentURL string, lang string) string {
	targetLang := strings.TrimSpace(lang)
	if targetLang == "" {
		targetLang = defaultLanguage
	}

	parsed, err := url.Parse(strings.TrimSpace(currentURL))
	if err != nil {
		parsed = &url.URL{Path: "/"}
	}
	if parsed.Path == "" {
		parsed.Path = "/"
	}

	query := parsed.Query()
	query.Set("lang", targetLang)
	parsed.RawQuery = query.Encode()
	parsed.Fragment = ""
	return parsed.RequestURI()
}
