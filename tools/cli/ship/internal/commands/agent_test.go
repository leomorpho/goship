package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	policies "github.com/leomorpho/goship/tools/cli/ship/internal/policies"
)

func TestAgentPolicySetupAndCheck(t *testing.T) {
	root := t.TempDir()
	writeGoModule(t, root)
	policyPath := filepath.Join(root, "tools", "agent-policy", "allowed-commands.yaml")
	if err := os.MkdirAll(filepath.Dir(policyPath), 0o755); err != nil {
		t.Fatal(err)
	}
	policy := strings.Join([]string{
		"version: 1",
		"commands:",
		"  - id: go_test",
		"    description: Run tests.",
		"    prefix: [\"go\", \"test\"]",
		"  - id: ship_doctor",
		"    description: Run doctor.",
		"    prefix: [\"ship\", \"doctor\"]",
		"",
	}, "\n")
	if err := os.WriteFile(policyPath, []byte(policy), 0o644); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	if code := policies.RunPolicySetup(out, errOut, root); code != 0 {
		t.Fatalf("setup code=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "agent setup complete") {
		t.Fatalf("unexpected setup output: %q", out.String())
	}

	out.Reset()
	errOut.Reset()
	if code := policies.RunPolicyCheck(out, errOut, root); code != 0 {
		t.Fatalf("check code=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "in sync") {
		t.Fatalf("unexpected check output: %q", out.String())
	}
}

func TestAgentPolicyCheckDetectsDrift(t *testing.T) {
	root := t.TempDir()
	writeGoModule(t, root)
	policyPath := filepath.Join(root, "tools", "agent-policy", "allowed-commands.yaml")
	if err := os.MkdirAll(filepath.Dir(policyPath), 0o755); err != nil {
		t.Fatal(err)
	}
	policy := strings.Join([]string{
		"version: 1",
		"commands:",
		"  - id: go_test",
		"    description: Run tests.",
		"    prefix: [\"go\", \"test\"]",
		"",
	}, "\n")
	if err := os.WriteFile(policyPath, []byte(policy), 0o644); err != nil {
		t.Fatal(err)
	}

	if code := policies.RunPolicySetup(&bytes.Buffer{}, &bytes.Buffer{}, root); code != 0 {
		t.Fatalf("setup failed")
	}
	if err := os.WriteFile(filepath.Join(root, "tools", "agent-policy", "generated", "codex-prefixes.txt"), []byte("stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	if code := policies.RunPolicyCheck(out, errOut, root); code != 1 {
		t.Fatalf("check code=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "out of sync") {
		t.Fatalf("unexpected check stderr: %q", errOut.String())
	}
}

func TestAgentStatus(t *testing.T) {
	root := t.TempDir()
	writeGoModule(t, root)
	writeAgentPolicyFixture(t, root)
	if code := policies.RunPolicySetup(&bytes.Buffer{}, &bytes.Buffer{}, root); code != 0 {
		t.Fatalf("setup failed")
	}

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	t.Run("in-sync when config contains all prefixes", func(t *testing.T) {
		cfg := filepath.Join(root, "codex-local.txt")
		content := "go test\nship doctor\n"
		if err := os.WriteFile(cfg, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunAgent([]string{"status", "--codex-file", cfg}, AgentDeps{Out: out, Err: errOut, FindGoModule: findGoModuleTest})
		if code != 0 {
			t.Fatalf("status code=%d stderr=%s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "- codex: in-sync") {
			t.Fatalf("unexpected status output: %s", out.String())
		}
	})

	t.Run("drifted when config has subset", func(t *testing.T) {
		cfg := filepath.Join(root, "codex-drifted.txt")
		content := "go test\n"
		if err := os.WriteFile(cfg, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunAgent([]string{"status", "--codex-file", cfg}, AgentDeps{Out: out, Err: errOut, FindGoModule: findGoModuleTest})
		if code != 0 {
			t.Fatalf("status code=%d stderr=%s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "- codex: drifted") {
			t.Fatalf("unexpected status output: %s", out.String())
		}
	})
}

func writeAgentPolicyFixture(t *testing.T, root string) {
	t.Helper()
	policyPath := filepath.Join(root, "tools", "agent-policy", "allowed-commands.yaml")
	if err := os.MkdirAll(filepath.Dir(policyPath), 0o755); err != nil {
		t.Fatal(err)
	}
	policy := strings.Join([]string{
		"version: 1",
		"commands:",
		"  - id: go_test",
		"    description: Run tests.",
		"    prefix: [\"go\", \"test\"]",
		"  - id: ship_doctor",
		"    description: Run doctor.",
		"    prefix: [\"ship\", \"doctor\"]",
		"",
	}, "\n")
	if err := os.WriteFile(policyPath, []byte(policy), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeGoModule(t *testing.T, root string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func findGoModuleTest(start string) (string, string, error) {
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
