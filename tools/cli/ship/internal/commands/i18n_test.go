package commands

import (
	"bytes"
	"encoding/json"
	"go/parser"
	"go/token"
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

func TestRunI18nMissingIncludesPluralSelectCompletenessDiagnostics(t *testing.T) {
	root := t.TempDir()
	writeI18nFixture(t, root, map[string]string{
		"locales/en.toml": `
"cart.items.one" = "{{.Count}} item"
"profile.role.admin" = "Administrator"
`,
		"app/web/controllers/sample.go": `package controllers
func demo() string {
	_ = container.I18n.TC(ctx, "cart.items", 2)
	_ = container.I18n.TS(ctx, "profile.role", "admin")
	return ""
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
	code := RunI18n([]string{"missing"}, I18nDeps{
		Out:          out,
		Err:          errOut,
		FindGoModule: findI18nGoModule,
	})
	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "en: cart.items (plural_missing_other)") {
		t.Fatalf("stdout = %q, want plural completeness issue", out.String())
	}
	if !strings.Contains(out.String(), "en: profile.role (select_missing_other)") {
		t.Fatalf("stdout = %q, want select completeness issue", out.String())
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

	en, err := os.ReadFile(filepath.Join(root, "locales", "en.toml"))
	if err != nil {
		t.Fatalf("read en locale: %v", err)
	}
	fr, err := os.ReadFile(filepath.Join(root, "locales", "fr.toml"))
	if err != nil {
		t.Fatalf("read fr locale: %v", err)
	}
	if !strings.Contains(string(en), `"app.title" =`) || !strings.Contains(string(fr), `"app.title" =`) {
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
	before, err := os.ReadFile(filepath.Join(root, "locales", "en.toml"))
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
	after, err := os.ReadFile(filepath.Join(root, "locales", "en.toml"))
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
		"locales/en.toml": `"custom" = "keep-me"` + "\n",
		"locales/fr.toml": `"custom" = "garde-moi"` + "\n",
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
	enNoForce, err := os.ReadFile(filepath.Join(root, "locales", "en.toml"))
	if err != nil {
		t.Fatalf("read en without force: %v", err)
	}
	if strings.TrimSpace(string(enNoForce)) != `"custom" = "keep-me"` {
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
	enForce, err := os.ReadFile(filepath.Join(root, "locales", "en.toml"))
	if err != nil {
		t.Fatalf("read en with force: %v", err)
	}
	if strings.TrimSpace(string(enForce)) == `"custom" = "keep-me"` {
		t.Fatalf("expected --force to overwrite en locale")
	}
}

func TestRunI18nCompileGeneratesTypedKeysForGoAndTS(t *testing.T) {
	root := t.TempDir()
	writeI18nFixture(t, root, map[string]string{
		"go.mod": "module example.com/i18n-test\n\ngo 1.25\n",
		"locales/en.toml": `
"app.title" = "Demo"
"auth.login.title" = "Sign in"
"profile.edit-phone.title" = "Edit phone"
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

	runCompile := func() {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		if code := RunI18n([]string{"compile"}, I18nDeps{
			Out:          out,
			Err:          errOut,
			FindGoModule: findI18nGoModule,
		}); code != 0 {
			t.Fatalf("compile code = %d, stderr = %s", code, errOut.String())
		}
		if strings.TrimSpace(errOut.String()) != "" {
			t.Fatalf("stderr = %q, want empty", errOut.String())
		}
	}

	runCompile()

	goPath := filepath.Join(root, "app", "i18nkeys", "keys_gen.go")
	goContent, err := os.ReadFile(goPath)
	if err != nil {
		t.Fatalf("read generated go keys: %v", err)
	}
	if !strings.Contains(string(goContent), `KeyAppTitle = "app.title"`) {
		t.Fatalf("generated go keys missing app title const:\n%s", string(goContent))
	}
	if !strings.Contains(string(goContent), `KeyAuthLoginTitle = "auth.login.title"`) {
		t.Fatalf("generated go keys missing auth login const:\n%s", string(goContent))
	}
	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, goPath, goContent, parser.ParseComments); err != nil {
		t.Fatalf("generated go file must parse: %v", err)
	}

	tsPath := filepath.Join(root, "frontend", "islands", "i18n-keys.ts")
	tsContent, err := os.ReadFile(tsPath)
	if err != nil {
		t.Fatalf("read generated ts keys: %v", err)
	}
	if !strings.Contains(string(tsContent), `"auth.login.title": "auth.login.title"`) {
		t.Fatalf("generated ts keys missing auth.login.title:\n%s", string(tsContent))
	}
	if !strings.Contains(string(tsContent), "export type I18nKey = keyof typeof i18nKeys;") {
		t.Fatalf("generated ts keys missing I18nKey type alias:\n%s", string(tsContent))
	}

	beforeGo := string(goContent)
	beforeTS := string(tsContent)
	runCompile()
	afterGo, err := os.ReadFile(goPath)
	if err != nil {
		t.Fatalf("read generated go keys after rerun: %v", err)
	}
	afterTS, err := os.ReadFile(tsPath)
	if err != nil {
		t.Fatalf("read generated ts keys after rerun: %v", err)
	}
	if beforeGo != string(afterGo) {
		t.Fatalf("generated go keys should be deterministic across runs")
	}
	if beforeTS != string(afterTS) {
		t.Fatalf("generated ts keys should be deterministic across runs")
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

func TestRunI18nMigrateConvertsYAMLToTOML(t *testing.T) {
	root := t.TempDir()
	writeI18nFixture(t, root, map[string]string{
		"go.mod": "module example.com/i18n-test\n\ngo 1.25\n",
		"locales/en.yaml": `
auth:
  login:
    title: "Sign in to your account"
`,
		"locales/fr.yaml": `
auth:
  login:
    title: "Connectez-vous a votre compte"
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
	if code := RunI18n([]string{"migrate"}, I18nDeps{
		Out:          out,
		Err:          errOut,
		FindGoModule: findI18nGoModule,
	}); code != 0 {
		t.Fatalf("migrate code = %d, stderr = %s", code, errOut.String())
	}
	if strings.TrimSpace(errOut.String()) != "" {
		t.Fatalf("stderr = %q, want empty", errOut.String())
	}

	enToml, err := os.ReadFile(filepath.Join(root, "locales", "en.toml"))
	if err != nil {
		t.Fatalf("read en.toml: %v", err)
	}
	frToml, err := os.ReadFile(filepath.Join(root, "locales", "fr.toml"))
	if err != nil {
		t.Fatalf("read fr.toml: %v", err)
	}
	if !strings.Contains(string(enToml), `"auth.login.title" = "Sign in to your account"`) {
		t.Fatalf("en.toml missing migrated key:\n%s", string(enToml))
	}
	if !strings.Contains(string(frToml), `"auth.login.title" = "Connectez-vous a votre compte"`) {
		t.Fatalf("fr.toml missing migrated key:\n%s", string(frToml))
	}
}

func TestRunI18nNormalizeIsIdempotentForTOML(t *testing.T) {
	root := t.TempDir()
	writeI18nFixture(t, root, map[string]string{
		"go.mod": "module example.com/i18n-test\n\ngo 1.25\n",
		"locales/en.toml": `
[auth]
  [auth.login]
  submit = "Sign in"
  title = "Sign in to your account"
[common]
save = "Save"
cancel = "Cancel"
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
	if code := RunI18n([]string{"normalize"}, I18nDeps{
		Out:          out,
		Err:          errOut,
		FindGoModule: findI18nGoModule,
	}); code != 0 {
		t.Fatalf("normalize code = %d, stderr = %s", code, errOut.String())
	}
	if strings.TrimSpace(errOut.String()) != "" {
		t.Fatalf("stderr = %q, want empty", errOut.String())
	}

	first, err := os.ReadFile(filepath.Join(root, "locales", "en.toml"))
	if err != nil {
		t.Fatalf("read first normalize output: %v", err)
	}
	if !strings.Contains(string(first), `"auth.login.submit" = "Sign in"`) {
		t.Fatalf("normalized output missing expected key:\n%s", string(first))
	}

	secondOut := &bytes.Buffer{}
	secondErr := &bytes.Buffer{}
	if code := RunI18n([]string{"normalize"}, I18nDeps{
		Out:          secondOut,
		Err:          secondErr,
		FindGoModule: findI18nGoModule,
	}); code != 0 {
		t.Fatalf("second normalize code = %d, stderr = %s", code, secondErr.String())
	}
	second, err := os.ReadFile(filepath.Join(root, "locales", "en.toml"))
	if err != nil {
		t.Fatalf("read second normalize output: %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("normalize should be idempotent:\nfirst:\n%s\nsecond:\n%s", string(first), string(second))
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

func TestRunI18nScanIncludesTemplAndIslandsJS(t *testing.T) {
	root := t.TempDir()
	writeI18nFixture(t, root, map[string]string{
		"go.mod": "module example.com/i18n-test\n\ngo 1.25\n",
		"app/views/web/pages/demo.templ": `package pages
templ Demo() {
	<div>
		<h1>Hello from templ</h1>
	</div>
}
`,
		"frontend/islands/chat.ts": `export function chat() {
	const cta = "Send message";
	return cta;
}
`,
		"frontend/helpers/strings.ts": `export const msg = "Do not scan non-islands ts files";`,
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
	code := RunI18n([]string{"scan", "--format", "json"}, I18nDeps{
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

	foundTempl := false
	foundIslandTS := false
	for _, issue := range parsed.Issues {
		if issue.Line <= 0 || issue.Column <= 0 {
			t.Fatalf("issue must include position: %+v", issue)
		}
		if strings.HasPrefix(issue.File, "app/views/web/pages/") {
			foundTempl = true
		}
		if strings.HasPrefix(issue.File, "frontend/islands/") {
			foundIslandTS = true
		}
		if strings.HasPrefix(issue.File, "frontend/helpers/") {
			t.Fatalf("scanner should ignore non-islands JS/TS paths, got issue %+v", issue)
		}
	}
	if !foundTempl {
		t.Fatal("expected at least one templ issue")
	}
	if !foundIslandTS {
		t.Fatal("expected at least one islands JS/TS issue")
	}
}

func TestRunI18nScanIncludesIslandsJSX(t *testing.T) {
	root := t.TempDir()
	writeI18nFixture(t, root, map[string]string{
		"go.mod": "module example.com/i18n-test\n\ngo 1.25\n",
		"frontend/islands/react_counter.jsx": `export function ReactCounter() {
	const label = "Click counter now";
	return <button>{label}</button>;
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
	code := RunI18n([]string{"scan", "--format", "json"}, I18nDeps{
		Out:          out,
		Err:          errOut,
		FindGoModule: findI18nGoModule,
	})
	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, errOut.String())
	}
	var parsed scanResult
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Fatalf("parse scan JSON: %v\nraw=%s", err, out.String())
	}
	found := false
	for _, issue := range parsed.Issues {
		if strings.HasPrefix(issue.File, "frontend/islands/react_counter.jsx") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected jsx issue in frontend/islands/react_counter.jsx, got %+v", parsed.Issues)
	}
}

func TestRunI18nScanIgnoresNestedI18nCalls(t *testing.T) {
	root := t.TempDir()
	writeI18nFixture(t, root, map[string]string{
		"go.mod": "module example.com/i18n-test\n\ngo 1.25\n",
		"app/web/controllers/sample.go": `package controllers
func demo() string {
	_ = container.I18n.T(ctx, "auth.login.title")
	return "Welcome user"
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
	code := RunI18n([]string{"scan", "--format", "json", "--paths", "app/web/controllers"}, I18nDeps{
		Out:          out,
		Err:          errOut,
		FindGoModule: findI18nGoModule,
	})
	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, errOut.String())
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

func TestRunI18nInstrumentDryRunDeterministicAndNoWrite(t *testing.T) {
	root := t.TempDir()
	writeI18nFixture(t, root, map[string]string{
		"go.mod": "module example.com/i18n-test\n\ngo 1.25\n",
		"locales/en.yaml": `
app:
  title: "Demo"
`,
		"app/web/controllers/home.go": `package controllers

import "net/http"

func Home(c *Controller) error {
	return c.String(http.StatusOK, "Welcome traveler")
}
`,
		"app/views/web/pages/demo.templ": `package pages
templ Demo() {
	<h1>Hello from templ</h1>
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

	run := func() instrumentResult {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunI18n([]string{
			"instrument",
			"--paths", "app/web/controllers,app/views",
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

		var parsed instrumentResult
		if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
			t.Fatalf("parse instrument JSON: %v\nraw=%s", err, out.String())
		}
		return parsed
	}

	first := run()
	second := run()
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("instrument output was not deterministic: first=%+v second=%+v", first, second)
	}
	if len(first.Rewrites) != 1 {
		t.Fatalf("rewrites len = %d, want 1", len(first.Rewrites))
	}
	if len(first.Skipped) == 0 {
		t.Fatal("expected at least one skipped finding in dry-run output")
	}

	content, err := os.ReadFile(filepath.Join(root, "app/web/controllers/home.go"))
	if err != nil {
		t.Fatalf("read source after dry-run: %v", err)
	}
	if !strings.Contains(string(content), `return c.String(http.StatusOK, "Welcome traveler")`) {
		t.Fatalf("dry-run should not rewrite file, got:\n%s", string(content))
	}
}

func TestRunI18nInstrumentApplyRewritesAndUpdatesLocale(t *testing.T) {
	root := t.TempDir()
	writeI18nFixture(t, root, map[string]string{
		"go.mod": "module example.com/i18n-test\n\ngo 1.25\n",
		"locales/en.yaml": `
app:
  title: "Demo"
`,
		"app/web/controllers/home.go": `package controllers

import "net/http"

func Home(c *Controller) error {
	return c.String(http.StatusOK, "Welcome traveler")
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
		"instrument",
		"--apply",
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

	rewrittenPath := filepath.Join(root, "app/web/controllers/home.go")
	rewritten, err := os.ReadFile(rewrittenPath)
	if err != nil {
		t.Fatalf("read rewritten source: %v", err)
	}
	if !strings.Contains(string(rewritten), `c.Container.I18n.T(c.Request().Context(), "app.welcome_traveler")`) {
		t.Fatalf("expected rewritten i18n call, got:\n%s", string(rewritten))
	}

	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, rewrittenPath, rewritten, parser.ParseComments); err != nil {
		t.Fatalf("rewritten file must remain parseable Go syntax: %v", err)
	}

	locale, err := os.ReadFile(filepath.Join(root, "locales/en.yaml"))
	if err != nil {
		t.Fatalf("read baseline locale: %v", err)
	}
	if !strings.Contains(string(locale), `app.welcome_traveler: "Welcome traveler"`) {
		t.Fatalf("expected generated locale key in en.yaml, got:\n%s", string(locale))
	}
}

func TestRunI18nInstrumentApplyRewritesCtxStringPattern(t *testing.T) {
	root := t.TempDir()
	writeI18nFixture(t, root, map[string]string{
		"go.mod": "module example.com/i18n-test\n\ngo 1.25\n",
		"locales/en.yaml": `
app:
  title: "Demo"
`,
		"app/web/controllers/home.go": `package controllers

import "net/http"

func Home(ctx *Controller) error {
	return ctx.String(http.StatusOK, "Welcome from ctx")
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
		"instrument",
		"--apply",
		"--paths", "app/web/controllers",
	}, I18nDeps{
		Out:          out,
		Err:          errOut,
		FindGoModule: findI18nGoModule,
	})
	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, errOut.String())
	}

	rewritten, err := os.ReadFile(filepath.Join(root, "app/web/controllers/home.go"))
	if err != nil {
		t.Fatalf("read rewritten source: %v", err)
	}
	if !strings.Contains(string(rewritten), `ctx.Container.I18n.T(ctx.Request().Context(), "app.welcome_from_ctx")`) {
		t.Fatalf("expected ctx receiver rewrite, got:\n%s", string(rewritten))
	}
}

func TestRunI18nInstrumentUsageAndInvalidArgs(t *testing.T) {
	helpOut := &bytes.Buffer{}
	helpErr := &bytes.Buffer{}
	if code := RunI18n([]string{"instrument", "--help"}, I18nDeps{
		Out:          helpOut,
		Err:          helpErr,
		FindGoModule: findI18nGoModule,
	}); code != 0 {
		t.Fatalf("help code = %d, stderr = %q", code, helpErr.String())
	}
	if !strings.Contains(helpOut.String(), "usage: ship i18n:instrument [--apply] [--paths <path1,path2,...>] [--limit <n>]") {
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
	if code := RunI18n([]string{"instrument", "--unknown"}, I18nDeps{
		Out:          out,
		Err:          errOut,
		FindGoModule: findI18nGoModule,
	}); code != 1 {
		t.Fatalf("invalid args code = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "usage: ship i18n:instrument") {
		t.Fatalf("stderr = %q, want instrument usage", errOut.String())
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

type instrumentResult struct {
	Rewrites []instrumentRewrite `json:"rewrites"`
	Skipped  []instrumentSkip    `json:"skipped"`
}

type instrumentRewrite struct {
	File         string `json:"file"`
	Line         int    `json:"line"`
	Column       int    `json:"column"`
	Before       string `json:"before"`
	After        string `json:"after"`
	SuggestedKey string `json:"suggested_key"`
}

type instrumentSkip struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Reason string `json:"reason"`
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
