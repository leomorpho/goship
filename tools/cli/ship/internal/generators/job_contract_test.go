package generators

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMakeJobContract_RedSpec(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", "..", "..", "..", ".."))

	makeHelp, err := os.ReadFile(filepath.Join(root, "tools", "cli", "ship", "internal", "commands", "help.go"))
	if err != nil {
		t.Fatal(err)
	}
	cliDispatch, err := os.ReadFile(filepath.Join(root, "tools", "cli", "ship", "internal", "cli", "cli.go"))
	if err != nil {
		t.Fatal(err)
	}
	cliRef, err := os.ReadFile(filepath.Join(root, "docs", "reference", "01-cli.md"))
	if err != nil {
		t.Fatal(err)
	}
	jobsGuide, err := os.ReadFile(filepath.Join(root, "docs", "guides", "05-jobs-module.md"))
	if err != nil {
		t.Fatal(err)
	}

	for _, required := range []string{
		"ship make:job <Name>",
		"Generate a background job scaffold",
	} {
		if !strings.Contains(string(makeHelp), required) {
			t.Fatalf("make help should advertise %q for the make:job generator", required)
		}
		if !strings.Contains(string(cliRef), required) {
			t.Fatalf("cli reference should advertise %q for the make:job generator", required)
		}
	}

	for _, required := range []string{
		`case "job":`,
		"runMakeJob(",
	} {
		if !strings.Contains(string(cliDispatch), required) {
			t.Fatalf("cli dispatch should include %q for ship make:job", required)
		}
	}

	for _, required := range []string{
		"ship make:job <Name>",
		"core.Jobs",
		"core.JobHandler",
	} {
		if !strings.Contains(string(jobsGuide), required) {
			t.Fatalf("jobs guide should describe %q as part of the make:job scaffold contract", required)
		}
	}
}
