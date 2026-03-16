package controllers_test

import (
	"bytes"
	"encoding/json"
	"html"
	"io"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/leomorpho/goship/framework/testutil"
)

func TestIslandsDemoBootstrapContract_DefaultLocale(t *testing.T) {
	s := testutil.NewTestServer(t)
	resp := s.Get("/demo/islands").AssertStatus(200)

	props, doc := decodeIslandProps(t, resp, "VanillaCounter")
	locale := stringField(t, nestedMap(t, props, "i18n"), "locale")
	if locale != "en" {
		t.Fatalf("i18n locale = %q, want en", locale)
	}
	label := stringField(t, nestedMap(t, nestedMap(t, props, "i18n"), "messages"), "label")
	if label != "Vanilla JS Counter" {
		t.Fatalf("i18n label = %q, want Vanilla JS Counter", label)
	}

	if got := doc.Find(`[data-island="VanillaCounter"] h3`).First().Text(); got != label {
		t.Fatalf("server fallback heading = %q, want %q", got, label)
	}
}

func TestIslandsDemoBootstrapContract_FrenchLocaleSwitch(t *testing.T) {
	s := testutil.NewTestServer(t)
	resp := s.Get("/demo/islands?lang=fr").AssertStatus(200)

	props, doc := decodeIslandProps(t, resp, "VanillaCounter")
	locale := stringField(t, nestedMap(t, props, "i18n"), "locale")
	if locale != "fr" {
		t.Fatalf("i18n locale = %q, want fr", locale)
	}
	label := stringField(t, nestedMap(t, nestedMap(t, props, "i18n"), "messages"), "label")
	if label != "Compteur Vanilla JS" {
		t.Fatalf("i18n label = %q, want Compteur Vanilla JS", label)
	}

	if got := doc.Find(`[data-island="VanillaCounter"] h3`).First().Text(); got != label {
		t.Fatalf("server fallback heading = %q, want %q", got, label)
	}
}

func decodeIslandProps(t *testing.T, resp *testutil.TestResponse, islandName string) (map[string]any, *goquery.Document) {
	t.Helper()
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("parse response HTML: %v", err)
	}

	raw, ok := doc.Find(`[data-island="` + islandName + `"]`).First().Attr("data-props")
	if !ok {
		t.Fatalf("missing data-props for island %s", islandName)
	}

	var props map[string]any
	if err := json.Unmarshal([]byte(html.UnescapeString(raw)), &props); err != nil {
		t.Fatalf("parse island props JSON: %v", err)
	}
	return props, doc
}

func nestedMap(t *testing.T, input map[string]any, key string) map[string]any {
	t.Helper()
	value, ok := input[key]
	if !ok {
		t.Fatalf("missing key %q in map %#v", key, input)
	}
	typed, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("key %q is %T, want map[string]any", key, value)
	}
	return typed
}

func stringField(t *testing.T, input map[string]any, key string) string {
	t.Helper()
	value, ok := input[key]
	if !ok {
		t.Fatalf("missing key %q in map %#v", key, input)
	}
	typed, ok := value.(string)
	if !ok {
		t.Fatalf("key %q is %T, want string", key, value)
	}
	return typed
}
