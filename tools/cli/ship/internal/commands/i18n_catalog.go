package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

type localeFormat int

const (
	localeFormatUnknown localeFormat = iota
	localeFormatYAML
	localeFormatTOML
)

func detectLocaleFormat(path string) localeFormat {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".toml":
		return localeFormatTOML
	case ".yaml", ".yml":
		return localeFormatYAML
	default:
		return localeFormatUnknown
	}
}

func localePriority(format localeFormat) int {
	switch format {
	case localeFormatTOML:
		return 2
	case localeFormatYAML:
		return 1
	default:
		return 0
	}
}

func localeCodeFromFilename(name string) string {
	ext := filepath.Ext(name)
	if ext == "" {
		return ""
	}
	code := strings.TrimSpace(strings.TrimSuffix(name, ext))
	if code == "" {
		return ""
	}
	return strings.ToLower(code)
}

func resolveEnglishLocalePath(localesDir string) (string, error) {
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

func resolveEnglishLocalePathForWrite(localesDir string) string {
	candidates := []string{
		filepath.Join(localesDir, "en.toml"),
		filepath.Join(localesDir, "en.yaml"),
		filepath.Join(localesDir, "en.yml"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return filepath.Join(localesDir, "en.toml")
}

func collectPreferredLocaleFiles(localesDir string) (map[string]string, error) {
	entries, err := os.ReadDir(localesDir)
	if err != nil {
		return nil, err
	}

	type chosenLocaleFile struct {
		path     string
		priority int
	}

	chosen := map[string]chosenLocaleFile{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		code := localeCodeFromFilename(entry.Name())
		if code == "" {
			continue
		}
		format := detectLocaleFormat(entry.Name())
		if format == localeFormatUnknown {
			continue
		}
		candidate := chosenLocaleFile{
			path:     filepath.Join(localesDir, entry.Name()),
			priority: localePriority(format),
		}
		existing, ok := chosen[code]
		if !ok || candidate.priority > existing.priority || (candidate.priority == existing.priority && candidate.path < existing.path) {
			chosen[code] = candidate
		}
	}

	out := make(map[string]string, len(chosen))
	for code, selected := range chosen {
		out[code] = selected.path
	}
	return out, nil
}

func loadLocaleFlatFromFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	switch detectLocaleFormat(path) {
	case localeFormatYAML:
		return loadLocaleFlatFromYAML(data)
	case localeFormatTOML:
		return loadLocaleFlatFromTOML(data)
	default:
		return nil, fmt.Errorf("unsupported locale file extension: %s", filepath.Ext(path))
	}
}

func loadLocaleFlatFromYAML(data []byte) (map[string]string, error) {
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	out := map[string]string{}
	flattenLocaleTree("", raw, out)
	return out, nil
}

func loadLocaleFlatFromTOML(data []byte) (map[string]string, error) {
	var raw map[string]any
	if _, err := toml.Decode(string(data), &raw); err != nil {
		return nil, err
	}
	out := map[string]string{}
	flattenLocaleTree("", raw, out)
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

func renderCanonicalTOML(flat map[string]string) string {
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
