package commands

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAddArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    string
		wantDry bool
		wantErr bool
	}{
		{name: "simple", args: []string{"Notifications"}, want: "notifications"},
		{name: "dry run", args: []string{"notifications", "--dry-run"}, want: "notifications", wantDry: true},
		{name: "unknown option", args: []string{"notifications", "--unknown"}, wantErr: true},
		{name: "missing name", args: []string{"--dry-run"}, wantErr: true},
		{name: "extra positional", args: []string{"notifications", "extra"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, dry, err := parseModuleArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseAddArgs error = %v, wantErr=%v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Fatalf("module = %q, want %q", got, tt.want)
			}
			if dry != tt.wantDry {
				t.Fatalf("dry run = %v, want %v", dry, tt.wantDry)
			}
		})
	}
}

func TestInsertBetweenMarkers(t *testing.T) {
	content := "start\n// ship:marker:start\nexisting\n// ship:marker:end\nend\n"
	snippet := "\t// ship:module:test\n"
	updated, changed, err := insertBetweenMarkers(content, "// ship:marker:start", "// ship:marker:end", snippet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatalf("expected change")
	}
	if !strings.Contains(updated, snippet) {
		t.Fatalf("snippet missing")
	}

	// second insertion should be no-op.
	updated2, changed2, err := insertBetweenMarkers(updated, "// ship:marker:start", "// ship:marker:end", snippet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed2 {
		t.Fatalf("expected no change on reinsert")
	}
	if updated2 != updated {
		t.Fatalf("content mutated unexpectedly")
	}
}

func TestBuildModulesManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "modules.yaml")
	changed, content, err := buildModulesManifest(path, "notifications")
	if err != nil {
		t.Fatalf("build manifest: %v", err)
	}
	if !changed {
		t.Fatalf("expected manifest changed")
	}
	if !strings.Contains(content, "- notifications") {
		t.Fatalf("module entry missing: %s", content)
	}
}

func TestRemoveSnippetFromContent(t *testing.T) {
	content := "start\n\t// ship:module:test\n\t// TODO: remove me\nend\n"
	updated, removed := removeSnippetFromContent(content, `
	// ship:module:test
	// TODO: remove me
`)
	if !removed {
		t.Fatal("expected snippet removal")
	}
	if strings.Contains(updated, "remove me") {
		t.Fatalf("snippet not removed: %s", updated)
	}
}

func TestRemoveModuleFromManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "modules.yaml")
	if err := os.WriteFile(path, []byte(modulesManifestHeader+"  - notifications\n  - jobs\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	removed, content, err := removeModuleFromManifest(path, "notifications")
	if err != nil {
		t.Fatalf("remove manifest: %v", err)
	}
	if !removed {
		t.Fatal("expected manifest change")
	}
	if strings.Contains(content, "- notifications") {
		t.Fatalf("module still present: %s", content)
	}
	if !strings.Contains(content, "- jobs") {
		t.Fatalf("unexpected manifest: %s", content)
	}
}

func TestApplyModuleAdd_TwoFactor(t *testing.T) {
	root := t.TempDir()

	containerPath := filepath.Join(root, "app", "foundation", "container.go")
	if err := os.MkdirAll(filepath.Dir(containerPath), 0o755); err != nil {
		t.Fatal(err)
	}
	containerContent := `package foundation

func NewContainer() *Container {
	c := &Container{}
	// ship:container:start
	// ship:container:end
	return c
}

type Container struct{}
`
	if err := os.WriteFile(containerPath, []byte(containerContent), 0o644); err != nil {
		t.Fatal(err)
	}

	routerPath := filepath.Join(root, "app", "router.go")
	if err := os.MkdirAll(filepath.Dir(routerPath), 0o755); err != nil {
		t.Fatal(err)
	}
	routerContent := `package goship

func registerPublicRoutes() {
	// ship:routes:public:start
	// ship:routes:public:end
}

func registerAuthRoutes() {
	// ship:routes:auth:start
	// ship:routes:auth:end
}

func registerExternalRoutes() {
	// ship:routes:external:start
	// ship:routes:external:end
}
`
	if err := os.WriteFile(routerPath, []byte(routerContent), 0o644); err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(root, "config", "modules.yaml")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, []byte(modulesManifestHeader), 0o644); err != nil {
		t.Fatal(err)
	}

	info, ok := moduleCatalog["2fa"]
	if !ok {
		t.Fatal("expected 2fa in module catalog")
	}
	if err := applyModuleAdd(root, info, false, io.Discard); err != nil {
		t.Fatalf("applyModuleAdd error: %v", err)
	}

	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(manifest), "- 2fa") {
		t.Fatalf("expected 2fa in modules manifest, got:\n%s", string(manifest))
	}
}
