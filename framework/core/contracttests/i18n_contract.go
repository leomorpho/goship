package contracttests

import (
	"context"
	"strings"
	"testing"
)

// I18nContractAdapter is the minimal adapter surface required by core i18n consumers.
type I18nContractAdapter interface {
	DefaultLanguage() string
	SupportedLanguages() []string
	NormalizeLanguage(raw string) string
	T(ctx context.Context, key string, templateData ...map[string]any) string
	TC(ctx context.Context, key string, count any, templateData ...map[string]any) string
	TS(ctx context.Context, key string, choice string, templateData ...map[string]any) string
}

// I18nContractSubject describes one adapter implementation to validate.
type I18nContractSubject struct {
	Name               string
	Build              func(t *testing.T) I18nContractAdapter
	KnownDefaultKey    string
	KnownDefaultResult string
}

// RunI18nContract executes shared i18n contract checks for adapter implementations.
func RunI18nContract(t *testing.T, subject I18nContractSubject) {
	t.Helper()

	if subject.Build == nil {
		t.Fatalf("subject.Build is required")
	}
	if strings.TrimSpace(subject.Name) == "" {
		t.Fatalf("subject.Name is required")
	}

	adapter := subject.Build(t)
	if adapter == nil {
		t.Fatalf("%s: Build returned nil adapter", subject.Name)
	}

	t.Run("default language is non-empty", func(t *testing.T) {
		if strings.TrimSpace(adapter.DefaultLanguage()) == "" {
			t.Fatalf("%s: DefaultLanguage must not be empty", subject.Name)
		}
	})

	t.Run("supported languages include default", func(t *testing.T) {
		defaultLang := adapter.DefaultLanguage()
		supported := adapter.SupportedLanguages()
		found := false
		for _, lang := range supported {
			if strings.EqualFold(strings.TrimSpace(lang), strings.TrimSpace(defaultLang)) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("%s: default language %q missing from SupportedLanguages %v", subject.Name, defaultLang, supported)
		}
	})

	t.Run("normalize unsupported language falls back to default", func(t *testing.T) {
		if got := adapter.NormalizeLanguage("zz-XX"); got != adapter.DefaultLanguage() {
			t.Fatalf("%s: NormalizeLanguage fallback = %q, want %q", subject.Name, got, adapter.DefaultLanguage())
		}
	})

	t.Run("missing key fallback returns key", func(t *testing.T) {
		const missingKey = "__contract_missing_key__"
		if got := adapter.T(context.Background(), missingKey); got != missingKey {
			t.Fatalf("%s: missing key fallback = %q, want %q", subject.Name, got, missingKey)
		}
		if got := adapter.TC(context.Background(), missingKey, 1); got != missingKey {
			t.Fatalf("%s: missing plural key fallback = %q, want %q", subject.Name, got, missingKey)
		}
		if got := adapter.TS(context.Background(), missingKey, "admin"); got != missingKey {
			t.Fatalf("%s: missing select key fallback = %q, want %q", subject.Name, got, missingKey)
		}
	})

	if strings.TrimSpace(subject.KnownDefaultKey) != "" {
		t.Run("known key resolves in default language", func(t *testing.T) {
			got := adapter.T(context.Background(), subject.KnownDefaultKey)
			if strings.TrimSpace(got) != strings.TrimSpace(subject.KnownDefaultResult) {
				t.Fatalf("%s: known key %q = %q, want %q", subject.Name, subject.KnownDefaultKey, got, subject.KnownDefaultResult)
			}
		})
	}
}
