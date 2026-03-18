package commands

import (
	"strings"
	"testing"
)

func TestPrintUpgradeHelp_ListsReadinessReportContract_RedSpec(t *testing.T) {
	out := captureHelp(t, PrintUpgradeHelp)

	for _, want := range []string{
		"upgrade readiness report",
		"blocker schema",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("upgrade help should mention %q\n%s", want, out)
		}
	}
}
