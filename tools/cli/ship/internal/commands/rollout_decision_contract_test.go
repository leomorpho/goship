package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRolloutDecisionContract_RedSpec(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", "..", "..", "..", ".."))

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
	for _, required := range []string{
		"staged-rollout-decision-v1",
		"schema_version",
		"runtime_contract_version",
		"policy_input_version",
		"canary",
		"verification",
		"ship runtime:report --json",
		"runtime facts",
		"control-plane policy",
	} {
		if !strings.Contains(managedText, required) {
			t.Fatalf("managed-mode contract doc should include rollout decision token %q", required)
		}
	}

	if !strings.Contains(string(cliRef), "staged-rollout-decision-v1") {
		t.Fatal("cli reference should describe the staged rollout decision contract dependency")
	}
	if !strings.Contains(string(roadmap), "staged-rollout-decision-v1") {
		t.Fatal("framework roadmap should track the staged rollout decision contract")
	}
	for _, required := range []string{
		"rollout engine",
		"traffic shaping",
		"staged-rollout-decision-v1",
	} {
		if !strings.Contains(string(risks), required) {
			t.Fatalf("known gaps doc should describe rollout decision contract risk token %q", required)
		}
	}
}
