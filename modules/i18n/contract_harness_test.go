package i18n

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/leomorpho/goship/framework/core/contracttests"
)

func TestDefaultAdapterPassesI18nContract(t *testing.T) {
	contracttests.RunI18nContract(t, contracttests.I18nContractSubject{
		Name:               "modules/i18n default adapter",
		KnownDefaultKey:    "app.welcome",
		KnownDefaultResult: "Welcome",
		Build: func(t *testing.T) contracttests.I18nContractAdapter {
			t.Helper()
			root := t.TempDir()
			writeContractLocale(t, root, "en.yaml", `
app:
  title: "GoShip"
  welcome: "Welcome"
`)
			writeContractLocale(t, root, "fr.yaml", `
app:
  title: "GoShip FR"
  welcome: "Bienvenue"
`)
			service, err := NewService(Options{LocaleDir: root, DefaultLanguage: "en"})
			if err != nil {
				t.Fatalf("new service: %v", err)
			}
			return service
		},
	})
}

func writeContractLocale(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir locales: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write locale file: %v", err)
	}
}
