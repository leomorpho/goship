package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDescribe(t *testing.T) {
	root := t.TempDir()
	writeDescribeFixture(t, root)

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	t.Run("pretty json contains live sections", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		if code := RunDescribe([]string{"--pretty"}, DescribeDeps{Out: out, Err: errOut, FindGoModule: findDescribeGoModule}); code != 0 {
			t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
		}
		if errOut.Len() != 0 {
			t.Fatalf("stderr = %q, want empty", errOut.String())
		}
		if !strings.Contains(out.String(), "\n  \"routes\"") {
			t.Fatalf("output = %q, want indented json", out.String())
		}

		var payload describeResult
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("decode json: %v", err)
		}
		if len(payload.Routes) != 3 {
			t.Fatalf("routes len = %d, want 3", len(payload.Routes))
		}
		if payload.Routes[1].Handler == "" {
			t.Fatalf("route handler = empty, want parsed handler")
		}
		if len(payload.Controllers) != 1 || payload.Controllers[0].Name != "login" {
			t.Fatalf("controllers = %+v, want login controller", payload.Controllers)
		}
		if len(payload.ViewModels) != 1 || payload.ViewModels[0].Name != "LoginPage" {
			t.Fatalf("viewmodels = %+v, want LoginPage", payload.ViewModels)
		}
		if len(payload.Components) != 1 || payload.Components[0].DataComponent != "navbar" {
			t.Fatalf("components = %+v, want navbar component", payload.Components)
		}
		if len(payload.Modules) != 1 || payload.Modules[0].ID != "notifications" {
			t.Fatalf("modules = %+v, want notifications module", payload.Modules)
		}
		if len(payload.Migrations) != 2 {
			t.Fatalf("migrations len = %d, want 2", len(payload.Migrations))
		}
		if len(payload.DBTables) == 0 || payload.DBTables[0] != "users" {
			t.Fatalf("db tables = %+v, want users", payload.DBTables)
		}
	})

	t.Run("help", func(t *testing.T) {
		out := &bytes.Buffer{}
		if code := RunDescribe([]string{"--help"}, DescribeDeps{Out: out, Err: &bytes.Buffer{}, FindGoModule: findDescribeGoModule}); code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if !strings.Contains(out.String(), "ship describe commands:") {
			t.Fatalf("stdout = %q, want help", out.String())
		}
	})
}

func writeDescribeFixture(t *testing.T, root string) {
	t.Helper()
	dirs := []string{
		filepath.Join(root, "app", "web", "controllers"),
		filepath.Join(root, "app", "web", "viewmodels"),
		filepath.Join(root, "app", "views", "web", "components"),
		filepath.Join(root, "config"),
		filepath.Join(root, "db", "queries"),
		filepath.Join(root, "db", "migrate", "migrations"),
		filepath.Join(root, "modules", "notifications", "db", "migrate", "migrations"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	files := map[string]string{
		filepath.Join(root, "go.mod"):                 "module example.com/describe\n\ngo 1.25\n",
		filepath.Join(root, "config", "modules.yaml"): "modules:\n  - notifications\n",
		filepath.Join(root, "app", "router.go"): `package goship

type group struct{}

func (group) GET(string, ...any) group    { return group{} }
func (group) POST(string, ...any) group   { return group{} }
func (group) DELETE(string, ...any) group { return group{} }

func registerPublicRoutes(g group) {
	login := struct{ Get any }{}
	g.GET("/login", login.Get)
}

func registerAuthRoutes(onboardedGroup group) {
	login := struct{ Post any }{}
	onboardedGroup.POST("/login", login.Post)
}

func registerExternalRoutes(e group) {
	login := struct{ Delete any }{}
	e.DELETE("/logout", login.Delete)
}
`,
		filepath.Join(root, "app", "web", "controllers", "login.go"): `package controllers

import "github.com/labstack/echo/v4"

type login struct{}

func (l *login) Get(ctx echo.Context) error { return nil }
func (l *login) Post(ctx echo.Context) error { return nil }
`,
		filepath.Join(root, "app", "web", "viewmodels", "login.go"): `package viewmodels

type LoginPage struct {
	Email string
	Error string
}
`,
		filepath.Join(root, "app", "views", "web", "components", "navbar.templ"): `<nav data-component="navbar"></nav>
`,
		filepath.Join(root, "db", "queries", "auth.sql"): `-- name: create_users
CREATE TABLE users (
	id INTEGER PRIMARY KEY
);

-- name: list_users
SELECT id FROM users;
`,
		filepath.Join(root, "db", "migrate", "migrations", "20260101000000_init.sql"):                        "-- init\n",
		filepath.Join(root, "modules", "notifications", "db", "migrate", "migrations", "20260102000000.sql"): "-- init\n",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func findDescribeGoModule(start string) (string, string, error) {
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
