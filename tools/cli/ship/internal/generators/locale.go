package generators

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
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
	sourcePath, err := resolveLocaleSourcePath(localesDir)
	if err != nil {
		fmt.Fprintf(d.Err, "%v\n", err)
		return 1
	}
	targetPath := filepath.Join(localesDir, opts.Code+".toml")

	if _, err := os.Stat(targetPath); err == nil {
		fmt.Fprintf(d.Err, "refusing to overwrite existing locale file: %s\n", targetPath)
		return 1
	}

	source, err := loadLocaleFlat(sourcePath)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to parse source locale file: %v\n", err)
		return 1
	}
	empty := map[string]string{}
	for key := range source {
		empty[key] = ""
	}
	rendered := []byte(renderLocaleFlatTOML(empty))

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

func resolveLocaleSourcePath(localesDir string) (string, error) {
	candidates := []string{
		filepath.Join(localesDir, "en.toml"),
		filepath.Join(localesDir, "en.yaml"),
		filepath.Join(localesDir, "en.yml"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("missing source locale file: %s", filepath.Join(localesDir, "en.toml"))
}

func loadLocaleFlat(path string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(path))
	var source map[string]any
	switch ext {
	case ".toml":
		if _, err := toml.Decode(string(raw), &source); err != nil {
			return nil, err
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(raw, &source); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported source locale format: %s", ext)
	}

	out := map[string]string{}
	flattenLocaleTree("", source, out)
	return out, nil
}

func flattenLocaleTree(prefix string, value any, out map[string]string) {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			next := key
			if prefix != "" {
				next = prefix + "." + key
			}
			flattenLocaleTree(next, typed[key], out)
		}
	case []any:
		if prefix != "" {
			out[prefix] = strings.TrimSpace(fmt.Sprint(typed))
		}
	case string:
		if prefix != "" {
			out[prefix] = strings.TrimSpace(typed)
		}
	case nil:
		if prefix != "" {
			out[prefix] = ""
		}
	default:
		if prefix != "" {
			out[prefix] = strings.TrimSpace(fmt.Sprint(typed))
		}
	}
}

func renderLocaleFlatTOML(flat map[string]string) string {
	keys := make([]string, 0, len(flat))
	for key := range flat {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, key := range keys {
		b.WriteString(strconv.Quote(key))
		b.WriteString(" = ")
		b.WriteString(strconv.Quote(flat[key]))
		b.WriteByte('\n')
	}
	return b.String()
}

func cloneLocaleTreeWithEmptyValues(value any) any {
	// preserved for compatibility with existing callers/tests in this package.
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
