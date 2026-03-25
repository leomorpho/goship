package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/leomorpho/goship/config"
)

type ProfileDeps struct {
	Out io.Writer
	Err io.Writer
}

type profilePreset struct {
	name     string
	values   map[string]string
	keyOrder []string
}

var profilePresets = map[string]profilePreset{
	"single-binary": {
		name: "single-binary",
		values: map[string]string{
			"PAGODA_RUNTIME_PROFILE":     string(config.RuntimeProfileSingleNode),
			"PAGODA_PROCESSES_WEB":       "true",
			"PAGODA_PROCESSES_WORKER":    "true",
			"PAGODA_PROCESSES_SCHEDULER": "true",
			"PAGODA_PROCESSES_COLOCATED": "true",
		},
		keyOrder: []string{
			"PAGODA_RUNTIME_PROFILE",
			"PAGODA_PROCESSES_WEB",
			"PAGODA_PROCESSES_WORKER",
			"PAGODA_PROCESSES_SCHEDULER",
			"PAGODA_PROCESSES_COLOCATED",
		},
	},
	"standard": {
		name: "standard",
		values: map[string]string{
			"PAGODA_RUNTIME_PROFILE":     string(config.RuntimeProfileServerDB),
			"PAGODA_PROCESSES_WEB":       "true",
			"PAGODA_PROCESSES_WORKER":    "false",
			"PAGODA_PROCESSES_SCHEDULER": "false",
			"PAGODA_PROCESSES_COLOCATED": "false",
		},
		keyOrder: []string{
			"PAGODA_RUNTIME_PROFILE",
			"PAGODA_PROCESSES_WEB",
			"PAGODA_PROCESSES_WORKER",
			"PAGODA_PROCESSES_SCHEDULER",
			"PAGODA_PROCESSES_COLOCATED",
		},
	},
	"distributed": {
		name: "distributed",
		values: map[string]string{
			"PAGODA_RUNTIME_PROFILE":     string(config.RuntimeProfileDistributed),
			"PAGODA_PROCESSES_WEB":       "true",
			"PAGODA_PROCESSES_WORKER":    "true",
			"PAGODA_PROCESSES_SCHEDULER": "true",
			"PAGODA_PROCESSES_COLOCATED": "false",
		},
		keyOrder: []string{
			"PAGODA_RUNTIME_PROFILE",
			"PAGODA_PROCESSES_WEB",
			"PAGODA_PROCESSES_WORKER",
			"PAGODA_PROCESSES_SCHEDULER",
			"PAGODA_PROCESSES_COLOCATED",
		},
	},
}

var profileAliases = map[string]string{
	"single-node": "single-binary",
	"server-db":   "standard",
	"single":      "single-binary",
	"local":       "single-binary",
}

func RunProfile(args []string, d ProfileDeps) int {
	if len(args) == 0 {
		PrintProfileHelp(d.Out)
		return 0
	}

	switch args[0] {
	case "set":
		return runProfileSet(args[1:], d)
	case "help", "-h", "--help":
		PrintProfileHelp(d.Out)
		return 0
	default:
		fmt.Fprintf(d.Err, "unknown profile command: %s\n\n", args[0])
		PrintProfileHelp(d.Err)
		return 1
	}
}

func PrintProfileHelp(w io.Writer) {
	fmt.Fprintln(w, "ship profile commands:")
	fmt.Fprintln(w, "  ship profile:set <single-binary|standard|distributed>  Rewrite the local runtime profile and process preset")
}

func runProfileSet(args []string, d ProfileDeps) int {
	if len(args) != 1 {
		fmt.Fprintln(d.Err, "usage: ship profile:set <single-binary|standard|distributed>")
		return 1
	}

	presetName := normalizeProfilePreset(args[0])
	preset, ok := profilePresets[presetName]
	if !ok {
		fmt.Fprintf(d.Err, "unknown profile preset %q (expected single-binary|standard|distributed)\n", args[0])
		return 1
	}

	envPath, err := findEnvFile(".")
	if err != nil {
		fmt.Fprintf(d.Err, "profile:set requires a .env file: %v\n", err)
		fmt.Fprintln(d.Err, "Next step: create a .env file (for example from .env.example), then rerun `ship profile:set <preset>`.")
		return 1
	}

	original, err := os.ReadFile(envPath)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to read %s: %v\n", envPath, err)
		return 1
	}

	updated, changed, err := rewriteEnvAssignments(string(original), preset.values, preset.keyOrder)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to update %s: %v\n", envPath, err)
		return 1
	}
	if !changed {
		fmt.Fprintf(d.Out, "profile preset %q already applied in %s\n", preset.name, envPath)
		return 0
	}

	if err := os.WriteFile(envPath, []byte(updated), 0o644); err != nil {
		fmt.Fprintf(d.Err, "failed to write %s: %v\n", envPath, err)
		fmt.Fprintln(d.Err, "Next step: ensure .env is writable and rerun `ship profile:set <preset>`.")
		return 1
	}

	fmt.Fprintf(d.Out, "profile preset %q applied in %s\n", preset.name, envPath)
	for _, key := range preset.keyOrder {
		fmt.Fprintf(d.Out, "- %s=%s\n", key, preset.values[key])
	}
	return 0
}

func normalizeProfilePreset(raw string) string {
	key := strings.ToLower(strings.TrimSpace(raw))
	if alias, ok := profileAliases[key]; ok {
		return alias
	}
	return key
}

func findEnvFile(start string) (string, error) {
	dir := filepath.Clean(start)
	for {
		path := filepath.Join(dir, ".env")
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no .env found from %s", start)
		}
		dir = parent
	}
}

func rewriteEnvAssignments(content string, desired map[string]string, order []string) (string, bool, error) {
	lines := strings.Split(content, "\n")
	changed := false
	seen := map[string]bool{}

	for i, line := range lines {
		key, prefix, ok := splitEnvAssignment(line)
		if !ok {
			continue
		}
		value, found := desired[key]
		if !found {
			continue
		}
		seen[key] = true
		replacement := prefix + key + "=" + value
		if line != replacement {
			lines[i] = replacement
			changed = true
		}
	}

	for _, key := range order {
		if seen[key] {
			continue
		}
		lines = append(lines, key+"="+desired[key])
		changed = true
	}

	return strings.Join(lines, "\n"), changed, nil
}

func splitEnvAssignment(line string) (key, prefix string, ok bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", "", false
	}
	prefix = line[:len(line)-len(strings.TrimLeft(line, " \t"))]
	body := strings.TrimSpace(line)
	if strings.HasPrefix(body, "export ") {
		prefix += "export "
		body = strings.TrimSpace(strings.TrimPrefix(body, "export "))
	}
	key, _, ok = strings.Cut(body, "=")
	if !ok || strings.TrimSpace(key) == "" {
		return "", "", false
	}
	return strings.TrimSpace(key), prefix, true
}
