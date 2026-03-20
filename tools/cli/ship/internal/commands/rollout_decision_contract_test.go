package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRolloutDecisionContract_RedSpec(t *testing.T) {
	root := repoRootFromCommandsTest(t)

	managedDoc, err := os.ReadFile(filepath.Join(root, "docs", "architecture", "09-standalone-and-managed-mode.md"))
	if err != nil {
		t.Fatal(err)
	}
	cliRef, err := os.ReadFile(filepath.Join(root, "docs", "reference", "01-cli.md"))
	if err != nil {
		t.Fatal(err)
	}
	roadmap, err := os.ReadFile(filepath.Join(root, "docs", "roadmap", "01-framework-plan.md"))
	if err != nil {
		t.Fatal(err)
	}
	risks, err := os.ReadFile(filepath.Join(root, "docs", "architecture", "06-known-gaps-and-risks.md"))
	if err != nil {
		t.Fatal(err)
	}

	managedText := string(managedDoc)
	assertContainsAll(t, managedText, []string{
		"staged-rollout-decision-v1",
		"schema_version",
		"runtime_contract_version",
		"policy_input_version",
		"canary",
		"verification",
		"ship runtime:report --json",
		"runtime facts",
		"control-plane policy",
		"`canary` must be omitted when `decision` is `hold`, `promote`, or `rollback`",
		"`canary` must include `cohort`, `percentage`, and `exit_criteria` when `decision=canary`",
		"`verification` must include `checks`, `evidence_refs`, and `verified_by`",
		"`hold` and `rollback` must keep at least one machine-readable blocker reason",
		"`promote` requires `blockers` to be empty",
	})

	assertContainsAll(t, string(cliRef), []string{
		"staged-rollout-decision-v1",
		"`verification` evidence (`checks`, `evidence_refs`, `verified_by`)",
		"`decision=canary` requires a populated `canary` object",
	})
	assertContainsAll(t, string(roadmap), []string{
		"staged-rollout-decision-v1",
		"`decision=canary` requires explicit cohort/percentage/exit criteria",
		"`hold`/`rollback` preserve blockers while `promote` requires blockers to be empty",
	})
	assertContainsAll(t, string(risks), []string{
		"rollout engine",
		"traffic shaping",
		"staged-rollout-decision-v1",
		"decision-conditioned invariants",
	})
}

func assertContainsAll(t *testing.T, text string, required []string) {
	t.Helper()

	for _, token := range required {
		if !strings.Contains(text, token) {
			t.Fatalf("expected text to include %q", token)
		}
	}
}
