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
