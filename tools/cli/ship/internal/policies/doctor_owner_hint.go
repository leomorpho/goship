package policies

import "strings"

// IssueOwnerHint maps doctor issue codes to the owning generator/package hint.
func IssueOwnerHint(code string) string {
	switch strings.TrimSpace(code) {
	case "DX001", "DX002", "DX005", "DX011":
		return "ship new scaffold generator (tools/cli/ship/internal/commands/project_new.go)"
	default:
		return "doctor policy checks (tools/cli/ship/internal/policies/doctor.go)"
	}
}
