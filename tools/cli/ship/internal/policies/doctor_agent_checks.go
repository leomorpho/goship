package policies

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
)

func checkAgentPolicyArtifacts(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	policyPath := filepath.Join(root, AgentPolicyFilePath)
	if !hasFile(policyPath) {
		return append(issues, DoctorIssue{
			Code:    "DX017",
			Message: fmt.Sprintf("missing agent policy file: %s", filepath.ToSlash(AgentPolicyFilePath)),
			Fix:     "add tools/agent-policy/allowed-commands.yaml and run ship agent:setup",
		})
	}
	policy, err := LoadPolicy(policyPath)
	if err != nil {
		return append(issues, DoctorIssue{
			Code:    "DX017",
			Message: "invalid agent policy file",
			Fix:     err.Error(),
		})
	}
	expected, err := RenderPolicyArtifacts(policy)
	if err != nil {
		return append(issues, DoctorIssue{
			Code:    "DX017",
			Message: "failed to render agent policy artifacts",
			Fix:     err.Error(),
		})
	}
	drifted, err := DiffPolicyArtifacts(root, expected)
	if err != nil {
		return append(issues, DoctorIssue{
			Code:    "DX017",
			Message: "failed to compare generated agent artifacts",
			Fix:     err.Error(),
		})
	}
	for _, rel := range drifted {
		issues = append(issues, DoctorIssue{
			Code:    "DX017",
			Message: fmt.Sprintf("agent artifact out of sync: %s", rel),
			Fix:     "run ship agent:setup",
		})
	}
	return issues
}

func defaultDoctorRunCmd(dir string, name string, args ...string) (int, string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), string(out), nil
		}
		return 1, string(out), err
	}
	return 0, string(out), nil
}

func printDoctorHelp(w io.Writer) {
	fmt.Fprintln(w, "ship doctor commands:")
	fmt.Fprintln(w, "  ship doctor [--json]")
	fmt.Fprintln(w, "  (validates canonical app structure and LLM/DX conventions)")
}
