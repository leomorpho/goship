package ship

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	agentPolicyFilePath = "tools/agent-policy/allowed-commands.yaml"
	agentGeneratedDir   = "tools/agent-policy/generated"
)

type agentPolicy struct {
	Version  int                  `yaml:"version"`
	Commands []agentPolicyCommand `yaml:"commands"`
}

type agentPolicyCommand struct {
	ID          string   `yaml:"id"`
	Description string   `yaml:"description"`
	Prefix      []string `yaml:"prefix"`
}

func (c CLI) runAgent(args []string) int {
	if len(args) == 0 {
		printAgentHelp(c.Out)
		return 0
	}

	switch args[0] {
	case "setup":
		return c.runAgentSetup(args[1:])
	case "check":
		return c.runAgentCheck(args[1:])
	case "help", "-h", "--help":
		printAgentHelp(c.Out)
		return 0
	default:
		fmt.Fprintf(c.Err, "unknown agent command: %s\n\n", args[0])
		printAgentHelp(c.Err)
		return 1
	}
}

func (c CLI) runAgentSetup(args []string) int {
	fs := flag.NewFlagSet("agent:setup", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	checkOnly := fs.Bool("check", false, "only check whether generated artifacts are in sync")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(c.Err, "invalid agent:setup arguments: %v\n", err)
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(c.Err, "unexpected agent:setup arguments: %v\n", fs.Args())
		return 1
	}

	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(c.Err, "failed to resolve working directory: %v\n", err)
		return 1
	}
	root, _, err := findGoModule(wd)
	if err != nil {
		fmt.Fprintf(c.Err, "failed to resolve project root (go.mod): %v\n", err)
		return 1
	}

	if *checkOnly {
		return runAgentPolicyCheck(c.Out, c.Err, root)
	}
	return runAgentPolicySetup(c.Out, c.Err, root)
}

func (c CLI) runAgentCheck(args []string) int {
	if len(args) > 0 {
		fmt.Fprintf(c.Err, "unexpected agent:check arguments: %v\n", args)
		return 1
	}

	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(c.Err, "failed to resolve working directory: %v\n", err)
		return 1
	}
	root, _, err := findGoModule(wd)
	if err != nil {
		fmt.Fprintf(c.Err, "failed to resolve project root (go.mod): %v\n", err)
		return 1
	}
	return runAgentPolicyCheck(c.Out, c.Err, root)
}

func runAgentPolicySetup(out, errOut io.Writer, root string) int {
	policy, err := loadAgentPolicy(filepath.Join(root, agentPolicyFilePath))
	if err != nil {
		fmt.Fprintf(errOut, "failed to load agent allowlist: %v\n", err)
		return 1
	}
	artifacts, err := renderAgentPolicyArtifacts(policy)
	if err != nil {
		fmt.Fprintf(errOut, "failed to render agent artifacts: %v\n", err)
		return 1
	}
	if err := writeAgentPolicyArtifacts(root, artifacts); err != nil {
		fmt.Fprintf(errOut, "failed to write agent artifacts: %v\n", err)
		return 1
	}

	fmt.Fprintf(out, "agent setup complete: synced %d artifact(s)\n", len(artifacts))
	for path := range artifacts {
		fmt.Fprintf(out, "- %s\n", path)
	}
	fmt.Fprintln(out, "next step: import the generated prefixes into your local Codex/Claude/Gemini permission config.")
	return 0
}

func runAgentPolicyCheck(out, errOut io.Writer, root string) int {
	policy, err := loadAgentPolicy(filepath.Join(root, agentPolicyFilePath))
	if err != nil {
		fmt.Fprintf(errOut, "failed to load agent allowlist: %v\n", err)
		return 1
	}
	artifacts, err := renderAgentPolicyArtifacts(policy)
	if err != nil {
		fmt.Fprintf(errOut, "failed to render agent artifacts: %v\n", err)
		return 1
	}
	missingOrDrifted, err := diffAgentPolicyArtifacts(root, artifacts)
	if err != nil {
		fmt.Fprintf(errOut, "failed to check agent artifacts: %v\n", err)
		return 1
	}
	if len(missingOrDrifted) > 0 {
		fmt.Fprintf(errOut, "agent policy artifacts out of sync (%d):\n", len(missingOrDrifted))
		for _, rel := range missingOrDrifted {
			fmt.Fprintf(errOut, "- %s\n", rel)
		}
		fmt.Fprintln(errOut, "run: ship agent:setup")
		return 1
	}
	fmt.Fprintln(out, "agent policy artifacts are in sync")
	return 0
}

func loadAgentPolicy(path string) (agentPolicy, error) {
	var policy agentPolicy
	b, err := os.ReadFile(path)
	if err != nil {
		return policy, err
	}
	return parseAgentPolicyBytes(b)
}

func parseAgentPolicyBytes(b []byte) (agentPolicy, error) {
	var policy agentPolicy
	if err := yaml.Unmarshal(b, &policy); err != nil {
		return policy, err
	}
	if policy.Version <= 0 {
		return policy, errors.New("policy version must be positive")
	}
	if len(policy.Commands) == 0 {
		return policy, errors.New("policy must include at least one command")
	}
	seen := make(map[string]struct{}, len(policy.Commands))
	for i, cmd := range policy.Commands {
		if strings.TrimSpace(cmd.ID) == "" {
			return policy, fmt.Errorf("commands[%d].id is required", i)
		}
		if _, ok := seen[cmd.ID]; ok {
			return policy, fmt.Errorf("duplicate command id: %s", cmd.ID)
		}
		seen[cmd.ID] = struct{}{}
		if len(cmd.Prefix) == 0 {
			return policy, fmt.Errorf("commands[%d].prefix cannot be empty", i)
		}
		for j, part := range cmd.Prefix {
			if strings.TrimSpace(part) == "" {
				return policy, fmt.Errorf("commands[%d].prefix[%d] cannot be empty", i, j)
			}
		}
	}
	return policy, nil
}

func renderAgentPolicyArtifacts(policy agentPolicy) (map[string][]byte, error) {
	prefixes := make([][]string, 0, len(policy.Commands))
	for _, cmd := range policy.Commands {
		prefix := append([]string(nil), cmd.Prefix...)
		prefixes = append(prefixes, prefix)
	}

	jsonPayload := struct {
		Version  int        `json:"version"`
		Prefixes [][]string `json:"prefixes"`
	}{
		Version:  policy.Version,
		Prefixes: prefixes,
	}
	jsonBytes, err := json.MarshalIndent(jsonPayload, "", "  ")
	if err != nil {
		return nil, err
	}
	jsonBytes = append(jsonBytes, '\n')

	prefixText := renderPrefixText(policy.Commands)
	installDoc := renderInstallDoc(policy.Commands)

	return map[string][]byte{
		filepath.ToSlash(filepath.Join(agentGeneratedDir, "allowed-prefixes.json")): []byte(jsonBytes),
		filepath.ToSlash(filepath.Join(agentGeneratedDir, "codex-prefixes.txt")):    []byte(prefixText),
		filepath.ToSlash(filepath.Join(agentGeneratedDir, "claude-prefixes.txt")):   []byte(prefixText),
		filepath.ToSlash(filepath.Join(agentGeneratedDir, "gemini-prefixes.txt")):   []byte(prefixText),
		filepath.ToSlash(filepath.Join(agentGeneratedDir, "INSTALL.md")):            []byte(installDoc),
	}, nil
}

func renderPrefixText(commands []agentPolicyCommand) string {
	var b strings.Builder
	for _, cmd := range commands {
		b.WriteString(strings.Join(cmd.Prefix, " "))
		b.WriteByte('\n')
	}
	return b.String()
}

func renderInstallDoc(commands []agentPolicyCommand) string {
	var b strings.Builder
	b.WriteString("# Agent Command Allowlist\n\n")
	b.WriteString("Source of truth: `tools/agent-policy/allowed-commands.yaml`\n\n")
	b.WriteString("Generated files in this directory are for local tool import.\n\n")
	b.WriteString("## Commands\n\n")
	for _, cmd := range commands {
		b.WriteString("- `")
		b.WriteString(strings.Join(cmd.Prefix, " "))
		b.WriteString("`")
		if strings.TrimSpace(cmd.Description) != "" {
			b.WriteString(" - ")
			b.WriteString(strings.TrimSpace(cmd.Description))
		}
		b.WriteByte('\n')
	}
	b.WriteString("\n## Setup\n\n")
	b.WriteString("1. Run `ship agent:setup` to sync generated artifacts.\n")
	b.WriteString("2. Import `codex-prefixes.txt`, `claude-prefixes.txt`, and `gemini-prefixes.txt` into each local tool's command-permission settings.\n")
	b.WriteString("3. Run `ship agent:check` in CI/pre-commit to enforce parity.\n")
	return b.String()
}

func writeAgentPolicyArtifacts(root string, artifacts map[string][]byte) error {
	paths := make([]string, 0, len(artifacts))
	for rel := range artifacts {
		paths = append(paths, rel)
	}
	sort.Strings(paths)

	for _, rel := range paths {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(abs, artifacts[rel], 0o644); err != nil {
			return err
		}
	}
	return nil
}

func diffAgentPolicyArtifacts(root string, expected map[string][]byte) ([]string, error) {
	drifted := make([]string, 0)
	paths := make([]string, 0, len(expected))
	for rel := range expected {
		paths = append(paths, rel)
	}
	sort.Strings(paths)

	for _, rel := range paths {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		actual, err := os.ReadFile(abs)
		if err != nil {
			drifted = append(drifted, rel)
			continue
		}
		if !bytes.Equal(actual, expected[rel]) {
			drifted = append(drifted, rel)
		}
	}
	return drifted, nil
}

func printAgentHelp(w io.Writer) {
	fmt.Fprintln(w, "ship agent commands:")
	fmt.Fprintln(w, "  ship agent:setup")
	fmt.Fprintln(w, "  ship agent:setup --check")
	fmt.Fprintln(w, "  ship agent:check")
	fmt.Fprintln(w, "  (syncs/checks generated allowlist artifacts for Codex, Claude, and Gemini)")
}
