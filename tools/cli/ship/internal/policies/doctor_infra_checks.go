package policies

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func checkGoWorkModules(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	goWorkPath := filepath.Join(root, "go.work")
	if !hasFile(goWorkPath) {
		return issues
	}
	b, err := os.ReadFile(goWorkPath)
	if err != nil {
		return append(issues, DoctorIssue{
			Code:    "DX014",
			Message: "failed to read go.work",
			Fix:     err.Error(),
		})
	}
	modules := parseGoWorkUseModules(string(b))
	for _, modPath := range modules {
		p := filepath.Clean(filepath.Join(root, filepath.FromSlash(modPath)))
		if !hasFile(filepath.Join(p, "go.mod")) {
			issues = append(issues, DoctorIssue{
				Code:    "DX014",
				Message: fmt.Sprintf("go.work references missing module go.mod: %s", modPath),
				Fix:     fmt.Sprintf("create %s/go.mod or remove %s from go.work use()", filepath.ToSlash(filepath.Join(modPath)), modPath),
			})
		}
	}
	return issues
}

func parseGoWorkUseModules(content string) []string {
	modules := make([]string, 0)
	lines := strings.Split(content, "\n")
	inUseBlock := false
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if strings.HasPrefix(line, "use ") {
			rest := strings.TrimSpace(strings.TrimPrefix(line, "use"))
			if rest == "(" {
				inUseBlock = true
				continue
			}
			rest = trimInlineComment(rest)
			rest = strings.Trim(rest, "\"")
			if rest != "" {
				modules = append(modules, rest)
			}
			continue
		}
		if inUseBlock {
			if line == ")" {
				inUseBlock = false
				continue
			}
			line = trimInlineComment(line)
			line = strings.Trim(line, "\"")
			if line != "" {
				modules = append(modules, line)
			}
		}
	}
	return modules
}

func checkDockerIgnoreCoverage(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	if !hasFile(filepath.Join(root, "infra", "docker", "Dockerfile")) {
		return issues
	}
	p := filepath.Join(root, ".dockerignore")
	if !hasFile(p) {
		return append(issues, DoctorIssue{
			Code:    "DX015",
			Message: "missing .dockerignore",
			Fix:     "add .dockerignore with heavy-path exclusions to keep docker build context small",
		})
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return append(issues, DoctorIssue{
			Code:    "DX015",
			Message: "failed to read .dockerignore",
			Fix:     err.Error(),
		})
	}
	text := string(b)
	requiredEntries := []string{
		".git",
		"node_modules",
		"frontend/node_modules",
		".local",
		"tools/scripts/venv",
	}
	for _, entry := range requiredEntries {
		if !containsDockerIgnoreEntry(text, entry) {
			issues = append(issues, DoctorIssue{
				Code:    "DX015",
				Message: fmt.Sprintf(".dockerignore missing required context exclusion: %s", entry),
				Fix:     "add required exclusion to keep docker build context small and stable",
			})
		}
	}
	return issues
}

func containsDockerIgnoreEntry(content, token string) bool {
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if line == token || line == "/"+token {
			return true
		}
	}
	return false
}

func checkDockerLocalReplaceOrder(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	dockerfile := filepath.Join(root, "infra", "docker", "Dockerfile")
	if !hasFile(dockerfile) {
		return issues
	}
	localReplaces := collectLocalReplaces(root)
	if len(localReplaces) == 0 {
		return issues
	}
	b, err := os.ReadFile(dockerfile)
	if err != nil {
		return append(issues, DoctorIssue{
			Code:    "DX016",
			Message: "failed to read infra/docker/Dockerfile",
			Fix:     err.Error(),
		})
	}
	lines := strings.Split(string(b), "\n")
	downloadIdx := -1
	copyAllIdx := -1
	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if downloadIdx == -1 && strings.Contains(line, "go mod download") {
			downloadIdx = i
		}
		if copyAllIdx == -1 && strings.HasPrefix(line, "COPY ") && strings.Contains(line, ". .") {
			copyAllIdx = i
		}
	}
	if downloadIdx == -1 {
		return append(issues, DoctorIssue{
			Code:    "DX016",
			Message: "Dockerfile does not run go mod download",
			Fix:     "add a deterministic go mod download step in builder stage",
		})
	}
	if copyAllIdx != -1 && copyAllIdx < downloadIdx {
		return issues
	}
	for _, rel := range localReplaces {
		found := false
		for i, raw := range lines {
			if i >= downloadIdx {
				break
			}
			line := strings.TrimSpace(raw)
			if !strings.HasPrefix(line, "COPY ") {
				continue
			}
			if strings.Contains(line, rel) || strings.Contains(line, filepath.ToSlash(rel)) {
				found = true
				break
			}
		}
		if !found {
			issues = append(issues, DoctorIssue{
				Code:    "DX016",
				Message: fmt.Sprintf("Dockerfile may fail local replace before go mod download: missing COPY for %s", rel),
				Fix:     "copy local replace paths (or COPY . .) before the first go mod download",
			})
		}
	}
	return issues
}

func collectLocalReplaces(root string) []string {
	paths := make([]string, 0)
	seen := map[string]struct{}{}
	goModFiles := []string{
		filepath.Join(root, "go.mod"),
	}
	for _, gm := range goModFiles {
		if !hasFile(gm) {
			continue
		}
		moduleRoot := filepath.Dir(gm)
		for _, p := range parseLocalReplacePaths(gm) {
			abs := filepath.Clean(filepath.Join(moduleRoot, filepath.FromSlash(p)))
			rel, err := filepath.Rel(root, abs)
			if err != nil {
				continue
			}
			rel = filepath.ToSlash(rel)
			if strings.HasPrefix(rel, "..") {
				continue
			}
			if _, ok := seen[rel]; ok {
				continue
			}
			seen[rel] = struct{}{}
			paths = append(paths, rel)
		}
	}
	return paths
}

func parseLocalReplacePaths(goModPath string) []string {
	b, err := os.ReadFile(goModPath)
	if err != nil {
		return nil
	}
	paths := make([]string, 0)
	inReplaceBlock := false
	replaceRe := regexp.MustCompile(`\s+=>\s+([^\s]+)`)
	lines := strings.Split(string(b), "\n")
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if strings.HasPrefix(line, "replace ") {
			rest := strings.TrimSpace(strings.TrimPrefix(line, "replace"))
			if rest == "(" {
				inReplaceBlock = true
				continue
			}
			if p := parseReplacePath(rest, replaceRe); p != "" {
				paths = append(paths, p)
			}
			continue
		}
		if inReplaceBlock {
			if line == ")" {
				inReplaceBlock = false
				continue
			}
			if p := parseReplacePath(line, replaceRe); p != "" {
				paths = append(paths, p)
			}
		}
	}
	return paths
}

func parseReplacePath(line string, re *regexp.Regexp) string {
	line = trimInlineComment(line)
	m := re.FindStringSubmatch(line)
	if len(m) != 2 {
		return ""
	}
	p := strings.TrimSpace(strings.Trim(m[1], "\""))
	if strings.HasPrefix(p, ".") {
		return filepath.ToSlash(p)
	}
	return ""
}

func trimInlineComment(line string) string {
	if idx := strings.Index(line, "//"); idx >= 0 {
		return strings.TrimSpace(line[:idx])
	}
	return strings.TrimSpace(line)
}
