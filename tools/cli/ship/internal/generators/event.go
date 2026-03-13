package generators

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type GenerateEventDeps struct {
	Out      io.Writer
	Err      io.Writer
	HasFile  func(path string) bool
	TypesDir string
}

func RunGenerateEvent(args []string, d GenerateEventDeps) int {
	name, force, err := parseGenerateEventArgs(args)
	if err != nil {
		fmt.Fprintf(d.Err, "%v\n", err)
		return 1
	}

	target := filepath.Join(d.TypesDir, ModelFileName(name)+".go")
	if d.HasFile != nil && d.HasFile(target) && !force {
		fmt.Fprintf(d.Err, "refusing to overwrite existing event file %s (use --force)\n", target)
		return 1
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		fmt.Fprintf(d.Err, "failed to create events directory: %v\n", err)
		return 1
	}

	if err := os.WriteFile(target, []byte(renderEventTemplate(name)), 0o644); err != nil {
		fmt.Fprintf(d.Err, "failed to write event file: %v\n", err)
		return 1
	}

	fmt.Fprintf(d.Out, "Wrote event scaffold: %s\n", target)
	return 0
}

func parseGenerateEventArgs(args []string) (string, bool, error) {
	if len(args) == 0 {
		return "", false, fmt.Errorf("usage: ship make:event <TypeName> [--force]")
	}

	name := strings.TrimSpace(args[0])
	if !ModelNamePattern.MatchString(name) {
		return "", false, fmt.Errorf("invalid event name %q: use PascalCase (e.g. UserLoggedIn)", name)
	}

	force := false
	for _, arg := range args[1:] {
		if strings.TrimSpace(arg) == "--force" {
			force = true
			continue
		}
		return "", false, fmt.Errorf("usage: ship make:event <TypeName> [--force]")
	}

	return name, force, nil
}

func renderEventTemplate(name string) string {
	return "package types\n\nimport \"time\"\n\ntype " + name + " struct {\n\tAt time.Time\n}\n"
}
