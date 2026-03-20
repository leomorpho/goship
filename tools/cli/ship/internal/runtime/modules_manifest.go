package runtime

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type ModulesManifest struct {
	Modules []string `yaml:"modules"`
}

func LoadModulesManifest(path string) (ModulesManifest, error) {
	var m ModulesManifest
	b, err := os.ReadFile(path)
	if err != nil {
		return m, err
	}
	if err := yaml.Unmarshal(b, &m); err != nil {
		return m, fmt.Errorf("parse modules manifest: %w", err)
	}
	normalized, err := NormalizeModules(m.Modules)
	if err != nil {
		return m, err
	}
	m.Modules = normalized
	return m, nil
}

func NormalizeModules(in []string) ([]string, error) {
	if len(in) == 0 {
		return []string{}, nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, raw := range in {
		v := strings.ToLower(strings.TrimSpace(raw))
		if v == "" {
			continue
		}
		if !isModuleToken(v) {
			return nil, fmt.Errorf("invalid module entry %q: use [a-z0-9_-]", raw)
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out, nil
}

func isModuleToken(v string) bool {
	for _, r := range v {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}
