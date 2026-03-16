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

func writeLocaleFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir locale dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write locale file %s: %v", name, err)
	}
}
