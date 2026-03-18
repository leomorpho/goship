package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPagodaIntakeGovernanceContract_RedSpec(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", "..", "..", "..", ".."))

	indexContent, err := os.ReadFile(filepath.Join(root, "docs", "00-index.md"))
	if err != nil {
		t.Fatal(err)
	}
	roadmapContent, err := os.ReadFile(filepath.Join(root, "docs", "roadmap", "01-framework-plan.md"))
	if err != nil {
		t.Fatal(err)
	}
	intakeLogContent, err := os.ReadFile(filepath.Join(root, "docs", "roadmap", "09-pagoda-intake-log.md"))
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(indexContent), "roadmap/09-pagoda-intake-log.md") {
		t.Fatal("docs index should reference the Pagoda intake log")
	}
	for _, required := range []string{
		"weekly or per-tag",
		"adopt",
		"adapt",
		"skip",
	} {
		if !strings.Contains(string(roadmapContent), required) {
			t.Fatalf("framework roadmap should describe Pagoda intake governance token %q", required)
		}
		if !strings.Contains(string(intakeLogContent), required) {
			t.Fatalf("Pagoda intake log should include governance token %q", required)
		}
	}
	if !strings.Contains(string(intakeLogContent), "| Upstream Ref | Area | Decision | Follow-Up |") {
		t.Fatal("Pagoda intake log should define the canonical adopt/adapt/skip table")
	}
}
