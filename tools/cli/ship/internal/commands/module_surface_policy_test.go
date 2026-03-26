package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseModuleSurfaceCatalog(t *testing.T) {
	t.Parallel()

	content := []byte(`
version: module-surface-v1
candidates:
  - id: notifications
    class: battery
    decision: keep
  - id: profile
    class: starter-app
    decision: eject
`)

	decisions, err := parseModuleSurfaceCatalog(content)
	if err != nil {
		t.Fatalf("parseModuleSurfaceCatalog error: %v", err)
	}
	if got := decisions["notifications"]; got.Class != "battery" || got.Decision != "keep" {
		t.Fatalf("notifications decision=%+v, want class=battery decision=keep", got)
	}
	if got := decisions["profile"]; got.Class != "starter-app" || got.Decision != "eject" {
		t.Fatalf("profile decision=%+v, want class=starter-app decision=eject", got)
	}
}

func TestParseModuleSurfaceDecisions(t *testing.T) {
	t.Parallel()

	content := `# Module Surface Reset

## Decision Matrix

| Candidate | Class | Decision | Notes |
| --- | --- | --- | --- |
| ` + "`notifications`" + ` | ` + "`battery`" + ` | ` + "`keep`" + ` | canonical |
| ` + "`profile`" + ` | ` + "`starter-app`" + ` | ` + "`eject`" + ` | move out |
`

	decisions, err := parseModuleSurfaceDecisions(content)
	if err != nil {
		t.Fatalf("parseModuleSurfaceDecisions error: %v", err)
	}
	if got := decisions["notifications"]; got.Class != "battery" || got.Decision != "keep" {
		t.Fatalf("notifications decision=%+v, want class=battery decision=keep", got)
	}
	if got := decisions["profile"]; got.Class != "starter-app" || got.Decision != "eject" {
		t.Fatalf("profile decision=%+v, want class=starter-app decision=eject", got)
	}
}

func TestParseModuleSurfaceDecisions_RejectsDuplicateCandidates(t *testing.T) {
	t.Parallel()

	content := `| ` + "`notifications`" + ` | ` + "`battery`" + ` | ` + "`keep`" + ` | one |
| ` + "`notifications`" + ` | ` + "`battery`" + ` | ` + "`rewrite`" + ` | two |
`
	_, err := parseModuleSurfaceDecisions(content)
	if err == nil {
		t.Fatal("expected duplicate candidate error")
	}
	if !strings.Contains(err.Error(), "duplicate decision entry") {
		t.Fatalf("error=%q, want duplicate decision entry diagnostic", err.Error())
	}
}

func TestCheckModuleSurfaceResetPolicy_RejectsMissingModuleDecision(t *testing.T) {
	root := t.TempDir()

	mustMkdirAllSurface(t, filepath.Join(root, "tools", "cli", "ship", "internal", "commands"))
	mustWriteFileSurface(t, filepath.Join(root, "tools", "cli", "ship", "internal", "commands", "module.go"), "package commands\n")

	for _, moduleID := range []string{"emailsubscriptions", "jobs", "notifications", "paidsubscriptions", "storage", "extra"} {
		mustMkdirAllSurface(t, filepath.Join(root, "modules", moduleID))
	}
	for _, moduleID := range []string{"emailsubscriptions", "jobs", "notifications", "paidsubscriptions", "storage"} {
		mustWriteFileSurface(t, filepath.Join(root, "modules", moduleID, "go.mod"), "module github.com/leomorpho/goship-modules/"+moduleID+"\n\ngo 1.25\n")
	}

	mustMkdirAllSurface(t, filepath.Join(root, "docs", "architecture"))
	mustWriteFileSurface(t, filepath.Join(root, moduleSurfaceResetDocRelPath), syntheticModuleSurfaceDoc(
		[]string{"emailsubscriptions", "jobs", "notifications", "paidsubscriptions", "storage"},
	))
	mustMkdirAllSurface(t, filepath.Join(root, "config"))
	mustWriteFileSurface(t, filepath.Join(root, moduleSurfaceCatalogRelPath), syntheticModuleSurfaceCatalog(
		[]string{"emailsubscriptions", "jobs", "notifications", "paidsubscriptions", "storage"},
	))

	err := checkModuleSurfaceResetPolicy(root)
	if err == nil {
		t.Fatal("expected missing module decision error")
	}
	if !strings.Contains(err.Error(), "missing decision row for first-party module candidate \"extra\"") {
		t.Fatalf("error=%q, want missing candidate decision diagnostic", err.Error())
	}
}

func syntheticModuleSurfaceDoc(keepBatteries []string) string {
	rows := make([]string, 0, len(keepBatteries))
	for _, id := range keepBatteries {
		rows = append(rows, "| `"+id+"` | `battery` | `keep` | synthetic |")
	}
	return strings.Join([]string{
		"# Module Surface Reset",
		"",
		"Canonical machine-readable source: `config/module-surface.yaml`.",
		"",
		"## Canonical Battery Contract",
		"- single entrypoint",
		"",
		"## Decision Matrix",
		"",
		"| Candidate | Class | Decision | Notes |",
		"| --- | --- | --- | --- |",
		strings.Join(rows, "\n"),
		"",
		"## Notifications Replacement Plan",
		"- notifications-inbox",
		"- notifications-push",
		"- notifications-email",
		"- notifications-sms",
		"- notifications-schedule",
	}, "\n")
}

func syntheticModuleSurfaceCatalog(keepBatteries []string) string {
	rows := make([]string, 0, len(keepBatteries))
	for _, id := range keepBatteries {
		rows = append(rows, "  - id: "+id+"\n    class: battery\n    decision: keep")
	}
	return strings.Join([]string{
		"version: module-surface-v1",
		"candidates:",
		strings.Join(rows, "\n"),
	}, "\n")
}

func mustMkdirAllSurface(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func mustWriteFileSurface(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
