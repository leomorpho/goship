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
}
