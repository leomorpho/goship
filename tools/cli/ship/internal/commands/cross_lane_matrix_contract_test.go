package commands

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCrossLaneDependencyMatrixContract_RedSpec(t *testing.T) {
	root := repoRootFromCommandsTest(t)
	roadmap := mustReadText(t, filepath.Join(root, "docs", "roadmap", "01-framework-plan.md"))
	gaps := mustReadText(t, filepath.Join(root, "docs", "architecture", "06-known-gaps-and-risks.md"))

	for _, text := range []string{roadmap, gaps} {
		if !strings.Contains(text, "cross-lane dependency matrix") {
			t.Fatal("docs should name the cross-lane dependency matrix contract")
		}
		if !strings.Contains(text, "must-finish-before contract map") {
			t.Fatal("docs should name the must-finish-before contract map")
		}
	}
	if !strings.Contains(roadmap, "| Lane | Requires | Must finish before | Why |") {
		t.Fatal("roadmap should include the explicit cross-lane dependency matrix table")
	}
	for _, token := range []string{
		"runtime-contract-v1",
		"upgrade-readiness-v1",
		"promotion-state-machine-v1",
		"backup-manifest-v1",
		"restore_evidence.record_links",
		"staged-rollout-decision-v1",
	} {
		if !strings.Contains(roadmap, token) {
			t.Fatalf("roadmap matrix should include %q", token)
		}
	}
}
