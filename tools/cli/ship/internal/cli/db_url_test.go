package ship

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveAtlasDBURL_PrefersEnv(t *testing.T) {
	prev := os.Getenv("DATABASE_URL")
	t.Cleanup(func() { _ = os.Setenv("DATABASE_URL", prev) })
	if err := os.Setenv("DATABASE_URL", "postgres://env-only"); err != nil {
		t.Fatal(err)
	}

	got, err := resolveAtlasDBURL()
	if err != nil {
		t.Fatalf("resolveAtlasDBURL error = %v", err)
	}
	if got != "postgres://env-only" {
		t.Fatalf("db url = %q, want %q", got, "postgres://env-only")
	}
}

func TestResolveAtlasDBURL_PrefersDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://primary")
	t.Setenv("PAGODA_DATABASE_URL", "postgres://secondary")
	got, err := resolveAtlasDBURL()
	if err != nil {
		t.Fatalf("resolveAtlasDBURL error = %v", err)
	}
	if got != "postgres://primary" {
		t.Fatalf("db url = %q, want %q", got, "postgres://primary")
	}
}

func TestResolveAtlasDBURL_RejectsLegacyPagodaDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("PAGODA_DATABASE_URL", "postgres://pagoda-env")
	_, err := resolveAtlasDBURL()
	if err == nil {
		t.Fatal("expected error for PAGODA_DATABASE_URL, got nil")
	}
	if !strings.Contains(err.Error(), "PAGODA_DATABASE_URL is not supported") {
		t.Fatalf("error = %q, want explicit legacy var message", err.Error())
	}
}

func TestResolveAtlasDBURL_FromConfig(t *testing.T) {
	root := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DATABASE_URL", "")
	t.Setenv("PAGODA_DATABASE_URL", "")
	t.Setenv("APP_ENV", "local")

	if err := os.MkdirAll(filepath.Join(root, "config", "environments"), 0o755); err != nil {
		t.Fatal(err)
	}
	application := `
app:
  environment: "local"
database:
  dbMode: "standalone"
  hostname: "db.local"
  port: 5432
  user: "app"
  password: "secret"
  databaseNameLocal: "goship_db"
  databaseNameProd: "goship_prod"
  testDatabase: "goship_test"
  sslMode: "disable"
`
	if err := os.WriteFile(filepath.Join(root, "config", "application.yaml"), []byte(application), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config", "environments", "local.yaml"), []byte("app:\n  environment: local\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := resolveAtlasDBURL()
	if err != nil {
		t.Fatalf("resolveAtlasDBURL error = %v", err)
	}
	if !strings.Contains(got, "db.local:5432") {
		t.Fatalf("db url = %q, want host/port", got)
	}
	if !strings.Contains(got, "/goship_db") {
		t.Fatalf("db url = %q, want local database name", got)
	}
}

func TestResolveAtlasDBURL_EmbeddedModeError(t *testing.T) {
	root := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DATABASE_URL", "")
	t.Setenv("PAGODA_DATABASE_URL", "")
	t.Setenv("APP_ENV", "local")

	if err := os.MkdirAll(filepath.Join(root, "config", "environments"), 0o755); err != nil {
		t.Fatal(err)
	}
	application := `
app:
  environment: "local"
database:
  dbMode: "embedded"
  embeddedConnection: "dbs/main.db"
`
	if err := os.WriteFile(filepath.Join(root, "config", "application.yaml"), []byte(application), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = resolveAtlasDBURL()
	if err == nil {
		t.Fatal("expected error for embedded mode, got nil")
	}
	if !strings.Contains(err.Error(), "embedded") {
		t.Fatalf("error = %q, want embedded message", err.Error())
	}
}

func TestResolveAtlasDBURL_UsesProductionDatabaseName(t *testing.T) {
	root := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DATABASE_URL", "")
	t.Setenv("PAGODA_DATABASE_URL", "")
	t.Setenv("APP_ENV", "production")

	if err := os.MkdirAll(filepath.Join(root, "config", "environments"), 0o755); err != nil {
		t.Fatal(err)
	}
	application := `
app:
  environment: "local"
database:
  dbMode: "standalone"
  hostname: "db.local"
  port: 5432
  user: "app"
  password: "secret"
  databaseNameLocal: "goship_db"
  databaseNameProd: "goship_prod"
  testDatabase: "goship_test"
  sslMode: "disable"
`
	if err := os.WriteFile(filepath.Join(root, "config", "application.yaml"), []byte(application), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := resolveAtlasDBURL()
	if err != nil {
		t.Fatalf("resolveAtlasDBURL error = %v", err)
	}
	if !strings.Contains(got, "/goship_prod") {
		t.Fatalf("db url = %q, want production database name", got)
	}
}

func TestResolveAtlasDBURL_UsesTestDatabaseName(t *testing.T) {
	root := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DATABASE_URL", "")
	t.Setenv("PAGODA_DATABASE_URL", "")
	t.Setenv("APP_ENV", "test")

	if err := os.MkdirAll(filepath.Join(root, "config", "environments"), 0o755); err != nil {
		t.Fatal(err)
	}
	application := `
app:
  environment: "local"
database:
  dbMode: "standalone"
  hostname: "db.local"
  port: 5432
  user: "app"
  password: "secret"
  databaseNameLocal: "goship_db"
  databaseNameProd: "goship_prod"
  testDatabase: "goship_test"
  sslMode: "disable"
`
	if err := os.WriteFile(filepath.Join(root, "config", "application.yaml"), []byte(application), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := resolveAtlasDBURL()
	if err != nil {
		t.Fatalf("resolveAtlasDBURL error = %v", err)
	}
	if !strings.Contains(got, "/goship_test") {
		t.Fatalf("db url = %q, want test database name", got)
	}
}

func TestResolveComposeCommandWith_DockerComposeAvailable(t *testing.T) {
	lookPath := func(bin string) (string, error) {
		if bin == "docker-compose" {
			return "/usr/bin/docker-compose", nil
		}
		return "", errors.New("missing")
	}
	got, err := resolveComposeCommandWith(lookPath, func() error { return nil })
	if err != nil {
		t.Fatalf("resolveComposeCommandWith error = %v", err)
	}
	if strings.Join(got, " ") != "docker-compose" {
		t.Fatalf("compose command = %v, want docker-compose", got)
	}
}

func TestResolveComposeCommandWith_DockerComposeSubcommandAvailable(t *testing.T) {
	lookPath := func(bin string) (string, error) {
		if bin == "docker" {
			return "/usr/bin/docker", nil
		}
		return "", errors.New("missing")
	}
	got, err := resolveComposeCommandWith(lookPath, func() error { return nil })
	if err != nil {
		t.Fatalf("resolveComposeCommandWith error = %v", err)
	}
	if strings.Join(got, " ") != "docker compose" {
		t.Fatalf("compose command = %v, want docker compose", got)
	}
}

func TestResolveComposeCommandWith_NoComposeAvailable(t *testing.T) {
	lookPath := func(string) (string, error) {
		return "", errors.New("missing")
	}
	_, err := resolveComposeCommandWith(lookPath, func() error { return errors.New("no compose") })
	if err == nil {
		t.Fatal("expected compose resolution error, got nil")
	}
}
