package policies

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunDoctorChecks_BoundaryImports(t *testing.T) {
	t.Run("controller db import boundary violation", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		path := filepath.Join(root, "app", "web", "controllers", "bad.go")
		content := `package controllers

import "github.com/leomorpho/goship/db/gen/user"

func _() { _ = user.FindUserByID }
`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX020")
	})

	t.Run("module isolation direct root import violation", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)
		moduleDir := filepath.Join(root, "modules", "local")
		if err := os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte("module example.com/local\n\ngo 1.23.0\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(moduleDir, "bad.go")
		content := `package local

import "github.com/leomorpho/goship/framework/core"

var _ = core.PubSub(nil)
`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX020")
	})
}

func TestRunDoctorChecks_ModuleIsolationAllowlistSuppressesKnownExceptions(t *testing.T) {
	root := t.TempDir()
	writeDoctorFixture(t, root)

	allowlistPath := filepath.Join(root, "tools", "scripts", "test")
	if err := os.MkdirAll(allowlistPath, 0o755); err != nil {
		t.Fatal(err)
	}
	moduleDir := filepath.Join(root, "modules", "local")
	if err := os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte("module example.com/local\n\ngo 1.23.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(moduleDir, "bad.go")
	content := `package local

import "github.com/leomorpho/goship/framework/core"

var _ = core.PubSub(nil)
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(allowlistPath, "module-isolation-allowlist.txt"), []byte("modules/local/bad.go\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	issues := RunDoctorChecks(root)
	mustNotContainIssueCode(t, issues, "DX020")
}

func TestCheckCanonicalRepoTopLevelPaths(t *testing.T) {
	t.Run("canonical framework repo shape passes", func(t *testing.T) {
		root := t.TempDir()
		writeCanonicalRepoFixture(t, root)

		issues := CheckCanonicalRepoTopLevelPaths(root)
		if len(issues) != 0 {
			t.Fatalf("expected no issues, got %+v", issues)
		}
	})

	t.Run("forbidden app shell path fails", func(t *testing.T) {
		root := t.TempDir()
		writeCanonicalRepoFixture(t, root)
		if err := os.MkdirAll(filepath.Join(root, "app"), 0o755); err != nil {
			t.Fatal(err)
		}

		issues := CheckCanonicalRepoTopLevelPaths(root)
		if !containsDoctorIssueMessage(issues, "forbidden top-level path present: app") {
			t.Fatalf("expected forbidden app path issue, got %+v", issues)
		}
	})

	t.Run("missing canonical runtime file fails", func(t *testing.T) {
		root := t.TempDir()
		writeCanonicalRepoFixture(t, root)
		if err := os.Remove(filepath.Join(root, "router.go")); err != nil {
			t.Fatal(err)
		}

		issues := CheckCanonicalRepoTopLevelPaths(root)
		if !containsDoctorIssueMessage(issues, "missing canonical top-level path: router.go") {
			t.Fatalf("expected missing router.go issue, got %+v", issues)
		}
	})

	t.Run("legacy app shell runtime files do not satisfy canonical root paths", func(t *testing.T) {
		root := t.TempDir()
		writeCanonicalRepoFixture(t, root)
		if err := os.Remove(filepath.Join(root, "container.go")); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(filepath.Join(root, "router.go")); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(filepath.Join(root, "schedules.go")); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(root, "app", "foundation"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(root, "app", "schedules"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "app", "foundation", "container.go"), []byte("package foundation\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "app", "router.go"), []byte("package app\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "app", "schedules", "schedules.go"), []byte("package schedules\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		issues := CheckCanonicalRepoTopLevelPaths(root)
		if !containsDoctorIssueMessage(issues, "missing canonical top-level path: container.go") {
			t.Fatalf("expected missing container.go issue, got %+v", issues)
		}
		if !containsDoctorIssueMessage(issues, "missing canonical top-level path: router.go") {
			t.Fatalf("expected missing router.go issue, got %+v", issues)
		}
		if !containsDoctorIssueMessage(issues, "missing canonical top-level path: schedules.go") {
			t.Fatalf("expected missing schedules.go issue, got %+v", issues)
		}
	})
}

func TestCheckFrameworkCIVerifyGate(t *testing.T) {
	t.Run("missing workflow fails", func(t *testing.T) {
		root := t.TempDir()
		writeCanonicalRepoFixture(t, root)
		issues := checkFrameworkCIVerifyGate(root)
		if !containsDoctorIssueMessage(issues, "missing CI workflow gate for strict framework verify profile") {
			t.Fatalf("expected missing workflow issue, got %+v", issues)
		}
	})

	t.Run("missing strict verify command fails", func(t *testing.T) {
		root := t.TempDir()
		writeCanonicalRepoFixture(t, root)
		workflowPath := filepath.Join(root, ".github", "workflows", "test.yml")
		if err := os.MkdirAll(filepath.Dir(workflowPath), 0o755); err != nil {
			t.Fatal(err)
		}
		workflow := "name: Test\njobs:\n  verify_strict:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo missing gate\n"
		if err := os.WriteFile(workflowPath, []byte(workflow), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := checkFrameworkCIVerifyGate(root)
		if !containsDoctorIssueMessage(issues, "CI workflow missing strict verify command") {
			t.Fatalf("expected missing strict command issue, got %+v", issues)
		}
	})

	t.Run("verify_strict gate present passes", func(t *testing.T) {
		root := t.TempDir()
		writeCanonicalRepoFixture(t, root)
		workflowPath := filepath.Join(root, ".github", "workflows", "test.yml")
		if err := os.MkdirAll(filepath.Dir(workflowPath), 0o755); err != nil {
			t.Fatal(err)
		}
		workflow := "name: Test\njobs:\n  verify_strict:\n    runs-on: ubuntu-latest\n    steps:\n      - run: go run ./tools/cli/ship/cmd/ship verify --profile strict\n"
		if err := os.WriteFile(workflowPath, []byte(workflow), 0o644); err != nil {
			t.Fatal(err)
		}
		issues := checkFrameworkCIVerifyGate(root)
		if len(issues) != 0 {
			t.Fatalf("expected no issues, got %+v", issues)
		}
	})
}

func writeCanonicalRepoFixture(t *testing.T, root string) {
	t.Helper()

	dirs := []string{
		filepath.Join(root, "cmd"),
		filepath.Join(root, "config"),
		filepath.Join(root, "db"),
		filepath.Join(root, "docs"),
		filepath.Join(root, "framework"),
		filepath.Join(root, "frontend"),
		filepath.Join(root, "infra"),
		filepath.Join(root, "locales"),
		filepath.Join(root, "modules"),
		filepath.Join(root, "static"),
		filepath.Join(root, "styles"),
		filepath.Join(root, "testdata"),
		filepath.Join(root, "tests"),
		filepath.Join(root, "tools", "cli", "ship"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	files := map[string]string{
		filepath.Join(root, "go.mod"):       "module example.com/goship\n\ngo 1.25\n",
		filepath.Join(root, "go.work"):      "go 1.25\n\nuse .\n",
		filepath.Join(root, "container.go"): "package goship\n",
		filepath.Join(root, "router.go"):    "package goship\n",
		filepath.Join(root, "schedules.go"): "package goship\n",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
