package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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

func TestRunI18nInitCreatesBaselineLocales(t *testing.T) {
	root := t.TempDir()
	writeI18nFixture(t, root, map[string]string{
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
	code := RunI18n([]string{"init"}, I18nDeps{
		Out:          out,
		Err:          errOut,
		FindGoModule: findI18nGoModule,
	})
	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, errOut.String())
	}
	if strings.TrimSpace(errOut.String()) != "" {
		t.Fatalf("stderr = %q, want empty", errOut.String())
	}

	en, err := os.ReadFile(filepath.Join(root, "locales", "en.yaml"))
	if err != nil {
		t.Fatalf("read en locale: %v", err)
	}
	fr, err := os.ReadFile(filepath.Join(root, "locales", "fr.yaml"))
	if err != nil {
		t.Fatalf("read fr locale: %v", err)
	}
	if !strings.Contains(string(en), "app:") || !strings.Contains(string(fr), "app:") {
		t.Fatalf("expected baseline locale content, en=%q fr=%q", string(en), string(fr))
	}
	if !strings.Contains(out.String(), "ship i18n:scan --format json") {
		t.Fatalf("stdout = %q, want follow-up migration commands", out.String())
	}
}

func TestRunI18nInitIsIdempotentWithoutForce(t *testing.T) {
	root := t.TempDir()
	writeI18nFixture(t, root, map[string]string{
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

	firstOut := &bytes.Buffer{}
	firstErr := &bytes.Buffer{}
	if code := RunI18n([]string{"init"}, I18nDeps{
		Out:          firstOut,
		Err:          firstErr,
		FindGoModule: findI18nGoModule,
	}); code != 0 {
		t.Fatalf("first run code = %d, stderr = %s", code, firstErr.String())
	}
	before, err := os.ReadFile(filepath.Join(root, "locales", "en.yaml"))
	if err != nil {
		t.Fatalf("read initial en locale: %v", err)
	}

	secondOut := &bytes.Buffer{}
	secondErr := &bytes.Buffer{}
	if code := RunI18n([]string{"init"}, I18nDeps{
		Out:          secondOut,
		Err:          secondErr,
		FindGoModule: findI18nGoModule,
	}); code != 0 {
		t.Fatalf("second run code = %d, stderr = %s", code, secondErr.String())
	}
	after, err := os.ReadFile(filepath.Join(root, "locales", "en.yaml"))
	if err != nil {
		t.Fatalf("read rerun en locale: %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf("en locale changed on rerun without --force")
	}
}

func TestRunI18nInitOverwriteGuardAndForce(t *testing.T) {
	root := t.TempDir()
	writeI18nFixture(t, root, map[string]string{
		"go.mod": "module example.com/i18n-test\n\ngo 1.25\n",
		"locales/en.yaml": "custom: keep-me\n",
		"locales/fr.yaml": "custom: garde-moi\n",
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
	if code := RunI18n([]string{"init"}, I18nDeps{
		Out:          out,
		Err:          errOut,
		FindGoModule: findI18nGoModule,
	}); code != 0 {
		t.Fatalf("init code = %d, stderr = %s", code, errOut.String())
	}
	enNoForce, err := os.ReadFile(filepath.Join(root, "locales", "en.yaml"))
	if err != nil {
		t.Fatalf("read en without force: %v", err)
	}
	if strings.TrimSpace(string(enNoForce)) != "custom: keep-me" {
		t.Fatalf("expected existing en locale to be preserved without --force, got %q", string(enNoForce))
	}

	forceOut := &bytes.Buffer{}
	forceErr := &bytes.Buffer{}
	if code := RunI18n([]string{"init", "--force"}, I18nDeps{
		Out:          forceOut,
		Err:          forceErr,
		FindGoModule: findI18nGoModule,
	}); code != 0 {
		t.Fatalf("force init code = %d, stderr = %s", code, forceErr.String())
	}
	enForce, err := os.ReadFile(filepath.Join(root, "locales", "en.yaml"))
	if err != nil {
		t.Fatalf("read en with force: %v", err)
	}
	if strings.TrimSpace(string(enForce)) == "custom: keep-me" {
		t.Fatalf("expected --force to overwrite en locale")
	}
}

func TestRunI18nInitUsageAndHelp(t *testing.T) {
	helpOut := &bytes.Buffer{}
	helpErr := &bytes.Buffer{}
	if code := RunI18n([]string{"init", "--help"}, I18nDeps{
		Out:          helpOut,
		Err:          helpErr,
		FindGoModule: findI18nGoModule,
	}); code != 0 {
		t.Fatalf("help code = %d, stderr = %q", code, helpErr.String())
	}
	if !strings.Contains(helpOut.String(), "usage: ship i18n:init [--force]") {
		t.Fatalf("help stdout = %q", helpOut.String())
	}

	root := t.TempDir()
	writeI18nFixture(t, root, map[string]string{
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
	if code := RunI18n([]string{"init", "--unknown"}, I18nDeps{
		Out:          out,
		Err:          errOut,
		FindGoModule: findI18nGoModule,
	}); code != 1 {
		t.Fatalf("invalid args code = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "usage: ship i18n:init [--force]") {
		t.Fatalf("stderr = %q, want init usage", errOut.String())
	}
}

func TestRunI18nScanJSONDeterministicWithPathsAndLimit(t *testing.T) {
	root := t.TempDir()
	writeI18nFixture(t, root, map[string]string{
		"go.mod": "module example.com/i18n-test\n\ngo 1.25\n",
		"app/web/controllers/sample.go": `package controllers
func demo() string {
	_ = container.I18n.T(ctx, "auth.login.title")
	return "Welcome from Go"
}
`,
		"app/views/web/pages/demo.templ": `package pages
templ Demo() {
	<h1>Hello from templ</h1>
}
`,
		"frontend/islands/demo.ts": `export function demo() {
	const label = "Click here now";
	return label;
}
`,
	})

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	run := func() scanResult {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunI18n([]string{
			"scan",
			"--format", "json",
			"--paths", "app/web/controllers,frontend/islands",
			"--limit", "2",
		}, I18nDeps{
			Out:          out,
			Err:          errOut,
			FindGoModule: findI18nGoModule,
		})
		if code != 0 {
			t.Fatalf("code = %d, stderr = %s", code, errOut.String())
		}
		if strings.TrimSpace(errOut.String()) != "" {
			t.Fatalf("stderr = %q, want empty", errOut.String())
		}
		var parsed scanResult
		if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
			t.Fatalf("parse scan JSON: %v\nraw=%s", err, out.String())
		}
		return parsed
	}

	first := run()
	second := run()
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("scan output was not deterministic: first=%+v second=%+v", first, second)
	}
	if len(first.Issues) != 2 {
		t.Fatalf("issues len = %d, want 2", len(first.Issues))
	}
	for _, issue := range first.Issues {
		if !strings.HasPrefix(issue.File, "app/web/controllers/") && !strings.HasPrefix(issue.File, "frontend/islands/") {
			t.Fatalf("issue file %q should respect --paths filter", issue.File)
		}
		if issue.ID == "" || issue.Kind == "" || issue.Severity == "" || issue.Message == "" || issue.Confidence == "" {
			t.Fatalf("issue missing required fields: %+v", issue)
		}
		if issue.Line <= 0 || issue.Column <= 0 {
			t.Fatalf("issue position must be positive: %+v", issue)
		}
	}
}

func TestRunI18nScanJSONInvalidArgs(t *testing.T) {
	root := t.TempDir()
	writeI18nFixture(t, root, map[string]string{
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
	if code := RunI18n([]string{"scan", "--format", "table"}, I18nDeps{
		Out:          out,
		Err:          errOut,
		FindGoModule: findI18nGoModule,
	}); code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "usage: ship i18n:scan") {
		t.Fatalf("stderr = %q, want usage", errOut.String())
	}
}

func TestRunI18nScanGoASTIgnoresLogsSQLAndTests(t *testing.T) {
	root := t.TempDir()
	writeI18nFixture(t, root, map[string]string{
		"go.mod": "module example.com/i18n-test\n\ngo 1.25\n",
		"app/web/controllers/sample.go": `package controllers
import (
	"log/slog"
)
func demo() string {
	slog.Info("worker started")
	query := "SELECT id, name FROM users"
	_ = query
	return "Welcome user"
}
`,
		"app/web/controllers/sample_test.go": `package controllers
func TestDemo(t *testing.T) {
	_ = "fixture string should be ignored"
}
`,
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
	code := RunI18n([]string{
		"scan",
		"--format", "json",
		"--paths", "app/web/controllers",
	}, I18nDeps{
		Out:          out,
		Err:          errOut,
		FindGoModule: findI18nGoModule,
	})
	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, errOut.String())
	}
	if strings.TrimSpace(errOut.String()) != "" {
		t.Fatalf("stderr = %q, want empty", errOut.String())
	}

	var parsed scanResult
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Fatalf("parse scan JSON: %v\nraw=%s", err, out.String())
	}
	if len(parsed.Issues) != 1 {
		t.Fatalf("issues len = %d, want 1", len(parsed.Issues))
	}
	if !strings.Contains(parsed.Issues[0].Message, "Welcome user") {
		t.Fatalf("issue message = %q, want welcome literal finding", parsed.Issues[0].Message)
	}
}

type scanResult struct {
	Issues []scanIssue `json:"issues"`
}

type scanIssue struct {
	ID           string `json:"id"`
	Kind         string `json:"kind"`
	Severity     string `json:"severity"`
	File         string `json:"file"`
	Line         int    `json:"line"`
	Column       int    `json:"column"`
	Message      string `json:"message"`
	SuggestedKey string `json:"suggested_key"`
	Confidence   string `json:"confidence"`
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
