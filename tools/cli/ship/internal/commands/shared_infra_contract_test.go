package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSharedInfraAdoptionContract_RedSpec(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", "..", "..", "..", ".."))

	describeSource, err := os.ReadFile(filepath.Join(root, "tools", "cli", "ship", "internal", "commands", "describe.go"))
	if err != nil {
		t.Fatal(err)
	}
	describeTests, err := os.ReadFile(filepath.Join(root, "tools", "cli", "ship", "internal", "commands", "describe_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	scopeDoc, err := os.ReadFile(filepath.Join(root, "docs", "architecture", "03-project-scope-analysis.md"))
	if err != nil {
		t.Fatal(err)
	}
	cliRef, err := os.ReadFile(filepath.Join(root, "docs", "reference", "01-cli.md"))
	if err != nil {
		t.Fatal(err)
	}

	for _, required := range []string{
		`"shared_infra"`,
		"SharedInfra",
		"shared_modules",
		"custom_app_jobs",
	} {
		if !strings.Contains(string(describeSource), required) {
			t.Fatalf("describe output should expose shared-infra adoption token %q", required)
		}
	}

	if !strings.Contains(string(describeTests), "SharedInfra") {
		t.Fatal("describe tests should lock the shared-infra adoption summary")
	}
	if !strings.Contains(string(scopeDoc), "shared-infra adoption") {
		t.Fatal("project scope analysis should describe the shared-infra adoption reporting surface")
	}
	if !strings.Contains(string(cliRef), "shared-infra adoption summary") {
		t.Fatal("cli reference should document the shared-infra adoption summary in ship describe")
	}
}
