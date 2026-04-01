package commands

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	policies "github.com/leomorpho/goship/tools/cli/ship/v2/internal/policies"
)

type AgentDeps struct {
	Out               io.Writer
	Err               io.Writer
	FindGoModule      func(start string) (string, string, error)
	RunGitWorktreeAdd func(root, worktreePath, branch string) error
	DescribeJSON      func(root string) (string, error)
	RunVerify         func(worktreePath string) error
	RunGit            func(dir string, args ...string) error
	RunGh             func(root string, args ...string) error
}

func RunAgent(args []string, d AgentDeps) int {
	if len(args) == 0 {
		printAgentHelp(d.Out)
		return 0
	}

	switch args[0] {
	case "setup":
		return runAgentSetup(args[1:], d)
	case "check":
		return runAgentCheck(args[1:], d)
	case "start":
		return runAgentStart(args[1:], d)
	case "finish":
		return runAgentFinish(args[1:], d)
	case "status":
		return runAgentStatus(args[1:], d)
	case "help", "-h", "--help":
		printAgentHelp(d.Out)
		return 0
	default:
		fmt.Fprintf(d.Err, "unknown agent command: %s\n\n", args[0])
		printAgentHelp(d.Err)
		return 1
	}
}

func runAgentSetup(args []string, d AgentDeps) int {
	fs := flag.NewFlagSet("agent:setup", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	checkOnly := fs.Bool("check", false, "only check whether generated artifacts are in sync")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(d.Err, "invalid agent:setup arguments: %v\n", err)
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(d.Err, "unexpected agent:setup arguments: %v\n", fs.Args())
		return 1
	}

	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve working directory: %v\n", err)
		return 1
	}
	root, _, err := d.FindGoModule(wd)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve project root (go.mod): %v\n", err)
		return 1
	}

	if *checkOnly {
		return policies.RunPolicyCheck(d.Out, d.Err, root)
	}
	return policies.RunPolicySetup(d.Out, d.Err, root)
}

func runAgentCheck(args []string, d AgentDeps) int {
	if len(args) > 0 {
		fmt.Fprintf(d.Err, "unexpected agent:check arguments: %v\n", args)
		return 1
	}

	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve working directory: %v\n", err)
		return 1
	}
	root, _, err := d.FindGoModule(wd)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve project root (go.mod): %v\n", err)
		return 1
	}
	return policies.RunPolicyCheck(d.Out, d.Err, root)
}

func runAgentStatus(args []string, d AgentDeps) int {
	fs := flag.NewFlagSet("agent:status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	toolFile := fs.String("tool-file", "", "path to local agent tool command permission file")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(d.Err, "invalid agent:status arguments: %v\n", err)
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(d.Err, "unexpected agent:status arguments: %v\n", fs.Args())
		return 1
	}

	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve working directory: %v\n", err)
		return 1
	}
	root, _, err := d.FindGoModule(wd)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve project root (go.mod): %v\n", err)
		return 1
	}
	policy, err := policies.LoadPolicy(filepath.Join(root, policies.AgentPolicyFilePath))
	if err != nil {
		fmt.Fprintf(d.Err, "failed to load agent allowlist: %v\n", err)
		return 1
	}
	if code := policies.RunPolicyCheck(io.Discard, io.Discard, root); code != 0 {
		fmt.Fprintln(d.Err, "agent policy artifacts are out of sync; run: ship agent:setup")
		return 1
	}

	statuses := []toolStatus{
		{
			Name:          "agent-tool",
			PolicyPath:    filepath.ToSlash(filepath.Join(policies.AgentGeneratedDir, "agent-prefixes.txt")),
			InstalledPath: resolveToolConfigPath(*toolFile, "SHIP_AGENT_TOOL_FILE", defaultAgentToolPaths()),
		},
	}

	policyPrefixes := make([]string, 0, len(policy.Commands))
	for _, cmd := range policy.Commands {
		policyPrefixes = append(policyPrefixes, strings.Join(cmd.Prefix, " "))
	}

	fmt.Fprintf(d.Out, "agent status (policy version=%d, commands=%d)\n", policy.Version, len(policy.Commands))
	for i := range statuses {
		st := &statuses[i]
		report := inspectToolInstall(*st, policyPrefixes)
		fmt.Fprintf(d.Out, "- %s: %s\n", st.Name, report.State)
		if report.Path != "" {
			fmt.Fprintf(d.Out, "  path: %s\n", report.Path)
		}
		if report.Matched >= 0 {
			fmt.Fprintf(d.Out, "  matched: %d/%d\n", report.Matched, len(policyPrefixes))
		}
		if report.Note != "" {
			fmt.Fprintf(d.Out, "  note: %s\n", report.Note)
		}
	}
	return 0
}

func printAgentHelp(w io.Writer) {
	fmt.Fprintln(w, "ship agent commands:")
	fmt.Fprintln(w, "  ship agent:setup                                                          Generate local agent allowlist artifacts from policy")
	fmt.Fprintln(w, "  ship agent:setup --check                                                  Validate generated allowlist artifacts are in sync")
	fmt.Fprintln(w, "  ship agent:start --task \"Add feature\" [--id ID]                           Create a scoped git worktree for an agent task")
	fmt.Fprintln(w, "  ship agent:finish --id TASK --message \"feat(...)\" [--pr]                  Verify, commit, optionally open PR, and clean up worktree")
	fmt.Fprintln(w, "  ship agent:check                                                          Run policy artifact drift checks")
	fmt.Fprintln(w, "  ship agent:status [--tool-file <path>]                                    Inspect local agent-tool policy sync state")
}

type toolStatus struct {
	Name          string
	PolicyPath    string
	InstalledPath string
}

type toolInstallReport struct {
	State   string
	Path    string
	Matched int
	Note    string
}

func inspectToolInstall(tool toolStatus, policyPrefixes []string) toolInstallReport {
	if strings.TrimSpace(tool.InstalledPath) == "" {
		return toolInstallReport{State: "not-detected", Matched: -1, Note: "no local config path detected (set flag or SHIP_AGENT_*_FILE env var)"}
	}
	b, err := os.ReadFile(tool.InstalledPath)
	if err != nil {
		return toolInstallReport{State: "not-installed", Path: tool.InstalledPath, Matched: -1, Note: "config path not readable"}
	}
	content := string(b)
	matched := 0
	for _, prefix := range policyPrefixes {
		if strings.Contains(content, prefix) {
			matched++
		}
	}
	state := "drifted"
	if matched == len(policyPrefixes) {
		state = "in-sync"
	} else if matched == 0 {
		state = "not-installed"
	}
	return toolInstallReport{State: state, Path: tool.InstalledPath, Matched: matched, Note: "best-effort substring match against local tool config"}
}

func resolveToolConfigPath(flagValue, envKey string, defaults []string) string {
	if v := strings.TrimSpace(flagValue); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		return v
	}
	for _, p := range defaults {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func defaultAgentToolPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{
		filepath.Join(home, ".config", "agent", "permissions.json"),
		filepath.Join(home, ".agent", "permissions.json"),
		filepath.Join(home, ".codex", "permissions.json"),
		filepath.Join(home, ".claude", "permissions.json"),
		filepath.Join(home, ".gemini", "permissions.json"),
	}
}
