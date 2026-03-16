package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type I18nDeps struct {
	Out          io.Writer
	Err          io.Writer
	FindGoModule func(start string) (string, string, error)
}

func RunI18n(args []string, d I18nDeps) int {
	if len(args) == 0 {
		PrintI18nHelp(d.Out)
		return 1
	}
	if args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		PrintI18nHelp(d.Out)
		return 0
	}
	if d.FindGoModule == nil {
		fmt.Fprintln(d.Err, "i18n commands require FindGoModule dependency")
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

	switch args[0] {
	case "missing":
		return runI18nMissing(d, root)
	case "unused":
		return runI18nUnused(d, root)
	default:
		fmt.Fprintf(d.Err, "unknown i18n command: %s\n\n", args[0])
		PrintI18nHelp(d.Err)
		return 1
	}
}

func PrintI18nHelp(w io.Writer) {
	fmt.Fprintln(w, "ship i18n commands:")
	fmt.Fprintln(w, "  ship i18n:missing")
	fmt.Fprintln(w, "  ship i18n:unused")
}

func runI18nMissing(d I18nDeps, root string) int {
	localesDir := filepath.Join(root, "locales")
	base, err := loadLocaleFlat(filepath.Join(localesDir, "en.yaml"))
	if err != nil {
		fmt.Fprintf(d.Err, "i18n:missing failed to read locales/en.yaml: %v\n", err)
		return 1
	}

	entries, err := os.ReadDir(localesDir)
	if err != nil {
		fmt.Fprintf(d.Err, "i18n:missing failed to read locales dir: %v\n", err)
		return 1
	}

	lines := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" || entry.Name() == "en.yaml" {
			continue
		}
		localeCode := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		current, err := loadLocaleFlat(filepath.Join(localesDir, entry.Name()))
		if err != nil {
			fmt.Fprintf(d.Err, "i18n:missing failed to parse %s: %v\n", entry.Name(), err)
			return 1
		}

		keys := make([]string, 0, len(base))
		for key := range base {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			value, ok := current[key]
			if !ok || strings.TrimSpace(value) == "" {
				lines = append(lines, fmt.Sprintf("%s: %s", localeCode, key))
			}
		}
	}

	if len(lines) == 0 {
		fmt.Fprintln(d.Out, "All locale keys are translated.")
		return 0
	}
	for _, line := range lines {
		fmt.Fprintln(d.Out, line)
	}
	return 0
}

func runI18nUnused(d I18nDeps, root string) int {
	localesDir := filepath.Join(root, "locales")
	entries, err := os.ReadDir(localesDir)
	if err != nil {
		fmt.Fprintf(d.Err, "i18n:unused failed to read locales dir: %v\n", err)
		return 1
	}

	allKeys := map[string]struct{}{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		current, err := loadLocaleFlat(filepath.Join(localesDir, entry.Name()))
		if err != nil {
			fmt.Fprintf(d.Err, "i18n:unused failed to parse %s: %v\n", entry.Name(), err)
			return 1
		}
		for key := range current {
			allKeys[key] = struct{}{}
		}
	}

	used, err := collectUsedI18nKeys(root)
	if err != nil {
		fmt.Fprintf(d.Err, "i18n:unused failed to scan source files: %v\n", err)
		return 1
	}

	unused := make([]string, 0)
	for key := range allKeys {
		if _, ok := used[key]; !ok {
			unused = append(unused, key)
		}
	}
	sort.Strings(unused)

	if len(unused) == 0 {
		fmt.Fprintln(d.Out, "No unused locale keys found.")
		return 0
	}
	for _, key := range unused {
		fmt.Fprintln(d.Out, key)
	}
	return 0
}

func loadLocaleFlat(path string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var data map[string]any
	if err := yaml.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	out := map[string]string{}
	flattenLocaleValue("", data, out)
	return out, nil
}

func flattenLocaleValue(prefix string, value any, out map[string]string) {
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
			flattenLocaleValue(next, typed[key], out)
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

var i18nKeyUsePattern = regexp.MustCompile(`(?:I18n\.T|i18n\.T)\s*\([^)]*"([a-zA-Z0-9._-]+)"`)

func collectUsedI18nKeys(root string) (map[string]struct{}, error) {
	used := map[string]struct{}{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			base := filepath.Base(path)
			switch base {
			case ".git", "node_modules", ".docket", "tmp":
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".go" && ext != ".templ" {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		matches := i18nKeyUsePattern.FindAllStringSubmatch(string(content), -1)
		for _, match := range matches {
			if len(match) >= 2 {
				used[match[1]] = struct{}{}
			}
		}
		return nil
	})
	return used, err
}
