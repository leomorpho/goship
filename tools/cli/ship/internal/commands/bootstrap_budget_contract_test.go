package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBootstrapBudgetContract_RedSpec(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", "..", "..", ".."))

	makefile, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	if !strings.Contains(string(makefile), "test-bootstrap-budget:") {
		t.Fatal("Makefile should define a test-bootstrap-budget target")
	}
	if !strings.Contains(string(makefile), "bash tools/scripts/check-bootstrap-budget.sh") {
		t.Fatal("test-bootstrap-budget should run the canonical bootstrap budget script")
	}

	workflow, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "test.yml"))
	if err != nil {
		t.Fatalf("read workflow: %v", err)
	}
	for _, token := range []string{
		"\n  bootstrap_budget:\n",
		"run: make test-bootstrap-budget",
	} {
		if !strings.Contains(string(workflow), token) {
			t.Fatalf("test workflow missing %q", token)
		}
	}

	script, err := os.ReadFile(filepath.Join(root, "tools", "scripts", "check-bootstrap-budget.sh"))
	if err != nil {
		t.Fatalf("read bootstrap budget script: %v", err)
	}
	for _, token := range []string{
		"ship new",
		"go run ./cmd/web",
		"BOOTSTRAP_BUDGET_SECONDS",
	} {
		if !strings.Contains(string(script), token) {
			t.Fatalf("bootstrap budget script missing %q", token)
		}
	}

	devGuide, err := os.ReadFile(filepath.Join(root, "docs", "guides", "02-development-workflows.md"))
	if err != nil {
		t.Fatalf("read development workflows guide: %v", err)
	}
	for _, token := range []string{
		"`make test-bootstrap-budget`",
		"`BOOTSTRAP_BUDGET_SECONDS`",
		"`ship new <app> --no-i18n`",
	} {
		if !strings.Contains(string(devGuide), token) {
			t.Fatalf("development workflows guide missing %q", token)
		}
	}
}
