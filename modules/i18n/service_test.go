package i18n

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestServiceTranslateByContextLanguage(t *testing.T) {
	localeDir := t.TempDir()
	writeLocaleFile(t, localeDir, "en.yaml", `
auth:
  login:
    title: "Sign in to your account"
`)
	writeLocaleFile(t, localeDir, "fr.yaml", `
auth:
  login:
    title: "Connectez-vous a votre compte"
`)

	service, err := NewService(Options{
		LocaleDir:       localeDir,
		DefaultLanguage: "en",
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	en := service.T(context.Background(), "auth.login.title")
	if en != "Sign in to your account" {
		t.Fatalf("english translation = %q", en)
	}

	frCtx := WithLanguage(context.Background(), "fr")
	fr := service.T(frCtx, "auth.login.title")
	if fr != "Connectez-vous a votre compte" {
		t.Fatalf("french translation = %q", fr)
	}

	fallback := service.T(WithLanguage(context.Background(), "fr-CA"), "auth.login.title")
	if fallback != "Connectez-vous a votre compte" {
		t.Fatalf("regional translation fallback = %q", fallback)
	}
}

func TestServiceMissingKeyReturnsKey(t *testing.T) {
	localeDir := t.TempDir()
	writeLocaleFile(t, localeDir, "en.yaml", `
auth:
  login:
    title: "Sign in"
`)

	service, err := NewService(Options{LocaleDir: localeDir, DefaultLanguage: "en"})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	got := service.T(context.Background(), "auth.login.unknown")
	if got != "auth.login.unknown" {
		t.Fatalf("missing key translation = %q", got)
	}
}

func TestServiceSupportsTOMLLocaleFiles(t *testing.T) {
	localeDir := t.TempDir()
	writeLocaleFile(t, localeDir, "en.toml", `
"auth.login.title" = "Sign in to your account"
`)
	writeLocaleFile(t, localeDir, "fr.toml", `
"auth.login.title" = "Connectez-vous a votre compte"
`)

	service, err := NewService(Options{LocaleDir: localeDir, DefaultLanguage: "en"})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	if got := service.T(WithLanguage(context.Background(), "fr"), "auth.login.title"); got != "Connectez-vous a votre compte" {
		t.Fatalf("french translation = %q", got)
	}
}

func TestServicePrefersTOMLWhenBothTOMLAndYAMLExist(t *testing.T) {
	localeDir := t.TempDir()
	writeLocaleFile(t, localeDir, "en.yaml", `
auth:
  login:
    title: "English from yaml"
`)
	writeLocaleFile(t, localeDir, "en.toml", `
"auth.login.title" = "English from toml"
`)

	service, err := NewService(Options{LocaleDir: localeDir, DefaultLanguage: "en"})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	if got := service.T(context.Background(), "auth.login.title"); got != "English from toml" {
		t.Fatalf("translation = %q, want toml value", got)
	}
}

func TestServicePluralTranslationWithCountHelper(t *testing.T) {
	localeDir := t.TempDir()
	writeLocaleFile(t, localeDir, "en.toml", `
"cart.items.one" = "You have {{.Count}} item"
"cart.items.other" = "You have {{.Count}} items"
`)

	service, err := NewService(Options{LocaleDir: localeDir, DefaultLanguage: "en"})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	cases := []struct {
		count int
		want  string
	}{
		{count: 0, want: "You have 0 items"},
		{count: 1, want: "You have 1 item"},
		{count: 2, want: "You have 2 items"},
		{count: 5, want: "You have 5 items"},
		{count: 11, want: "You have 11 items"},
		{count: 21, want: "You have 21 items"},
	}
	for _, tc := range cases {
		if got := service.TC(context.Background(), "cart.items", tc.count); got != tc.want {
			t.Fatalf("count=%d translation = %q, want %q", tc.count, got, tc.want)
		}
	}
}

func TestServicePluralFallbackToDefaultLocale(t *testing.T) {
	localeDir := t.TempDir()
	writeLocaleFile(t, localeDir, "en.toml", `
"notifications.count.one" = "{{.Count}} notification"
"notifications.count.other" = "{{.Count}} notifications"
`)
	writeLocaleFile(t, localeDir, "fr.toml", `
"notifications.count.other" = "{{.Count}} notifications FR"
`)

	service, err := NewService(Options{LocaleDir: localeDir, DefaultLanguage: "en"})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	frCtx := WithLanguage(context.Background(), "fr")
	if got := service.TC(frCtx, "notifications.count", 1); got != "1 notification" {
		t.Fatalf("count=1 fallback translation = %q", got)
	}
	if got := service.TC(frCtx, "notifications.count", 3); got != "3 notifications FR" {
		t.Fatalf("count=3 localized translation = %q", got)
	}
}

func TestServiceSelectHelperUsesChoiceThenOtherFallback(t *testing.T) {
	localeDir := t.TempDir()
	writeLocaleFile(t, localeDir, "en.toml", `
"profile.role.admin" = "Administrator"
"profile.role.member" = "Member"
"profile.role.other" = "User"
`)

	service, err := NewService(Options{LocaleDir: localeDir, DefaultLanguage: "en"})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	if got := service.TS(context.Background(), "profile.role", "admin"); got != "Administrator" {
		t.Fatalf("admin translation = %q", got)
	}
	if got := service.TS(context.Background(), "profile.role", "unknown"); got != "User" {
		t.Fatalf("unknown choice translation = %q", got)
	}
}

func writeLocaleFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir locale dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write locale file %s: %v", name, err)
	}
}
