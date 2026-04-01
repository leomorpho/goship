package commands

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leomorpho/goship/tools/cli/ship/internal/generators"
	policies "github.com/leomorpho/goship/tools/cli/ship/internal/policies"
)

func TestStarterMakeResourceAndDestroyStayBuildable(t *testing.T) {
	root := t.TempDir()
	appPath := filepath.Join(root, "demo")

	opts := NewProjectOptions{
		Name:        "demo",
		Module:      "example.com/demo",
		AppPath:     appPath,
		UIProvider:  newUIProviderFranken,
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

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(appPath); err != nil {
		t.Fatalf("os.Chdir(%q) error = %v", appPath, err)
	}

	var out bytes.Buffer
	if code := generators.RunGenerateResource([]string{"contact", "--wire"}, &out, &out); code != 0 {
		t.Fatalf("RunGenerateResource() exit code = %d\n%s", code, out.String())
	}

	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = appPath
	buildOut, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build after make:resource failed: %v\n%s", err, buildOut)
	}

	out.Reset()
	if code := RunDestroy([]string{"resource:contact"}, DestroyDeps{Out: &out, Err: &out, Cwd: appPath}); code != 0 {
		t.Fatalf("RunDestroy() exit code = %d\n%s", code, out.String())
	}

	cmd = exec.Command("go", "build", "./...")
	cmd.Dir = appPath
	buildOut, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build after destroy failed: %v\n%s", err, buildOut)
	}
}

func TestStarterModuleAddFailsWithoutMutatingGoMod(t *testing.T) {
	root := t.TempDir()
	appPath := filepath.Join(root, "demo")

	opts := NewProjectOptions{
		Name:        "demo",
		Module:      "example.com/demo",
		AppPath:     appPath,
		UIProvider:  newUIProviderFranken,
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

	before, err := os.ReadFile(filepath.Join(appPath, "go.mod"))
	if err != nil {
		t.Fatalf("os.ReadFile(go.mod) error = %v", err)
	}

	err = applyModuleAdd(appPath, moduleCatalog["notifications"], false, &bytes.Buffer{})
	if err == nil {
		t.Fatal("applyModuleAdd() error = nil, want starter scaffold rejection")
	}
	if !strings.Contains(err.Error(), "starter scaffold") {
		t.Fatalf("applyModuleAdd() error = %v, want starter scaffold rejection", err)
	}

	after, err := os.ReadFile(filepath.Join(appPath, "go.mod"))
	if err != nil {
		t.Fatalf("os.ReadFile(go.mod) after error = %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf("go.mod mutated on failed module:add\nbefore:\n%s\nafter:\n%s", before, after)
	}
}
