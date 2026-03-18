package commands

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestModuleIsolationContract_ReportsModuleAndFileContext_RedSpec(t *testing.T) {
	root := t.TempDir()
	repoRoot := repoRootFromCommandsTest(t)
	if err := os.MkdirAll(filepath.Join(root, "modules", "local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "modules", "local", "go.mod"), []byte("module example.com/local\n\ngo 1.23.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.25.6\n\nuse ./modules/local\nuse "+repoRoot+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "modules", "local", "bad.go"), []byte(`package local

import (
	"github.com/leomorpho/goship/framework/core"
)

var _ = core.PubSub(nil)
`), 0o644); err != nil {
		t.Fatal(err)
	}

	script := filepath.Join(repoRootFromCommandsTest(t), "tools", "scripts", "check-module-isolation.sh")
	cmd := exec.Command("bash", script)
	cmd.Env = append(os.Environ(), "ROOT_DIR="+root)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err == nil {
		t.Fatal("expected module isolation script to fail for a direct root import")
	}

	text := out.String()
	for _, token := range []string{
		"module=modules/local",
		"file=modules/local/bad.go",
		"github.com/leomorpho/goship/framework/core",
	} {
		if !strings.Contains(text, token) {
			t.Fatalf("module isolation output missing %q:\n%s", token, text)
		}
	}
}

func TestModuleIsolationContract_FailsOnStaleAllowlistEntry_RedSpec(t *testing.T) {
	root := t.TempDir()
	repoRoot := repoRootFromCommandsTest(t)
	if err := os.MkdirAll(filepath.Join(root, "modules", "local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "modules", "local", "go.mod"), []byte("module example.com/local\n\ngo 1.23.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.25.6\n\nuse ./modules/local\nuse "+repoRoot+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "modules", "local", "clean.go"), []byte(`package local

import (
	"fmt"
)

var _ = fmt.Sprintf
`), 0o644); err != nil {
		t.Fatal(err)
	}
	allowlistDir := filepath.Join(root, "tools", "scripts", "test")
	if err := os.MkdirAll(allowlistDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(allowlistDir, "module-isolation-allowlist.txt"), []byte("modules/local/clean.go\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	script := filepath.Join(repoRootFromCommandsTest(t), "tools", "scripts", "check-module-isolation.sh")
	cmd := exec.Command("bash", script)
	cmd.Env = append(os.Environ(), "ROOT_DIR="+root)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err == nil {
		t.Fatal("expected module isolation script to fail for a stale allowlist entry")
	}

	text := out.String()
	for _, token := range []string{
		"stale allowlist entry",
		"modules/local/clean.go",
	} {
		if !strings.Contains(text, token) {
			t.Fatalf("module isolation output missing %q:\n%s", token, text)
		}
	}
}
