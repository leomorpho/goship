package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunI18nMissing(t *testing.T) {
	root := t.TempDir()
	writeI18nFixture(t, root, map[string]string{
		"locales/en.yaml": `
auth:
  login:
    title: "Sign in to your account"
    submit: "Sign in"
`,
		"locales/fr.yaml": `
auth:
  login:
    title: "Connectez-vous a votre compte"
    submit: ""
`,
		"go.mod": "module example.com/i18n-test\n\ngo 1.25\n",
	})

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := RunI18n([]string{"missing"}, I18nDeps{
		Out:          out,
		Err:          errOut,
		FindGoModule: findI18nGoModule,
	})
	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "fr: auth.login.submit") {
		t.Fatalf("stdout = %q, want missing fr auth.login.submit", out.String())
	}
}

func TestRunI18nUnused(t *testing.T) {
	root := t.TempDir()
	writeI18nFixture(t, root, map[string]string{
		"locales/en.yaml": `
auth:
  login:
    title: "Sign in to your account"
navigation:
  home: "Home"
`,
		"locales/fr.yaml": `
auth:
  login:
    title: "Connectez-vous a votre compte"
navigation:
  home: "Accueil"
`,
		"app/web/controllers/sample.go": `package controllers
func demo() {
	_ = container.I18n.T(ctx, "auth.login.title")
}
`,
		"go.mod": "module example.com/i18n-test\n\ngo 1.25\n",
	})

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := RunI18n([]string{"unused"}, I18nDeps{
		Out:          out,
		Err:          errOut,
		FindGoModule: findI18nGoModule,
	})
	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "navigation.home") {
		t.Fatalf("stdout = %q, want navigation.home", out.String())
	}
	if strings.Contains(out.String(), "auth.login.title") {
		t.Fatalf("stdout = %q, should not list used key auth.login.title", out.String())
	}
}

func TestRunI18nHelp(t *testing.T) {
	out := &bytes.Buffer{}
	if code := RunI18n([]string{"--help"}, I18nDeps{
		Out:          out,
		Err:          &bytes.Buffer{},
		FindGoModule: findI18nGoModule,
	}); code != 0 {
		t.Fatalf("help code = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "ship i18n commands:") {
		t.Fatalf("stdout = %q, want i18n help", out.String())
	}
}

func writeI18nFixture(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for relPath, content := range files {
		abs := filepath.Join(root, relPath)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", relPath, err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", relPath, err)
		}
	}
}

func findI18nGoModule(start string) (string, string, error) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, "", nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", "", os.ErrNotExist
		}
		dir = parent
	}
}
