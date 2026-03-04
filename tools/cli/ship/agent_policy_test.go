package ship

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if code := runAgentPolicySetup(out, errOut, root); code != 0 {
		t.Fatalf("setup code=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "agent setup complete") {
		t.Fatalf("unexpected setup output: %q", out.String())
	}

	out.Reset()
	errOut.Reset()
	if code := runAgentPolicyCheck(out, errOut, root); code != 0 {
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

	if code := runAgentPolicySetup(&bytes.Buffer{}, &bytes.Buffer{}, root); code != 0 {
		t.Fatalf("setup failed")
	}
	if err := os.WriteFile(filepath.Join(root, "tools", "agent-policy", "generated", "codex-prefixes.txt"), []byte("stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	if code := runAgentPolicyCheck(out, errOut, root); code != 1 {
		t.Fatalf("check code=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "out of sync") {
		t.Fatalf("unexpected check stderr: %q", errOut.String())
	}
}

func writeGoModule(t *testing.T, root string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}
