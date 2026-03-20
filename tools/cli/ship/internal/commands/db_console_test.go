package commands

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestRunDBConsole_Postgres(t *testing.T) {
	t.Parallel()

	var gotName string
	var gotArgs []string
	code := RunDB([]string{"console"}, DBDeps{
		Out:             &bytes.Buffer{},
		Err:             &bytes.Buffer{},
		ResolveDBURL:    func() (string, error) { return "postgres://user:pass@localhost:5432/app?sslmode=disable", nil },
		ResolveDBDriver: func() (string, error) { return "postgres", nil },
		RunCmd: func(name string, args ...string) int {
			gotName = name
			gotArgs = append([]string{}, args...)
			return 0
		},
	})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if gotName != "psql" {
		t.Fatalf("command = %q, want psql", gotName)
	}
	wantArgs := []string{"postgres://user:pass@localhost:5432/app?sslmode=disable"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("args = %v, want %v", gotArgs, wantArgs)
	}
}

func TestRunDBConsole_SQLite(t *testing.T) {
	t.Parallel()

	var gotName string
	var gotArgs []string
	code := RunDB([]string{"console"}, DBDeps{
		Out:             &bytes.Buffer{},
		Err:             &bytes.Buffer{},
		ResolveDBURL:    func() (string, error) { return "sqlite://.local/db/main.db?_journal=WAL&_timeout=5000&_fk=true", nil },
		ResolveDBDriver: func() (string, error) { return "sqlite", nil },
		RunCmd: func(name string, args ...string) int {
			gotName = name
			gotArgs = append([]string{}, args...)
			return 0
		},
	})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if gotName != "sqlite3" {
		t.Fatalf("command = %q, want sqlite3", gotName)
	}
	wantArgs := []string{".local/db/main.db"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("args = %v, want %v", gotArgs, wantArgs)
	}
}

func TestRunDBConsole_MySQL(t *testing.T) {
	t.Parallel()

	var gotName string
	var gotArgs []string
	code := RunDB([]string{"console"}, DBDeps{
		Out:             &bytes.Buffer{},
		Err:             &bytes.Buffer{},
		ResolveDBURL:    func() (string, error) { return "mysql://app:secret@db.local:3306/goship?tls=skip-verify", nil },
		ResolveDBDriver: func() (string, error) { return "mysql", nil },
		RunCmd: func(name string, args ...string) int {
			gotName = name
			gotArgs = append([]string{}, args...)
			return 0
		},
	})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if gotName != "mysql" {
		t.Fatalf("command = %q, want mysql", gotName)
	}
	wantArgs := []string{"--host", "db.local", "--port", "3306", "--user", "app", "--password=secret", "goship"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("args = %v, want %v", gotArgs, wantArgs)
	}
}

func TestRunDBConsole_InfersDriverFromURL(t *testing.T) {
	t.Parallel()

	var gotName string
	code := RunDB([]string{"console"}, DBDeps{
		Out:          &bytes.Buffer{},
		Err:          &bytes.Buffer{},
		ResolveDBURL: func() (string, error) { return "postgres://user:pass@localhost:5432/app", nil },
		RunCmd: func(name string, args ...string) int {
			gotName = name
			return 0
		},
	})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if gotName != "psql" {
		t.Fatalf("command = %q, want psql", gotName)
	}
}

func TestRunDBConsole_DBURLResolutionError(t *testing.T) {
	t.Parallel()

	errOut := &bytes.Buffer{}
	code := RunDB([]string{"console"}, DBDeps{
		Out:          &bytes.Buffer{},
		Err:          errOut,
		ResolveDBURL: func() (string, error) { return "", errors.New("missing url") },
		RunCmd:       func(string, ...string) int { return 0 },
	})
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "failed to resolve database URL") {
		t.Fatalf("stderr = %q, want DB URL resolution error", errOut.String())
	}
}

func TestRunDBConsole_DBDriverResolutionError(t *testing.T) {
	t.Parallel()

	errOut := &bytes.Buffer{}
	code := RunDB([]string{"console"}, DBDeps{
		Out:             &bytes.Buffer{},
		Err:             errOut,
		ResolveDBURL:    func() (string, error) { return "postgres://user:pass@localhost:5432/app", nil },
		ResolveDBDriver: func() (string, error) { return "", errors.New("bad driver") },
		RunCmd:          func(string, ...string) int { return 0 },
	})
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "failed to resolve database driver") {
		t.Fatalf("stderr = %q, want DB driver resolution error", errOut.String())
	}
}

func TestRunDBConsole_Usage(t *testing.T) {
	t.Parallel()

	errOut := &bytes.Buffer{}
	code := RunDB([]string{"console", "extra"}, DBDeps{
		Out:          &bytes.Buffer{},
		Err:          errOut,
		ResolveDBURL: func() (string, error) { return "", nil },
		RunCmd:       func(string, ...string) int { return 0 },
	})
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "usage: ship db:console") {
		t.Fatalf("stderr = %q, want usage", errOut.String())
	}
}
