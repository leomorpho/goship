package i18ncheck

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

var (
	countCallPattern  = regexp.MustCompile(`(?:I18n\.TC|i18n\.TC)\s*\(\s*(?:[^,]+,\s*)?"([a-zA-Z0-9._-]+)"`)
	selectCallPattern = regexp.MustCompile(`(?:I18n\.TS|i18n\.TS)\s*\(\s*(?:[^,]+,\s*)?"([a-zA-Z0-9._-]+)"`)
)

type CompletenessIssue struct {
	ID      string
	Locale  string
	Kind    string
	BaseKey string
	File    string
	Message string
}

func CollectCompletenessIssues(root string) ([]CompletenessIssue, error) {
	preferred, err := collectPreferredLocaleFiles(filepath.Join(root, "locales"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if len(preferred) == 0 {
		return nil, nil
	}

	countBases, err := collectI18nCallBases(root, countCallPattern)
	if err != nil {
		return nil, err
	}
	selectBases, err := collectI18nCallBases(root, selectCallPattern)
	if err != nil {
		return nil, err
	}
	if len(countBases) == 0 && len(selectBases) == 0 {
		return nil, nil
	}

	locales := make([]string, 0, len(preferred))
	for locale := range preferred {
		locales = append(locales, locale)
	}
	sort.Strings(locales)
	countBaseList := sortedSetKeys(countBases)
	selectBaseList := sortedSetKeys(selectBases)

	issues := make([]CompletenessIssue, 0)
	for _, locale := range locales {
		path := preferred[locale]
		flat, err := loadLocaleFlat(path)
		if err != nil {
			return nil, fmt.Errorf("parse locale file %s: %w", filepath.Base(path), err)
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)

		for _, base := range countBaseList {
			otherKey := base + ".other"
			if strings.TrimSpace(flat[otherKey]) == "" {
				issues = append(issues, newCompletenessIssue(locale, rel, "plural_missing_other", base, fmt.Sprintf("missing required plural fallback key %s for TC usage", otherKey)))
			}
			hasCountVariant := false
			for _, suffix := range []string{"zero", "one", "two", "few", "many"} {
				if strings.TrimSpace(flat[base+"."+suffix]) != "" {
					hasCountVariant = true
					break
				}
			}
			if !hasCountVariant {
				issues = append(issues, newCompletenessIssue(locale, rel, "plural_missing_variant", base, fmt.Sprintf("missing non-other plural variant for TC usage of %s", base)))
			}
		}

		for _, base := range selectBaseList {
			otherKey := base + ".other"
			if strings.TrimSpace(flat[otherKey]) == "" {
				issues = append(issues, newCompletenessIssue(locale, rel, "select_missing_other", base, fmt.Sprintf("missing required select fallback key %s for TS usage", otherKey)))
			}
			prefix := base + "."
			hasVariant := false
			for key, value := range flat {
				if !strings.HasPrefix(key, prefix) || strings.TrimSpace(value) == "" {
					continue
				}
				suffix := strings.TrimPrefix(key, prefix)
				if suffix == "" || suffix == "other" || strings.Contains(suffix, ".") {
					continue
				}
				hasVariant = true
				break
			}
			if !hasVariant {
				issues = append(issues, newCompletenessIssue(locale, rel, "select_missing_variant", base, fmt.Sprintf("missing non-other select variant for TS usage of %s", base)))
			}
		}
	}

	sort.Slice(issues, func(i, j int) bool {
		left := issues[i]
		right := issues[j]
		if left.Locale != right.Locale {
			return left.Locale < right.Locale
		}
		if left.File != right.File {
			return left.File < right.File
		}
		if left.Kind != right.Kind {
			return left.Kind < right.Kind
		}
		if left.BaseKey != right.BaseKey {
			return left.BaseKey < right.BaseKey
		}
		return left.ID < right.ID
	})
	return issues, nil
}

func collectPreferredLocaleFiles(localesDir string) (map[string]string, error) {
	entries, err := os.ReadDir(localesDir)
	if err != nil {
		return nil, err
	}
	type selected struct {
		path     string
		priority int
	}
	chosen := map[string]selected{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		priority := 0
		switch ext {
		case ".toml":
			priority = 2
		case ".yaml", ".yml":
			priority = 1
		default:
			continue
		}
		locale := strings.ToLower(strings.TrimSpace(strings.TrimSuffix(name, ext)))
		if locale == "" {
			continue
		}
		path := filepath.Join(localesDir, name)
		existing, ok := chosen[locale]
		if !ok || priority > existing.priority || (priority == existing.priority && path < existing.path) {
			chosen[locale] = selected{path: path, priority: priority}
		}
	}
	out := map[string]string{}
	for locale, value := range chosen {
		out[locale] = value.path
	}
	return out, nil
}

func loadLocaleFlat(path string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var decoded map[string]any
	switch strings.ToLower(filepath.Ext(path)) {
	case ".toml":
		if _, err := toml.Decode(string(raw), &decoded); err != nil {
			return nil, err
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(raw, &decoded); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported locale file extension: %s", filepath.Ext(path))
	}
	out := map[string]string{}
	flattenLocale("", decoded, out)
	return out, nil
}

func flattenLocale(prefix string, value any, out map[string]string) {
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
			flattenLocale(next, typed[key], out)
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

func collectI18nCallBases(root string, pattern *regexp.Regexp) (map[string]struct{}, error) {
	out := map[string]struct{}{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".docket", "node_modules", "tmp", "vendor", "gen":
				return filepath.SkipDir
			default:
				return nil
			}
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".go" && ext != ".templ" {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		matches := pattern.FindAllStringSubmatch(string(raw), -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			base := strings.TrimSpace(match[1])
			if base == "" {
				continue
			}
			out[base] = struct{}{}
		}
		return nil
	})
	return out, err
}

func sortedSetKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func newCompletenessIssue(locale, file, kind, base, message string) CompletenessIssue {
	idBase := locale + "|" + file + "|" + kind + "|" + base
	sum := sha1.Sum([]byte(idBase))
	return CompletenessIssue{
		ID:      "I18N-C-" + strings.ToUpper(hex.EncodeToString(sum[:]))[:10],
		Locale:  locale,
		Kind:    kind,
		BaseKey: base,
		File:    file,
		Message: message,
	}
}
