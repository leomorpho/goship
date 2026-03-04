package ship

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
		{name: "bad name", args: []string{"-bad"}, wantErr: true},
		{name: "unknown option", args: []string{"demo", "--wat"}, wantErr: true},
		{name: "too many args", args: []string{"demo", "extra"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseNewArgs(tt.args)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestScaffoldNewProject(t *testing.T) {
	root := t.TempDir()
	opts := newProjectOptions{
		Name:    "demo",
		Module:  "example.com/demo",
		AppPath: filepath.Join(root, "demo"),
	}

	if err := scaffoldNewProject(opts); err != nil {
		t.Fatalf("scaffoldNewProject failed: %v", err)
	}

	checkFiles := []string{
		filepath.Join(opts.AppPath, "go.mod"),
		filepath.Join(opts.AppPath, "config", "modules.yaml"),
		filepath.Join(opts.AppPath, "apps", "site", "router.go"),
		filepath.Join(opts.AppPath, "apps", "db", "schema", "user.go"),
		filepath.Join(opts.AppPath, "apps", "db", "migrate", "migrations", ".gitkeep"),
		filepath.Join(opts.AppPath, "apps", "site", "web", "routenames", "routenames.go"),
		filepath.Join(opts.AppPath, "apps", "site", "views", "templates.go"),
		filepath.Join(opts.AppPath, "apps", "site", "web", "controllers", "controllers.go"),
		filepath.Join(opts.AppPath, "apps", "site", "web", "middleware", "middleware.go"),
		filepath.Join(opts.AppPath, "apps", "site", "web", "ui", "ui.go"),
		filepath.Join(opts.AppPath, "apps", "site", "web", "viewmodels", "viewmodels.go"),
		filepath.Join(opts.AppPath, "apps", "site", "jobs", "jobs.go"),
		filepath.Join(opts.AppPath, "apps", "site", "foundation", "container.go"),
		filepath.Join(opts.AppPath, "apps", "site", "app", "profiles", "repo.go"),
		filepath.Join(opts.AppPath, "apps", "site", "app", "notifications", "notifier.go"),
		filepath.Join(opts.AppPath, "apps", "site", "app", "subscriptions", "repo.go"),
		filepath.Join(opts.AppPath, "apps", "site", "app", "emailsubscriptions", "repo.go"),
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

	routerBytes, err := os.ReadFile(filepath.Join(opts.AppPath, "apps", "site", "router.go"))
	if err != nil {
		t.Fatal(err)
	}
	router := string(routerBytes)
	if !strings.Contains(router, "ship:routes:public:start") || !strings.Contains(router, "ship:routes:auth:start") {
		t.Fatalf("router markers missing:\n%s", router)
	}
}

func TestScaffoldNewProjectDryRun(t *testing.T) {
	root := t.TempDir()
	opts := newProjectOptions{
		Name:    "demo",
		Module:  "example.com/demo",
		AppPath: filepath.Join(root, "demo"),
		DryRun:  true,
	}
	if err := scaffoldNewProject(opts); err != nil {
		t.Fatalf("dry-run scaffold failed: %v", err)
	}
	if _, err := os.Stat(opts.AppPath); !os.IsNotExist(err) {
		t.Fatalf("expected no files in dry-run mode")
	}
}
