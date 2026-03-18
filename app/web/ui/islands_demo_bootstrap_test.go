package ui

import (
	"testing"

	frameworkpage "github.com/leomorpho/goship/framework/web/page"
)

func TestIslandsDemoLabel_UsesBootstrapLabel(t *testing.T) {
	page := &Page{
		Base: frameworkpage.Base{
			Data: IslandsDemoBootstrap{
				Locale: "fr",
				Labels: map[string]string{
					"VanillaCounter": "Compteur Vanilla JS",
				},
			},
		},
	}

	got := IslandsDemoLabel(page, "VanillaCounter", "Vanilla JS Counter")
	if got != "Compteur Vanilla JS" {
		t.Fatalf("label = %q, want %q", got, "Compteur Vanilla JS")
	}
}

func TestIslandsDemoLabel_FallsBackWhenMissing(t *testing.T) {
	page := &Page{
		Base: frameworkpage.Base{
			Data: IslandsDemoBootstrap{
				Labels: map[string]string{},
			},
		},
	}

	got := IslandsDemoLabel(page, "VanillaCounter", "Vanilla JS Counter")
	if got != "Vanilla JS Counter" {
		t.Fatalf("label = %q, want %q", got, "Vanilla JS Counter")
	}
}

func TestIslandsDemoIslandProps_DefaultLocale(t *testing.T) {
	page := &Page{
		Base: frameworkpage.Base{
			Data: IslandsDemoBootstrap{
				Labels: map[string]string{
					"ReactCounter": "React Counter",
				},
			},
		},
	}

	props := IslandsDemoIslandProps(page, "ReactCounter", "React Counter", 10)
	i18n, ok := props["i18n"].(map[string]any)
	if !ok {
		t.Fatalf("i18n payload type = %T, want map[string]any", props["i18n"])
	}
	if locale, _ := i18n["locale"].(string); locale != "en" {
		t.Fatalf("locale = %q, want en", locale)
	}
}
