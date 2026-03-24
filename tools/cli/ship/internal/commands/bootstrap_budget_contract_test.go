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
	if !strings.Contains(string(makefile), "test-fresh-app-ci:") {
		t.Fatal("Makefile should define a test-fresh-app-ci target")
	}
	if !strings.Contains(string(makefile), "bash tools/scripts/check-fresh-app-ci.sh") {
		t.Fatal("test-fresh-app-ci should run the canonical fresh-app CI script")
	}

	workflow, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "test.yml"))
	if err != nil {
		t.Fatalf("read workflow: %v", err)
	}
	for _, token := range []string{
		"\n  bootstrap_budget:\n",
		"run: make test-bootstrap-budget",
		"\n  fresh_app_ci:\n",
		"run: make test-fresh-app-ci",
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
		"ship db:migrate",
		"go run ./cmd/web",
		"curl --fail --silent http://127.0.0.1:",
		"BOOTSTRAP_BUDGET_SECONDS",
	} {
		if !strings.Contains(string(script), token) {
			t.Fatalf("bootstrap budget script missing %q", token)
		}
	}

	freshAppScript, err := os.ReadFile(filepath.Join(root, "tools", "scripts", "check-fresh-app-ci.sh"))
	if err != nil {
		t.Fatalf("read fresh-app CI script: %v", err)
	}
	for _, token := range []string{
		"go test ./tools/cli/ship/internal/commands -run TestFreshApp -count=1",
		"go test ./framework/web/controllers -count=1",
		"fresh-app CI lane passed",
	} {
		if !strings.Contains(string(freshAppScript), token) {
			t.Fatalf("fresh-app CI script missing %q", token)
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
		"`ship db:migrate`",
		"`/health/readiness`",
		"`make test-fresh-app-ci`",
	} {
		if !strings.Contains(string(devGuide), token) {
			t.Fatalf("development workflows guide missing %q", token)
		}
	}

	cliRef, err := os.ReadFile(filepath.Join(root, "docs", "reference", "01-cli.md"))
	if err != nil {
		t.Fatalf("read CLI reference: %v", err)
	}
	for _, token := range []string{
		"`cmd/web/main.go`",
		"`cmd/worker/main.go`",
		"`app/router.go`",
		"`db/migrate/migrations/`",
		"`static/`",
		"`styles/`",
		"`tools/agent-policy/generated/`",
	} {
		if !strings.Contains(string(cliRef), token) {
			t.Fatalf("CLI reference missing %q from the generated-app layout contract", token)
		}
	}
}
