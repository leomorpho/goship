package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestResolveGooseDirs_CoreOnlyWhenNoFinder(t *testing.T) {
	t.Parallel()

	dirs, err := resolveGooseDirs(DBDeps{GooseDir: "db/migrate/migrations"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"db/migrate/migrations"}
	if !reflect.DeepEqual(dirs, want) {
		t.Fatalf("dirs = %v, want %v", dirs, want)
	}
}

func TestResolveGooseDirs_WithModulesManifest(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config", "modules.yaml"), []byte("modules:\n  - notifications\n  - jobs\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		filepath.Join("modules", "jobs", "db", "migrate", "migrations"),
		filepath.Join("modules", "notifications", "db", "migrate", "migrations"),
	} {
		if err := os.MkdirAll(filepath.Join(root, rel), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	dirs, err := resolveGooseDirs(DBDeps{
		GooseDir: "db/migrate/migrations",
		FindGoModule: func(string) (string, string, error) {
			return root, "example.com/test", nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// modules are normalized/sorted by runtime manifest parser.
	want := []string{
		"db/migrate/migrations",
		"modules/jobs/db/migrate/migrations",
		"modules/notifications/db/migrate/migrations",
	}
	if !reflect.DeepEqual(dirs, want) {
		t.Fatalf("dirs = %v, want %v", dirs, want)
	}
}

func TestResolveGooseDirs_MissingModuleMigrations(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config", "modules.yaml"), []byte("modules:\n  - jobs\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := resolveGooseDirs(DBDeps{
		GooseDir: "db/migrate/migrations",
		FindGoModule: func(string) (string, string, error) {
			return root, "example.com/test", nil
		},
	})
	if err == nil {
		t.Fatal("expected error for missing module migrations directory")
	}
}

func TestResolveBobgenConfigs_CoreOnlyWhenNoFinder(t *testing.T) {
	t.Parallel()

	configs, err := resolveBobgenConfigs(DBDeps{}, "db/bobgen.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"db/bobgen.yaml"}
	if !reflect.DeepEqual(configs, want) {
		t.Fatalf("configs = %v, want %v", configs, want)
	}
}

func TestResolveBobgenConfigs_WithModulesManifest(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config", "modules.yaml"), []byte("modules:\n  - notifications\n  - jobs\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		filepath.Join("modules", "jobs", "db"),
		filepath.Join("modules", "notifications", "db"),
	} {
		if err := os.MkdirAll(filepath.Join(root, rel), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, rel, "bobgen.yaml"), []byte("sql:\n  dialect: psql\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	configs, err := resolveBobgenConfigs(DBDeps{
		FindGoModule: func(string) (string, string, error) {
			return root, "example.com/test", nil
		},
	}, "db/bobgen.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{
		"db/bobgen.yaml",
		"modules/jobs/db/bobgen.yaml",
		"modules/notifications/db/bobgen.yaml",
	}
	if !reflect.DeepEqual(configs, want) {
		t.Fatalf("configs = %v, want %v", configs, want)
	}
}

func TestResolveBobgenConfigs_MissingModuleConfig(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config", "modules.yaml"), []byte("modules:\n  - jobs\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := resolveBobgenConfigs(DBDeps{
		FindGoModule: func(string) (string, string, error) {
			return root, "example.com/test", nil
		},
	}, "db/bobgen.yaml")
	if err == nil {
		t.Fatal("expected error for missing module bobgen config")
	}
}

func TestGooseDirLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		dir  string
		want string
	}{
		{name: "core", dir: "db/migrate/migrations", want: "core migrations"},
		{name: "module", dir: "modules/notifications/db/migrate/migrations", want: "module notifications migrations"},
		{name: "fallback", dir: "custom/migrations", want: "migrations: custom/migrations"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := gooseDirLabel(tt.dir); got != tt.want {
				t.Fatalf("gooseDirLabel(%q) = %q, want %q", tt.dir, got, tt.want)
			}
		})
	}
}

func TestRunGooseStatusAll_PrintsSectionHeaders(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config", "modules.yaml"), []byte("modules:\n  - notifications\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		filepath.Join("db", "migrate", "migrations"),
		filepath.Join("modules", "notifications", "db", "migrate", "migrations"),
	} {
		if err := os.MkdirAll(filepath.Join(root, rel), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	var out bytes.Buffer
	deps := DBDeps{
		Out:      &out,
		Err:      &out,
		GooseDir: "db/migrate/migrations",
		FindGoModule: func(string) (string, string, error) {
			return root, "example.com/test", nil
		},
		RunGoose: func(args ...string) int {
			return 0
		},
	}
	if code := runGooseStatusAll(deps, "sqlite://file:test.db"); code != 0 {
		t.Fatalf("runGooseStatusAll returned %d, want 0", code)
	}
	got := out.String()
	if !containsAll(got, "== core migrations ==", "== module notifications migrations ==") {
		t.Fatalf("expected section headers in output, got:\n%s", got)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(s, part) {
			return false
		}
	}
	return true
}
