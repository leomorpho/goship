package generators

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type MakeLocaleOptions struct {
	Code string
}

type LocaleDeps struct {
	Out io.Writer
	Err io.Writer
	Cwd string
}

func RunMakeLocale(args []string, d LocaleDeps) int {
	if len(args) > 0 {
		switch args[0] {
		case "help", "-h", "--help":
			fmt.Fprintln(d.Out, "usage: ship make:locale <code>")
			return 0
		}
	}

	opts, err := ParseMakeLocaleArgs(args)
	if err != nil {
		fmt.Fprintf(d.Err, "invalid make:locale arguments: %v\n", err)
		return 1
	}

	cwd := strings.TrimSpace(d.Cwd)
	if cwd == "" {
		cwd, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(d.Err, "failed to resolve working directory: %v\n", err)
			return 1
		}
	}

	localesDir := filepath.Join(cwd, "locales")
	sourcePath := filepath.Join(localesDir, "en.yaml")
	targetPath := filepath.Join(localesDir, opts.Code+".yaml")

	if _, err := os.Stat(sourcePath); err != nil {
		fmt.Fprintf(d.Err, "missing source locale file: %s\n", sourcePath)
		return 1
	}
	if _, err := os.Stat(targetPath); err == nil {
		fmt.Fprintf(d.Err, "refusing to overwrite existing locale file: %s\n", targetPath)
		return 1
	}

	sourceBytes, err := os.ReadFile(sourcePath)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to read source locale file: %v\n", err)
		return 1
	}

	var source map[string]any
	if err := yaml.Unmarshal(sourceBytes, &source); err != nil {
		fmt.Fprintf(d.Err, "failed to parse source locale file: %v\n", err)
		return 1
	}

	empty := cloneLocaleTreeWithEmptyValues(source)
	rendered, err := yaml.Marshal(empty)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to render locale file: %v\n", err)
		return 1
	}

	if err := os.MkdirAll(localesDir, 0o755); err != nil {
		fmt.Fprintf(d.Err, "failed to create locales directory: %v\n", err)
		return 1
	}
	if err := os.WriteFile(targetPath, rendered, 0o644); err != nil {
		fmt.Fprintf(d.Err, "failed to write locale file: %v\n", err)
		return 1
	}

	fmt.Fprintf(d.Out, "Generated locale: %s\n", targetPath)
	return 0
}

func ParseMakeLocaleArgs(args []string) (MakeLocaleOptions, error) {
	opts := MakeLocaleOptions{}
	if len(args) == 0 {
		return opts, errors.New("usage: ship make:locale <code>")
	}
	code := normalizeLocaleCode(args[0])
	if code == "" || strings.HasPrefix(args[0], "-") {
		return opts, errors.New("usage: ship make:locale <code>")
	}
	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "-") {
			return opts, fmt.Errorf("unknown option: %s", arg)
		}
		return opts, fmt.Errorf("unexpected argument: %s", arg)
	}
	opts.Code = code
	return opts, nil
}

func normalizeLocaleCode(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}
		return ""
	}
	return value
}

func cloneLocaleTreeWithEmptyValues(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[key] = cloneLocaleTreeWithEmptyValues(child)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for idx, child := range typed {
			out[idx] = cloneLocaleTreeWithEmptyValues(child)
		}
		return out
	default:
		return ""
	}
}
