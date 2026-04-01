package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	policies "github.com/leomorpho/goship/tools/cli/ship/internal/policies"
	startertemplate "github.com/leomorpho/goship/tools/cli/ship/internal/templates/starter"
)

func TestValidateStarterScaffoldLayoutEmbeddedTemplate(t *testing.T) {
	t.Parallel()

	if err := validateStarterScaffoldLayout(startertemplate.Files, starterTemplateRoot); err != nil {
		t.Fatalf("validateStarterScaffoldLayout() error = %v", err)
	}
}

func TestScaffoldNewProjectProducesBuildableStarter(t *testing.T) {
	t.Parallel()

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

	if _, err := os.Stat(filepath.Join(appPath, "tmp")); err != nil {
		t.Fatalf("starter tmp dir missing: %v", err)
	}

	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = appPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build ./... failed: %v\n%s", err, out)
	}
}
