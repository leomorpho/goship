package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

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
		{name: "unsupported locale pack equals", args: []string{"demo", "--i18n-locale-pack=top15"}, wantErr: true},
		{name: "unsupported locale pack spaced", args: []string{"demo", "--i18n-locale-pack", "starter"}, wantErr: true},
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

}

func TestParseNewUIFlag(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantProvider  string
		wantErrSubstr string
	}{
		{
			name:         "defaults to franken",
			args:         []string{"demo"},
			wantProvider: "franken",
		},
		{
			name:         "explicit franken",
			args:         []string{"demo", "--ui", "franken"},
			wantProvider: "franken",
		},
		{
			name:         "explicit daisy",
			args:         []string{"demo", "--ui=daisy"},
			wantProvider: "daisy",
		},
		{
			name:         "explicit bare",
			args:         []string{"demo", "--ui", "bare"},
			wantProvider: "bare",
		},
		{
			name:          "invalid provider",
			args:          []string{"demo", "--ui", "unknown"},
			wantErrSubstr: "unsupported --ui provider",
		},
		{
			name:          "missing provider value",
			args:          []string{"demo", "--ui"},
			wantErrSubstr: "missing value for --ui",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := ParseNewArgs(tt.args)
			if tt.wantErrSubstr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErrSubstr)
				}
				if !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("error = %q, want substring %q", err.Error(), tt.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if opts.UIProvider != tt.wantProvider {
				t.Fatalf("UIProvider = %q, want %q", opts.UIProvider, tt.wantProvider)
			}
		})
	}
}

func TestScaffoldNewProject(t *testing.T) {
	root := t.TempDir()
	opts := NewProjectOptions{
		Name:       "demo",
		Module:     "example.com/demo",
		AppPath:    filepath.Join(root, "demo"),
		UIProvider: newUIProviderDaisy,
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
		filepath.Join(opts.AppPath, "app", "views", "web", "layouts", "base.templ"),
		filepath.Join(opts.AppPath, "app", "jobs", "jobs.go"),
		filepath.Join(opts.AppPath, "app", "foundation", "container.go"),
		filepath.Join(opts.AppPath, "app", "profiles", "repo.go"),
		filepath.Join(opts.AppPath, "app", "notifications", "notifier.go"),
		filepath.Join(opts.AppPath, "app", "subscriptions", "repo.go"),
		filepath.Join(opts.AppPath, "app", "emailsubscriptions", "repo.go"),
		filepath.Join(opts.AppPath, "cmd", "worker", "main.go"),
		filepath.Join(opts.AppPath, "db", "migrate", "migrations", "00001_starter_bootstrap.sql"),
		filepath.Join(opts.AppPath, "docs", "00-index.md"),
		filepath.Join(opts.AppPath, "docs", "architecture", "01-architecture.md"),
		filepath.Join(opts.AppPath, "docs", "architecture", "08-cognitive-model.md"),
		filepath.Join(opts.AppPath, "docs", "architecture", "10-extension-zones.md"),
		filepath.Join(opts.AppPath, "go.sum"),
		filepath.Join(opts.AppPath, "static", "styles_bundle.css"),
		filepath.Join(opts.AppPath, "styles", "styles.css"),
		filepath.Join(opts.AppPath, "cmd", "web", "main.go"),
		filepath.Join(opts.AppPath, ".env"),
		filepath.Join(opts.AppPath, ".env.example"),
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

	dotEnvBytes, err := os.ReadFile(filepath.Join(opts.AppPath, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(dotEnvBytes), "UI_PROVIDER=daisy") {
		t.Fatalf(".env missing provider line:\n%s", string(dotEnvBytes))
	}

	dotEnvExampleBytes, err := os.ReadFile(filepath.Join(opts.AppPath, ".env.example"))
	if err != nil {
		t.Fatal(err)
	}
	dotEnvExample := string(dotEnvExampleBytes)
	if !strings.Contains(dotEnvExample, "UI_PROVIDER=franken") {
		t.Fatalf(".env.example missing default provider line:\n%s", dotEnvExample)
	}
	if !strings.Contains(dotEnvExample, "valid values: franken, daisy, bare") {
		t.Fatalf(".env.example missing provider options comment:\n%s", dotEnvExample)
	}

	baseLayoutBytes, err := os.ReadFile(filepath.Join(opts.AppPath, "app", "views", "web", "layouts", "base.templ"))
	if err != nil {
		t.Fatal(err)
	}
	baseLayout := string(baseLayoutBytes)
	if !strings.Contains(baseLayout, "https://cdn.jsdelivr.net/npm/flowbite") {
		t.Fatalf("base.templ missing daisy provider asset:\n%s", baseLayout)
	}
	if strings.Contains(baseLayout, "https://cdn.jsdelivr.net/npm/uikit") {
		t.Fatalf("base.templ should not include uikit for daisy:\n%s", baseLayout)
	}

	gotLayout, err := snapshotGeneratedProjectLayout(opts.AppPath)
	if err != nil {
		t.Fatalf("snapshotGeneratedProjectLayout failed: %v", err)
	}
	wantLayout := canonicalGeneratedProjectLayoutSnapshot(opts, defaultNewLayoutArtifactPaths())
	if !slices.Equal(gotLayout, wantLayout) {
		t.Fatalf("generated layout mismatch\nwant:\n%s\ngot:\n%s", strings.Join(wantLayout, "\n"), strings.Join(gotLayout, "\n"))
	}
}

func TestScaffoldNewProject_StripsStarterStyleClasses(t *testing.T) {
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

	for _, rel := range []string{
		filepath.Join("app", "views", "web", "pages", "landing.templ"),
		filepath.Join("app", "views", "web", "pages", "home_feed.templ"),
		filepath.Join("app", "views", "web", "pages", "profile.templ"),
	} {
		contentBytes, err := os.ReadFile(filepath.Join(opts.AppPath, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		content := string(contentBytes)

		if strings.Contains(content, "starter-") {
			t.Fatalf("%s should not include starter-* tokens:\n%s", rel, content)
		}
		if strings.Contains(content, `class="`) {
			t.Fatalf("%s should not include style classes:\n%s", rel, content)
		}
		if !strings.Contains(content, "data-component=") {
			t.Fatalf("%s should preserve data-component hook:\n%s", rel, content)
		}
	}

	stylesBundleBytes, err := os.ReadFile(filepath.Join(opts.AppPath, "static", "styles_bundle.css"))
	if err != nil {
		t.Fatalf("read static/styles_bundle.css: %v", err)
	}
	if strings.TrimSpace(string(stylesBundleBytes)) != "" {
		t.Fatalf("static/styles_bundle.css should be empty, got:\n%s", string(stylesBundleBytes))
	}

	stylesSourceBytes, err := os.ReadFile(filepath.Join(opts.AppPath, "styles", "styles.css"))
	if err != nil {
		t.Fatalf("read styles/styles.css: %v", err)
	}
	stylesSource := string(stylesSourceBytes)
	if strings.Contains(stylesSource, "starter-") {
		t.Fatalf("styles/styles.css should not include starter-* tokens, got:\n%s", stylesSource)
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
	for _, p := range []string{
		filepath.Join(root, "demo", "locales", "en.toml"),
		filepath.Join(root, "demo", "locales", "fr.toml"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected file %s: %v", p, err)
		}
	}
}

func TestRenderStarterTemplateFilesFromFS_MissingRoot(t *testing.T) {
	opts := NewProjectOptions{
		Name:    "demo",
		Module:  "example.com/demo",
		AppPath: "demo",
	}

	_, err := renderStarterTemplateFilesFromFS(opts, fstest.MapFS{}, starterTemplateRoot)
	if err == nil {
		t.Fatal("expected error for missing scaffold root")
	}
	if !strings.Contains(err.Error(), `missing template root "testdata/scaffold"`) {
		t.Fatalf("err = %v, want missing root diagnostic", err)
	}
}

func TestRenderStarterTemplateFilesFromFS_MissingRequiredFile(t *testing.T) {
	opts := NewProjectOptions{
		Name:    "demo",
		Module:  "example.com/demo",
		AppPath: "demo",
	}

	templateFS := fstest.MapFS{
		"testdata/scaffold/README.md": {Data: []byte("# scaffold\n")},
	}

	_, err := renderStarterTemplateFilesFromFS(opts, templateFS, starterTemplateRoot)
	if err == nil {
		t.Fatal("expected error for missing required starter file")
	}
	if !strings.Contains(err.Error(), `missing required starter file "testdata/scaffold/app/foundation/container.go"`) {
		t.Fatalf("err = %v, want missing required file diagnostic", err)
	}
}

func TestRenderStarterTemplateFilesFromFS_ValidLayout(t *testing.T) {
	opts := NewProjectOptions{
		Name:    "demo",
		Module:  "example.com/demo",
		AppPath: "demo",
	}

	templateFS := fstest.MapFS{
		"testdata/scaffold/README.md":                              {Data: []byte("# GoShip Starter\n")},
		"testdata/scaffold/app/foundation/container.go":            {Data: []byte("package foundation\n")},
		"testdata/scaffold/app/router.go":                          {Data: []byte("package app\n")},
		"testdata/scaffold/app/router_test.go":                     {Data: []byte("package app\n")},
		"testdata/scaffold/app/views/templates.go":                 {Data: []byte("package views\n")},
		"testdata/scaffold/app/views/web/pages/home_feed.templ":    {Data: []byte("templ HomeFeed(){<div>Home Feed</div>}")},
		"testdata/scaffold/app/views/web/pages/home_feed_templ.go": {Data: []byte("package pages\n")},
		"testdata/scaffold/app/views/web/pages/landing.templ":      {Data: []byte("templ Landing(){<div>GoShip Starter</div>}")},
		"testdata/scaffold/app/views/web/pages/landing_templ.go":   {Data: []byte("package pages\n")},
		"testdata/scaffold/app/views/web/pages/profile.templ":      {Data: []byte("templ Profile(){<div>Profile</div>}")},
		"testdata/scaffold/app/views/web/pages/profile_templ.go":   {Data: []byte("package pages\n")},
		"testdata/scaffold/app/web/routenames/routenames.go":       {Data: []byte("package routenames\n")},
		"testdata/scaffold/cmd/web/main.go":                        {Data: []byte("package main\n")},
		"testdata/scaffold/go.mod":                                 {Data: []byte("module github.com/leomorpho/goship/starter\n")},
		"testdata/scaffold/config/modules.yaml":                    {Data: []byte("modules: []\n")},
	}

	files, err := renderStarterTemplateFilesFromFS(opts, templateFS, starterTemplateRoot)
	if err != nil {
		t.Fatalf("renderStarterTemplateFilesFromFS error: %v", err)
	}
	if _, ok := files[filepath.Join("demo", "config", "modules.yaml")]; ok {
		t.Fatal("config/modules.yaml should be skipped from starter templates")
	}
	gotGoMod := files[filepath.Join("demo", "go.mod")]
	if !strings.Contains(gotGoMod, "module example.com/demo") {
		t.Fatalf("go.mod rewrite missing module replacement:\n%s", gotGoMod)
	}
	gotLanding := files[filepath.Join("demo", "app", "views", "web", "pages", "landing.templ")]
	if !strings.Contains(gotLanding, "Demo") {
		t.Fatalf("landing rewrite missing starter display replacement:\n%s", gotLanding)
	}
}

func TestCanonicalGeneratedProjectLayoutGolden(t *testing.T) {
	packageDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	opts := NewProjectOptions{
		Name:    "demo",
		Module:  "example.com/demo",
		AppPath: "demo",
	}
	assertCLIGoldenSnapshot(
		t,
		packageDir,
		"project_new_layout.golden",
		strings.Join(canonicalGeneratedProjectLayoutSnapshot(opts, defaultNewLayoutArtifactPaths()), "\n")+"\n",
	)

	opts.I18nEnabled = true
	opts.I18nSet = true
	assertCLIGoldenSnapshot(
		t,
		packageDir,
		"project_new_layout_i18n.golden",
		strings.Join(canonicalGeneratedProjectLayoutSnapshot(opts, defaultNewLayoutArtifactPaths()), "\n")+"\n",
	)
}
