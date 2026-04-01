package commands

import (
	"fmt"
	"os"

	policies "github.com/leomorpho/goship/tools/cli/ship/v2/internal/policies"
)

func runI18nCI(args []string, d I18nDeps, root string) int {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			fmt.Fprintln(d.Out, "usage: ship i18n:ci")
			return 0
		}
	}
	if len(args) > 0 {
		fmt.Fprintln(d.Err, "usage: ship i18n:ci")
		return 1
	}

	findings, err := CollectI18nScanFindings(root, nil)
	if err != nil {
		fmt.Fprintf(d.Err, "i18n:ci failed scanner: %v\n", err)
		return 1
	}

	previousMode, hadMode := os.LookupEnv("PAGODA_I18N_STRICT_MODE")
	_ = os.Setenv("PAGODA_I18N_STRICT_MODE", "error")
	defer func() {
		if hadMode {
			_ = os.Setenv("PAGODA_I18N_STRICT_MODE", previousMode)
			return
		}
		_ = os.Unsetenv("PAGODA_I18N_STRICT_MODE")
	}()

	doctorIssues := policies.RunDoctorChecks(root)
	dx029 := make([]policies.DoctorIssue, 0)
	for _, issue := range doctorIssues {
		if issue.Code == "DX029" {
			dx029 = append(dx029, issue)
		}
	}

	if len(findings) == 0 && len(dx029) == 0 {
		fmt.Fprintln(d.Out, "i18n:ci passed.")
		return 0
	}

	fmt.Fprintln(d.Err, "i18n:ci failed.")
	fmt.Fprintf(d.Err, "  scanner findings: %d\n", len(findings))
	for _, finding := range findings {
		fmt.Fprintf(d.Err, "    - %s:%d:%d (%s)\n", finding.File, finding.Line, finding.Column, finding.ID)
	}
	fmt.Fprintf(d.Err, "  doctor DX029 findings: %d\n", len(dx029))
	for _, issue := range dx029 {
		fmt.Fprintf(d.Err, "    - %s\n", issue.Message)
	}
	return 1
}
