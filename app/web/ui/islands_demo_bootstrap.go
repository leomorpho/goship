package ui

import "strings"

const islandsDemoDefaultLocale = "en"

type IslandsDemoBootstrap struct {
	Locale string
	Labels map[string]string
}

func IslandsDemoLabel(page *Page, islandName string, fallback string) string {
	bootstrap := islandsDemoBootstrap(page)
	if bootstrap.Labels != nil {
		if label := strings.TrimSpace(bootstrap.Labels[islandName]); label != "" {
			return label
		}
	}
	return fallback
}

func IslandsDemoIslandProps(page *Page, islandName string, fallback string, initialCount int) map[string]any {
	label := IslandsDemoLabel(page, islandName, fallback)
	bootstrap := islandsDemoBootstrap(page)
	locale := strings.TrimSpace(bootstrap.Locale)
	if locale == "" {
		locale = islandsDemoDefaultLocale
	}

	return map[string]any{
		"initialCount": initialCount,
		"label":        label,
		"i18n": map[string]any{
			"locale": locale,
			"messages": map[string]string{
				"label": label,
			},
		},
	}
}

func islandsDemoBootstrap(page *Page) IslandsDemoBootstrap {
	if page == nil || page.Data == nil {
		return IslandsDemoBootstrap{}
	}
	switch typed := page.Data.(type) {
	case IslandsDemoBootstrap:
		return typed
	case *IslandsDemoBootstrap:
		if typed == nil {
			return IslandsDemoBootstrap{}
		}
		return *typed
	default:
		return IslandsDemoBootstrap{}
	}
}
