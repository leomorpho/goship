package runtime

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNormalizeModules(t *testing.T) {
	t.Parallel()

	got, err := NormalizeModules([]string{" Notifications ", "jobs", "notifications", "paid_subscriptions", ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"jobs", "notifications", "paid_subscriptions"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalized modules = %v, want %v", got, want)
	}
}

func TestNormalizeModules_InvalidEntry(t *testing.T) {
	t.Parallel()

	_, err := NormalizeModules([]string{"ok", "bad/name"})
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestLoadModulesManifest(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "modules.yaml")
	if err := os.WriteFile(path, []byte("modules:\n  - Notifications\n  - jobs\n  - jobs\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := LoadModulesManifest(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"jobs", "notifications"}
	if !reflect.DeepEqual(m.Modules, want) {
		t.Fatalf("manifest modules = %v, want %v", m.Modules, want)
	}
}
