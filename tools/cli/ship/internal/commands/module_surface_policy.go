package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const moduleSurfaceResetDocRelPath = "docs/architecture/11-module-surface-reset.md"
const moduleSurfaceCatalogRelPath = "config/module-surface.yaml"

var (
	moduleSurfaceDecisionRowPattern = regexp.MustCompile(`^\|\s*` + "`" + `([^` + "`" + `]+)` + "`" + `\s*\|\s*` + "`" + `([^` + "`" + `]+)` + "`" + `\s*\|\s*` + "`" + `([^` + "`" + `]+)` + "`" + `\s*\|`)
	allowedModuleSurfaceClasses     = map[string]struct{}{
		"core":        {},
		"battery":     {},
		"starter-app": {},
		"delete":      {},
	}
	allowedModuleSurfaceDecisions = map[string]struct{}{
		"keep":    {},
		"rewrite": {},
		"eject":   {},
	}
)

type moduleSurfaceDecision struct {
	Class    string
	Decision string
}

type moduleSurfaceCatalog struct {
	Version    string                   `yaml:"version"`
	Candidates []moduleSurfaceCandidate `yaml:"candidates"`
}

type moduleSurfaceCandidate struct {
	ID          string `yaml:"id"`
	Class       string `yaml:"class"`
	Decision    string `yaml:"decision"`
	Destination string `yaml:"destination"`
	Rationale   string `yaml:"rationale"`
}

func checkModuleSurfaceResetPolicy(root string) error {
	if !isGoShipFrameworkRepo(root) {
		return nil
	}

	catalogPath := filepath.Join(root, moduleSurfaceCatalogRelPath)
	catalogBody, err := os.ReadFile(catalogPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", filepath.ToSlash(moduleSurfaceCatalogRelPath), err)
	}
	catalogDecisions, err := parseModuleSurfaceCatalog(catalogBody)
	if err != nil {
		return fmt.Errorf("%s: %w", filepath.ToSlash(moduleSurfaceCatalogRelPath), err)
	}

	docPath := filepath.Join(root, moduleSurfaceResetDocRelPath)
	docBody, err := os.ReadFile(docPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", filepath.ToSlash(moduleSurfaceResetDocRelPath), err)
	}
	doc := string(docBody)

	for _, token := range []string{
		"# Module Surface Reset",
		"config/module-surface.yaml",
		"## Canonical Battery Contract",
		"## Decision Matrix",
		"## Notifications Replacement Plan",
	} {
		if !strings.Contains(doc, token) {
			return fmt.Errorf("%s missing required section %q", filepath.ToSlash(moduleSurfaceResetDocRelPath), token)
		}
	}
	for _, token := range []string{
		"notifications-inbox",
		"notifications-push",
		"notifications-email",
		"notifications-sms",
		"notifications-schedule",
	} {
		if !strings.Contains(doc, token) {
			return fmt.Errorf("%s missing notifications split target %q", filepath.ToSlash(moduleSurfaceResetDocRelPath), token)
		}
	}

	decisions, err := parseModuleSurfaceDecisions(doc)
	if err != nil {
		return fmt.Errorf("%s: %w", filepath.ToSlash(moduleSurfaceResetDocRelPath), err)
	}
	for id, decision := range decisions {
		catalogDecision, ok := catalogDecisions[id]
		if !ok {
			return fmt.Errorf("%s missing candidate %q declared in %s", filepath.ToSlash(moduleSurfaceCatalogRelPath), id, filepath.ToSlash(moduleSurfaceResetDocRelPath))
		}
		if catalogDecision != decision {
			return fmt.Errorf(
				"candidate %q drift between %s and %s (catalog=%s/%s docs=%s/%s)",
				id,
				filepath.ToSlash(moduleSurfaceCatalogRelPath),
				filepath.ToSlash(moduleSurfaceResetDocRelPath),
				catalogDecision.Class,
				catalogDecision.Decision,
				decision.Class,
				decision.Decision,
			)
		}
	}

	moduleCandidates, err := listFirstPartyModuleCandidates(filepath.Join(root, "modules"))
	if err != nil {
		return err
	}
	for _, candidate := range moduleCandidates {
		decision, ok := catalogDecisions[candidate]
		if !ok {
			return fmt.Errorf("missing decision row for first-party module candidate %q in %s", candidate, filepath.ToSlash(moduleSurfaceCatalogRelPath))
		}
		if decision.Class == "battery" && decision.Decision == "keep" {
			localGoMod := filepath.Join(root, "modules", candidate, "go.mod")
			if _, statErr := os.Stat(localGoMod); statErr != nil {
				if os.IsNotExist(statErr) {
					return fmt.Errorf("keep+battery candidate %q must be standalone with modules/%s/go.mod", candidate, candidate)
				}
				return fmt.Errorf("stat modules/%s/go.mod: %w", candidate, statErr)
			}
		}
	}

	policiesByID := map[string]moduleInfo{}
	for _, info := range standaloneModulePolicies(root) {
		id := strings.TrimSpace(info.ID)
		if id == "" {
			continue
		}
		if _, exists := policiesByID[id]; exists {
			continue
		}
		policiesByID[id] = info
	}
	for id, info := range policiesByID {
		decision, ok := catalogDecisions[id]
		if !ok {
			return fmt.Errorf("missing decision row for standalone module policy %q in %s", id, filepath.ToSlash(moduleSurfaceCatalogRelPath))
		}
		if decision.Class != "battery" || decision.Decision != "keep" {
			return fmt.Errorf("standalone module policy %q must be classified as class=battery decision=keep in %s", id, filepath.ToSlash(moduleSurfaceCatalogRelPath))
		}
		localGoMod := filepath.Join(root, info.LocalPath, "go.mod")
		if _, statErr := os.Stat(localGoMod); statErr != nil {
			if os.IsNotExist(statErr) {
				return fmt.Errorf("standalone module policy %q requires %s", id, filepath.ToSlash(filepath.Join(info.LocalPath, "go.mod")))
			}
			return fmt.Errorf("stat %s: %w", filepath.ToSlash(filepath.Join(info.LocalPath, "go.mod")), statErr)
		}
	}

	return nil
}

func parseModuleSurfaceCatalog(content []byte) (map[string]moduleSurfaceDecision, error) {
	var catalog moduleSurfaceCatalog
	if err := yaml.Unmarshal(content, &catalog); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	if strings.TrimSpace(catalog.Version) == "" {
		return nil, fmt.Errorf("version is required")
	}
	decisions := map[string]moduleSurfaceDecision{}
	for _, candidate := range catalog.Candidates {
		id := strings.TrimSpace(candidate.ID)
		class := strings.TrimSpace(candidate.Class)
		decision := strings.TrimSpace(candidate.Decision)
		if id == "" {
			return nil, fmt.Errorf("candidate id cannot be empty")
		}
		if _, ok := allowedModuleSurfaceClasses[class]; !ok {
			return nil, fmt.Errorf("invalid class %q for candidate %q (allowed: core, battery, starter-app, delete)", class, id)
		}
		if _, ok := allowedModuleSurfaceDecisions[decision]; !ok {
			return nil, fmt.Errorf("invalid decision %q for candidate %q (allowed: keep, rewrite, eject)", decision, id)
		}
		if _, exists := decisions[id]; exists {
			return nil, fmt.Errorf("duplicate decision entry for candidate %q", id)
		}
		decisions[id] = moduleSurfaceDecision{Class: class, Decision: decision}
	}
	return decisions, nil
}

func parseModuleSurfaceDecisions(content string) (map[string]moduleSurfaceDecision, error) {
	decisions := map[string]moduleSurfaceDecision{}
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		matches := moduleSurfaceDecisionRowPattern.FindStringSubmatch(line)
		if len(matches) != 4 {
			continue
		}
		id := strings.TrimSpace(matches[1])
		class := strings.TrimSpace(matches[2])
		decision := strings.TrimSpace(matches[3])
		if id == "" {
			continue
		}
		if _, ok := allowedModuleSurfaceClasses[class]; !ok {
			return nil, fmt.Errorf("invalid class %q for candidate %q (allowed: core, battery, starter-app, delete)", class, id)
		}
		if _, ok := allowedModuleSurfaceDecisions[decision]; !ok {
			return nil, fmt.Errorf("invalid decision %q for candidate %q (allowed: keep, rewrite, eject)", decision, id)
		}
		if _, exists := decisions[id]; exists {
			return nil, fmt.Errorf("duplicate decision entry for candidate %q", id)
		}
		decisions[id] = moduleSurfaceDecision{Class: class, Decision: decision}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return decisions, nil
}

func listFirstPartyModuleCandidates(modulesRoot string) ([]string, error) {
	entries, err := os.ReadDir(modulesRoot)
	if err != nil {
		return nil, fmt.Errorf("read modules directory: %w", err)
	}
	candidates := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if strings.HasPrefix(name, ".") || name == "" {
			continue
		}
		candidates = append(candidates, name)
	}
	sort.Strings(candidates)
	return candidates, nil
}

func isGoShipFrameworkRepo(root string) bool {
	required := []string{
		filepath.Join(root, "modules"),
		filepath.Join(root, "tools", "cli", "ship", "internal", "commands", "module.go"),
	}
	for _, path := range required {
		if _, err := os.Stat(path); err != nil {
			return false
		}
	}
	return true
}
