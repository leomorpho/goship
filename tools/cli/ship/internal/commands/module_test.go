package commands

import (
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
