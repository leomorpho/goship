package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	policies "github.com/leomorpho/goship/tools/cli/ship/internal/policies"
)

func TestRunVerifyDefaultGeneratedAppRequiresViews(t *testing.T) {
	t.Parallel()

	root := scaffoldVerifyTestProject(t, false)
	if err := os.RemoveAll(filepath.Join(root, "app", "views")); err != nil {
		t.Fatalf("os.RemoveAll(app/views) error = %v", err)
	}

	var out bytes.Buffer
	code := RunVerify([]string{"--profile", "fast"}, VerifyDeps{
		Out:          &out,
		Err:          &out,
		FindGoModule: func(string) (string, string, error) { return root, "", nil },
		RunStep:      verifyTestRunStep,
		RunDoctor:    func() (int, string, error) { return 0, `{"ok":true,"issues":[]}`, nil },
	})
	if code == 0 {
		t.Fatalf("RunVerify() exit code = %d, want non-zero\n%s", code, out.String())
	}
	if !strings.Contains(out.String(), "missing required directory: app/views") {
		t.Fatalf("RunVerify() output missing app/views requirement\n%s", out.String())
	}
}

func TestRunVerifyAPIOnlyGeneratedAppAllowsMissingViews(t *testing.T) {
	t.Parallel()

	root := scaffoldVerifyTestProject(t, true)
	if err := os.RemoveAll(filepath.Join(root, "app", "views")); err != nil {
		t.Fatalf("os.RemoveAll(app/views) error = %v", err)
	}

	var out bytes.Buffer
	code := RunVerify([]string{"--profile", "fast"}, VerifyDeps{
		Out:          &out,
		Err:          &out,
		FindGoModule: func(string) (string, string, error) { return root, "", nil },
		RunStep:      verifyTestRunStep,
		RunDoctor:    func() (int, string, error) { return 0, `{"ok":true,"issues":[]}`, nil },
	})
	if code != 0 {
		t.Fatalf("RunVerify() exit code = %d, want zero\n%s", code, out.String())
	}
	if !strings.Contains(out.String(), "verify passed") {
		t.Fatalf("RunVerify() output missing success marker\n%s", out.String())
	}
}

func TestRunVerifyFrameworkRepoUsesCanonicalRepoLayoutChecks(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "framework", "core"), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(framework/core) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "tools", "cli", "ship"), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(tools/cli/ship) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/framework\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(go.mod) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.24.0\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(go.work) error = %v", err)
	}

	var out bytes.Buffer
	code := RunVerify([]string{"--profile", "fast"}, VerifyDeps{
		Out:          &out,
		Err:          &out,
		FindGoModule: func(string) (string, string, error) { return root, "", nil },
		RunStep:      verifyTestRunStep,
		RunDoctor:    func() (int, string, error) { return 0, `{"ok":true,"issues":[]}`, nil },
	})
	if code == 0 {
		t.Fatalf("RunVerify() exit code = %d, want non-zero\n%s", code, out.String())
	}
	if !strings.Contains(out.String(), "missing canonical top-level path: modules") {
		t.Fatalf("RunVerify() output missing canonical repo layout issue\n%s", out.String())
	}
}

func scaffoldVerifyTestProject(t *testing.T, apiOnly bool) string {
	t.Helper()

	root := filepath.Join(t.TempDir(), "demo")
	opts := NewProjectOptions{
		Name:        "demo",
		Module:      "example.com/demo",
		AppPath:     root,
		UIProvider:  newUIProviderFranken,
		APIMode:     apiOnly,
		I18nEnabled: false,
	}
	deps := NewDeps{
		ParseAgentPolicyBytes: func(b []byte) (policies.AgentPolicy, error) {
			return policies.AgentPolicy{}, nil
		},
		RenderAgentPolicyArtifacts: func(policy policies.AgentPolicy) (map[string][]byte, error) {
			return map[string][]byte{}, nil
		},
		AgentPolicyFilePath: policies.AgentPolicyFilePath,
	}
	if err := ScaffoldNewProject(opts, deps); err != nil {
		t.Fatalf("ScaffoldNewProject() error = %v", err)
	}
	return root
}

func verifyTestRunStep(name string, args ...string) (int, string, error) {
	return 0, strings.TrimSpace(strings.Join(append([]string{name}, args...), " ")), nil
}
