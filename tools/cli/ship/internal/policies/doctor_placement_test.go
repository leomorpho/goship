package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDoctorChecks_CanonicalPlacement(t *testing.T) {
	t.Run("handler outside controllers is rejected", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		path := filepath.Join(root, "app", "web", "bad_handler.go")
		content := `package web

import "github.com/labstack/echo/v4"

type badHandler struct{}

func (h *badHandler) Get(ctx echo.Context) error { return nil }
`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX021")
	})

	t.Run("free helper with echo.Context is ignored", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		path := filepath.Join(root, "app", "web", "helpers.go")
		content := `package web

import "github.com/labstack/echo/v4"

func helper(ctx echo.Context) error { return nil }
`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		for _, issue := range issues {
			if issue.Code == "DX021" && issue.File == "app/web/helpers.go" {
				t.Fatalf("unexpected DX021 issue for helper function: %+v", issue)
			}
		}
	})

	t.Run("route registration outside router is rejected", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		path := filepath.Join(root, "app", "web", "rogue_routes.go")
		content := `package web

type router struct{}

func (router) GET(string, ...any) {}

func registerBadRoutes(r router) {
	r.GET("/bad")
}
`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX021")
	})

	t.Run("route registration without literal path is ignored", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		path := filepath.Join(root, "app", "web", "rogue_routes.go")
		content := `package web

type router struct{}

func (router) GET(string, ...any) {}

func registerBadRoutes(r router) {
	const path = "/bad"
	r.GET(path)
}
`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		for _, issue := range issues {
			if issue.Code == "DX021" && issue.File == "app/web/rogue_routes.go" {
				t.Fatalf("unexpected DX021 issue for non-literal route: %+v", issue)
			}
		}
	})

	t.Run("inline sql outside store layer is rejected", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		path := filepath.Join(root, "app", "profile", "rogue_sql.go")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		content := `package profile

func bad(db interface{ Exec(string, ...any) (any, error) }) {
	_, _ = db.Exec("SELECT 1")
}
`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX021")
	})

	t.Run("migration outside canonical directory is rejected", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		path := filepath.Join(root, "scratch", "20260306120000_add_users.sql")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("-- migration"), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX021")
	})

	t.Run("config struct outside config.go is rejected", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		path := filepath.Join(root, "config", "extra.go")
		content := `package config

type ExtraConfig struct{}
`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX021")
	})

	t.Run("static route wiring stays allowed", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		path := filepath.Join(root, "app", "web", "wiring.go")
		content := `package web

type webGroup struct{}

func (webGroup) GET(string, ...any) {}

type container struct {
	Web webGroup
}

func RegisterStaticRoutes(c *container) {
	c.Web.GET("/service-worker.js")
}
`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		for _, issue := range issues {
			if issue.Code == "DX021" && issue.File == "app/web/wiring.go" {
				t.Fatalf("unexpected DX021 issue for allowed static route file: %+v", issue)
			}
		}
	})

	t.Run("store file with inline sql stays allowed", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		path := filepath.Join(root, "app", "profile", "good_store.go")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		content := `package profile

func ok(db interface{ Exec(string, ...any) (any, error) }) {
	_, _ = db.Exec("SELECT 1")
}
`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		for _, issue := range issues {
			if issue.Code == "DX021" && issue.File == "app/profile/good_store.go" {
				t.Fatalf("unexpected DX021 issue for allowed store file: %+v", issue)
			}
		}
	})

	t.Run("modules admin routes keep inline sql out of route layer", func(t *testing.T) {
		root := findRepoRoot(t)
		issues := RunDoctorChecks(root)
		for _, issue := range issues {
			if issue.Code == "DX021" && issue.File == "modules/admin/routes.go" {
				t.Fatalf("unexpected DX021 issue for modules/admin/routes.go: %+v", issue)
			}
		}
	})
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if hasFile(filepath.Join(dir, "go.mod")) &&
			hasFile(filepath.Join(dir, "app", "router.go")) &&
			hasFile(filepath.Join(dir, "modules", "admin", "routes.go")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir || strings.TrimSpace(parent) == "" {
			t.Fatalf("repo root not found from %s", dir)
		}
		dir = parent
	}
}
