package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTaskOrientedOSSDocsContract_RedSpec(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", "..", "..", "..", ".."))

	indexContent, err := os.ReadFile(filepath.Join(root, "docs", "00-index.md"))
	if err != nil {
		t.Fatal(err)
	}
	playbookContent, err := os.ReadFile(filepath.Join(root, "docs", "guides", "03-how-to-playbook.md"))
	if err != nil {
		t.Fatal(err)
	}
	cliRefContent, err := os.ReadFile(filepath.Join(root, "docs", "reference", "01-cli.md"))
	if err != nil {
		t.Fatal(err)
	}

	requiredGuides := []string{
		"guides/11-add-endpoint-workflow.md",
		"guides/12-add-module-workflow.md",
		"guides/13-add-background-job-workflow.md",
	}
	for _, guide := range requiredGuides {
		if !strings.Contains(string(indexContent), guide) {
			t.Fatalf("docs index should reference canonical OSS workflow guide %q", guide)
		}
		if !strings.Contains(string(playbookContent), guide) {
			t.Fatalf("how-to playbook should reference canonical OSS workflow guide %q", guide)
		}
	}

	for _, required := range []string{
		"ship make:module <Name>",
		"ship make:job <Name>",
		"ship make:resource <name>",
	} {
		if !strings.Contains(string(cliRefContent), required) {
			t.Fatalf("cli reference should document workflow command %q", required)
		}
	}
}
