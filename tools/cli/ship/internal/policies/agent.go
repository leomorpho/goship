package policies

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	AgentPolicyFilePath = "tools/agent-policy/allowed-commands.yaml"
	AgentGeneratedDir   = "tools/agent-policy/generated"
)

type AgentPolicy struct {
	Version  int                  `yaml:"version"`
	Commands []AgentPolicyCommand `yaml:"commands"`
}

type AgentPolicyCommand struct {
	ID          string   `yaml:"id"`
	Description string   `yaml:"description"`
	Prefix      []string `yaml:"prefix"`
}

func RunPolicySetup(out, errOut io.Writer, root string) int {
	policy, err := LoadPolicy(filepath.Join(root, AgentPolicyFilePath))
	if err != nil {
		fmt.Fprintf(errOut, "failed to load agent allowlist: %v\n", err)
		return 1
	}
	artifacts, err := RenderPolicyArtifacts(policy)
	if err != nil {
		fmt.Fprintf(errOut, "failed to render agent artifacts: %v\n", err)
		return 1
	}
	if err := WritePolicyArtifacts(root, artifacts); err != nil {
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

func RunPolicyCheck(out, errOut io.Writer, root string) int {
	policy, err := LoadPolicy(filepath.Join(root, AgentPolicyFilePath))
	if err != nil {
		fmt.Fprintf(errOut, "failed to load agent allowlist: %v\n", err)
		return 1
	}
	artifacts, err := RenderPolicyArtifacts(policy)
	if err != nil {
		fmt.Fprintf(errOut, "failed to render agent artifacts: %v\n", err)
		return 1
	}
	missingOrDrifted, err := DiffPolicyArtifacts(root, artifacts)
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

func LoadPolicy(path string) (AgentPolicy, error) {
	var policy AgentPolicy
	b, err := os.ReadFile(path)
	if err != nil {
		return policy, err
	}
	return ParsePolicyBytes(b)
}

func ParsePolicyBytes(b []byte) (AgentPolicy, error) {
	var policy AgentPolicy
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

func RenderPolicyArtifacts(policy AgentPolicy) (map[string][]byte, error) {
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
		filepath.ToSlash(filepath.Join(AgentGeneratedDir, "allowed-prefixes.json")): jsonBytes,
		filepath.ToSlash(filepath.Join(AgentGeneratedDir, "codex-prefixes.txt")):    []byte(prefixText),
		filepath.ToSlash(filepath.Join(AgentGeneratedDir, "claude-prefixes.txt")):   []byte(prefixText),
		filepath.ToSlash(filepath.Join(AgentGeneratedDir, "gemini-prefixes.txt")):   []byte(prefixText),
		filepath.ToSlash(filepath.Join(AgentGeneratedDir, "INSTALL.md")):            []byte(installDoc),
	}, nil
}

func WritePolicyArtifacts(root string, artifacts map[string][]byte) error {
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

func DiffPolicyArtifacts(root string, expected map[string][]byte) ([]string, error) {
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

func renderPrefixText(commands []AgentPolicyCommand) string {
	var b strings.Builder
	for _, cmd := range commands {
		b.WriteString(strings.Join(cmd.Prefix, " "))
		b.WriteByte('\n')
	}
	return b.String()
}

func renderInstallDoc(commands []AgentPolicyCommand) string {
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
