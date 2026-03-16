package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	policies "github.com/leomorpho/goship/tools/cli/ship/internal/policies"
)

func TestParseNewArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{name: "minimal", args: []string{"demo"}},
		{name: "module equals", args: []string{"demo", "--module=example.com/demo"}},
		{name: "module spaced", args: []string{"demo", "--module", "example.com/demo"}},
		{name: "dry-run", args: []string{"demo", "--dry-run"}},
		{name: "force", args: []string{"demo", "--force"}},
		{name: "i18n enabled", args: []string{"demo", "--i18n"}},
		{name: "i18n disabled", args: []string{"demo", "--no-i18n"}},
		{name: "i18n locale pack equals", args: []string{"demo", "--i18n-locale-pack=top15"}},
		{name: "i18n locale pack spaced", args: []string{"demo", "--i18n-locale-pack", "starter"}},
		{name: "invalid i18n locale pack", args: []string{"demo", "--i18n-locale-pack", "badpack"}, wantErr: true},
		{name: "conflicting i18n flags", args: []string{"demo", "--i18n", "--no-i18n"}, wantErr: true},
		{name: "bad name", args: []string{"-bad"}, wantErr: true},
		{name: "unknown option", args: []string{"demo", "--wat"}, wantErr: true},
		{name: "too many args", args: []string{"demo", "extra"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseNewArgs(tt.args)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestParseNewArgsI18nFlags(t *testing.T) {
	opts, err := ParseNewArgs([]string{"demo", "--i18n"})
	if err != nil {
		t.Fatalf("ParseNewArgs returned error: %v", err)
	}
	if !opts.I18nSet {
		t.Fatalf("I18nSet = false, want true")
	}
	if !opts.I18nEnabled {
		t.Fatalf("I18nEnabled = false, want true")
	}

	opts, err = ParseNewArgs([]string{"demo", "--no-i18n"})
	if err != nil {
		t.Fatalf("ParseNewArgs returned error: %v", err)
	}
	if !opts.I18nSet {
		t.Fatalf("I18nSet = false, want true")
	}
	if opts.I18nEnabled {
		t.Fatalf("I18nEnabled = true, want false")
	}

	opts, err = ParseNewArgs([]string{"demo", "--i18n-locale-pack", "top15"})
	if err != nil {
		t.Fatalf("ParseNewArgs returned error: %v", err)
	}
	if !opts.I18nLocalePackSet {
		t.Fatalf("I18nLocalePackSet = false, want true")
	}
	if opts.I18nLocalePack != "top15" {
		t.Fatalf("I18nLocalePack = %q, want top15", opts.I18nLocalePack)
	}
}

func TestResolveNewI18nOptionsLocalePackBehavior(t *testing.T) {
	t.Run("locale pack implies i18n enabled", func(t *testing.T) {
		opts, err := ParseNewArgs([]string{"demo", "--i18n-locale-pack", "top15"})
		if err != nil {
			t.Fatalf("ParseNewArgs returned error: %v", err)
		}
		resolved, err := resolveNewI18nOptions(opts, NewDeps{})
		if err != nil {
			t.Fatalf("resolveNewI18nOptions returned error: %v", err)
		}
		if !resolved.I18nEnabled || !resolved.I18nSet {
			t.Fatalf("resolved options = %+v, want i18n enabled/set", resolved)
		}
		if resolved.I18nLocalePack != "top15" {
			t.Fatalf("I18nLocalePack = %q, want top15", resolved.I18nLocalePack)
		}
	})

	t.Run("no-i18n with locale pack is rejected", func(t *testing.T) {
		opts, err := ParseNewArgs([]string{"demo", "--no-i18n", "--i18n-locale-pack", "starter"})
		if err != nil {
			t.Fatalf("ParseNewArgs returned error: %v", err)
		}
		if _, err := resolveNewI18nOptions(opts, NewDeps{}); err == nil {
			t.Fatal("expected resolveNewI18nOptions to fail for --no-i18n + --i18n-locale-pack")
		}
	})
}

func TestScaffoldNewProject(t *testing.T) {
	root := t.TempDir()
	opts := NewProjectOptions{
		Name:    "demo",
		Module:  "example.com/demo",
		AppPath: filepath.Join(root, "demo"),
	}

	if err := ScaffoldNewProject(opts, NewDeps{
		ParseAgentPolicyBytes:      func(b []byte) (policies.AgentPolicy, error) { return policies.ParsePolicyBytes(b) },
		RenderAgentPolicyArtifacts: policies.RenderPolicyArtifacts,
		AgentPolicyFilePath:        policies.AgentPolicyFilePath,
	}); err != nil {
		t.Fatalf("ScaffoldNewProject failed: %v", err)
	}

	checkFiles := []string{
		filepath.Join(opts.AppPath, "go.mod"),
		filepath.Join(opts.AppPath, "Procfile"),
		filepath.Join(opts.AppPath, "Procfile.dev"),
		filepath.Join(opts.AppPath, "Procfile.worker"),
		filepath.Join(opts.AppPath, "config", "modules.yaml"),
		filepath.Join(opts.AppPath, "app", "router.go"),
		filepath.Join(opts.AppPath, "db", "queries", "user.sql"),
		filepath.Join(opts.AppPath, "db", "migrate", "migrations", ".gitkeep"),
		filepath.Join(opts.AppPath, "app", "web", "routenames", "routenames.go"),
		filepath.Join(opts.AppPath, "app", "views", "templates.go"),
		filepath.Join(opts.AppPath, "app", "web", "controllers", "controllers.go"),
		filepath.Join(opts.AppPath, "app", "web", "middleware", "middleware.go"),
		filepath.Join(opts.AppPath, "app", "web", "ui", "ui.go"),
		filepath.Join(opts.AppPath, "app", "web", "viewmodels", "viewmodels.go"),
		filepath.Join(opts.AppPath, "app", "jobs", "jobs.go"),
		filepath.Join(opts.AppPath, "app", "foundation", "container.go"),
		filepath.Join(opts.AppPath, "app", "profiles", "repo.go"),
		filepath.Join(opts.AppPath, "app", "notifications", "notifier.go"),
		filepath.Join(opts.AppPath, "app", "subscriptions", "repo.go"),
		filepath.Join(opts.AppPath, "app", "emailsubscriptions", "repo.go"),
		filepath.Join(opts.AppPath, "docs", "00-index.md"),
		filepath.Join(opts.AppPath, "docs", "architecture", "01-architecture.md"),
		filepath.Join(opts.AppPath, "docs", "architecture", "08-cognitive-model.md"),
		filepath.Join(opts.AppPath, "cmd", "web", "main.go"),
	}
	for _, p := range checkFiles {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected file %s: %v", p, err)
		}
	}

	routerBytes, err := os.ReadFile(filepath.Join(opts.AppPath, "app", "router.go"))
	if err != nil {
		t.Fatal(err)
	}
	router := string(routerBytes)
	if !strings.Contains(router, "ship:routes:public:start") || !strings.Contains(router, "ship:routes:auth:start") {
		t.Fatalf("router markers missing:\n%s", router)
	}
	if !strings.Contains(router, "RouteNameHomeFeed") {
		t.Fatalf("expected starter router content copied into scaffold:\n%s", router)
	}

	containerBytes, err := os.ReadFile(filepath.Join(opts.AppPath, "app", "foundation", "container.go"))
	if err != nil {
		t.Fatal(err)
	}
	container := string(containerBytes)
	if !strings.Contains(container, "ship:container:start") || !strings.Contains(container, "ship:container:end") {
		t.Fatalf("container markers missing:\n%s", container)
	}
	if !strings.Contains(container, "EnabledModules") {
		t.Fatalf("expected starter container content copied into scaffold:\n%s", container)
	}
}

func TestScaffoldNewProjectDryRun(t *testing.T) {
	root := t.TempDir()
	opts := NewProjectOptions{
		Name:    "demo",
		Module:  "example.com/demo",
		AppPath: filepath.Join(root, "demo"),
		DryRun:  true,
	}
	if err := ScaffoldNewProject(opts, NewDeps{
		ParseAgentPolicyBytes:      func(b []byte) (policies.AgentPolicy, error) { return policies.ParsePolicyBytes(b) },
		RenderAgentPolicyArtifacts: policies.RenderPolicyArtifacts,
		AgentPolicyFilePath:        policies.AgentPolicyFilePath,
	}); err != nil {
		t.Fatalf("dry-run scaffold failed: %v", err)
	}
	if _, err := os.Stat(opts.AppPath); !os.IsNotExist(err) {
		t.Fatalf("expected no files in dry-run mode")
	}
}

func TestScaffoldNewProjectI18nEnabled(t *testing.T) {
	root := t.TempDir()
	opts := NewProjectOptions{
		Name:        "demo",
		Module:      "example.com/demo",
		AppPath:     filepath.Join(root, "demo"),
		I18nEnabled: true,
		I18nSet:     true,
	}
	if err := ScaffoldNewProject(opts, NewDeps{
		ParseAgentPolicyBytes:      func(b []byte) (policies.AgentPolicy, error) { return policies.ParsePolicyBytes(b) },
		RenderAgentPolicyArtifacts: policies.RenderPolicyArtifacts,
		AgentPolicyFilePath:        policies.AgentPolicyFilePath,
	}); err != nil {
		t.Fatalf("ScaffoldNewProject failed: %v", err)
	}

	for _, p := range []string{
		filepath.Join(opts.AppPath, "locales", "en.toml"),
		filepath.Join(opts.AppPath, "locales", "fr.toml"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected file %s: %v", p, err)
		}
	}

	containerBytes, err := os.ReadFile(filepath.Join(opts.AppPath, "app", "foundation", "container.go"))
	if err != nil {
		t.Fatal(err)
	}
	container := string(containerBytes)
	if !strings.Contains(container, "[]string{\"auth\", \"profile\", \"i18n\"}") {
		t.Fatalf("expected i18n module enabled in starter container:\n%s", container)
	}
}

func TestScaffoldNewProjectI18nTop15Pack(t *testing.T) {
	root := t.TempDir()
	opts := NewProjectOptions{
		Name:              "demo",
		Module:            "example.com/demo",
		AppPath:           filepath.Join(root, "demo"),
		I18nEnabled:       true,
		I18nSet:           true,
		I18nLocalePack:    "top15",
		I18nLocalePackSet: true,
	}
	if err := ScaffoldNewProject(opts, NewDeps{
		ParseAgentPolicyBytes:      func(b []byte) (policies.AgentPolicy, error) { return policies.ParsePolicyBytes(b) },
		RenderAgentPolicyArtifacts: policies.RenderPolicyArtifacts,
		AgentPolicyFilePath:        policies.AgentPolicyFilePath,
	}); err != nil {
		t.Fatalf("ScaffoldNewProject failed: %v", err)
	}

	wantCodes := []string{"ar", "de", "en", "es", "fr", "hi", "id", "it", "ja", "ko", "nl", "pt", "ru", "tr", "zh"}
	for _, code := range wantCodes {
		path := filepath.Join(opts.AppPath, "locales", code+".toml")
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected locale file %s: %v", path, err)
		}
	}
}

func TestRunNewPromptsForI18nWhenInteractive(t *testing.T) {
	root := t.TempDir()
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
	if code := RunNew([]string{"demo"}, NewDeps{
		Out:                        out,
		Err:                        errOut,
		ParseAgentPolicyBytes:      policies.ParsePolicyBytes,
		RenderAgentPolicyArtifacts: policies.RenderPolicyArtifacts,
		AgentPolicyFilePath:        policies.AgentPolicyFilePath,
		IsInteractive:              func() bool { return true },
		PromptI18nEnable:           func() (bool, error) { return true, nil },
	}); code != 0 {
		t.Fatalf("RunNew code = %d, stderr = %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "I18n enabled") {
		t.Fatalf("stdout = %q, want i18n enabled hint", out.String())
	}
	if !strings.Contains(out.String(), "starter pack") {
		t.Fatalf("stdout = %q, want starter locale pack details", out.String())
	}
	for _, p := range []string{
		filepath.Join(root, "demo", "locales", "en.toml"),
		filepath.Join(root, "demo", "locales", "fr.toml"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected file %s: %v", p, err)
		}
	}
}
