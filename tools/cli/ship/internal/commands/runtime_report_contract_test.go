package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrintRootHelp_ListsRuntimeReport_RedSpec(t *testing.T) {
	out := captureHelp(t, PrintRootHelp)
	line := findLineByPrefix(out, "  ship runtime:report --json")
	if line == "" {
		t.Fatalf("root help missing runtime report line\n%s", out)
	}
	if !containsRuntimeReportTokens(line, "machine-readable", "runtime", "capability") {
		t.Fatalf("runtime report help line should describe machine-readable runtime capability output: %q", line)
	}
}

func TestRunRuntimeReport_JSONContract_RedSpec(t *testing.T) {
	root := repoRootForRuntimeReportTest(t)
	cliSource := mustReadRuntimeReportText(t, filepath.Join(root, "tools", "cli", "ship", "internal", "cli", "cli.go"))
	helpSource := mustReadRuntimeReportText(t, filepath.Join(root, "tools", "cli", "ship", "internal", "commands", "help.go"))

	if !strings.Contains(cliSource, `case "runtime":`) && !strings.Contains(cliSource, `case "runtime:report":`) {
		t.Fatal("cli dispatcher does not yet expose a runtime report command path")
	}
	if !strings.Contains(helpSource, "ship runtime:report --json") {
		t.Fatal("help output does not yet advertise ship runtime:report --json")
	}
}

func containsRuntimeReportTokens(text string, want ...string) bool {
	for _, token := range want {
		if !strings.Contains(text, token) {
			return false
		}
	}
	return true
}

func repoRootForRuntimeReportTest(t *testing.T) string {
	t.Helper()
	return repoRootFromCommandsTest(t)
}

func mustReadRuntimeReportText(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}
