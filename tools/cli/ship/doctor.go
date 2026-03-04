package ship

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type doctorIssue struct {
	Code    string
	Message string
	Fix     string
}

func (c CLI) runDoctor(args []string) int {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			printDoctorHelp(c.Out)
			return 0
		}
	}
	if len(args) > 0 {
		fmt.Fprintf(c.Err, "unexpected doctor arguments: %v\n", args)
		return 1
	}

	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(c.Err, "failed to resolve working directory: %v\n", err)
		return 1
	}
	root, _, err := findGoModule(wd)
	if err != nil {
		fmt.Fprintf(c.Err, "failed to resolve project root (go.mod): %v\n", err)
		return 1
	}

	issues := runDoctorChecks(root)
	if len(issues) == 0 {
		fmt.Fprintf(c.Out, "ship doctor: OK (%s)\n", root)
		return 0
	}

	fmt.Fprintf(c.Err, "ship doctor: found %d issue(s)\n", len(issues))
	for _, issue := range issues {
		fmt.Fprintf(c.Err, "- [%s] %s\n", issue.Code, issue.Message)
		if issue.Fix != "" {
			fmt.Fprintf(c.Err, "  fix: %s\n", issue.Fix)
		}
	}
	return 1
}

func runDoctorChecks(root string) []doctorIssue {
	issues := make([]doctorIssue, 0)

	requiredDirs := []string{
		filepath.Join("apps", "site"),
		filepath.Join("apps", "site", "app"),
		filepath.Join("apps", "site", "foundation"),
		filepath.Join("apps", "site", "web", "controllers"),
		filepath.Join("apps", "site", "web", "middleware"),
		filepath.Join("apps", "site", "web", "ui"),
		filepath.Join("apps", "site", "web", "viewmodels"),
		filepath.Join("apps", "site", "jobs"),
		filepath.Join("apps", "site", "views"),
		filepath.Join("apps", "db", "schema"),
		filepath.Join("apps", "db", "migrate", "migrations"),
	}
	for _, rel := range requiredDirs {
		if !isDir(filepath.Join(root, rel)) {
			issues = append(issues, doctorIssue{
				Code:    "DX001",
				Message: fmt.Sprintf("missing required directory: %s", rel),
				Fix:     fmt.Sprintf("create %s or regenerate the app scaffold with `ship new`", rel),
			})
		}
	}

	requiredFiles := []string{
		filepath.Join("apps", "site", "router.go"),
		filepath.Join("apps", "site", "foundation", "container.go"),
		filepath.Join("apps", "site", "web", "routenames", "routenames.go"),
		filepath.Join("config", "modules.yaml"),
		filepath.Join("docs", "00-index.md"),
		filepath.Join("docs", "architecture", "01-architecture.md"),
		filepath.Join("docs", "architecture", "08-cognitive-model.md"),
	}
	for _, rel := range requiredFiles {
		if !hasFile(filepath.Join(root, rel)) {
			issues = append(issues, doctorIssue{
				Code:    "DX002",
				Message: fmt.Sprintf("missing required file: %s", rel),
				Fix:     "restore missing documentation or scaffold files",
			})
		}
	}

	forbidden := []string{
		filepath.Join("app", "site"),
		filepath.Join("apps", "site", "bootstrap"),
		filepath.Join("apps", "site", "domains"),
		filepath.Join("apps", "site", "tasks"),
		filepath.Join("apps", "site", "types"),
		filepath.Join("apps", "site", "webui"),
		filepath.Join("apps", "site", "middleware"),
	}
	for _, rel := range forbidden {
		if pathExists(filepath.Join(root, rel)) {
			issues = append(issues, doctorIssue{
				Code:    "DX003",
				Message: fmt.Sprintf("forbidden legacy path present: %s", rel),
				Fix:     "remove or migrate legacy paths to canonical app layout",
			})
		}
	}

	rootBinaries := []string{"web", "worker", "seed", "ship", "ship-mcp"}
	for _, name := range rootBinaries {
		if hasFile(filepath.Join(root, name)) {
			issues = append(issues, doctorIssue{
				Code:    "DX008",
				Message: fmt.Sprintf("root build artifact present: %s", name),
				Fix:     fmt.Sprintf("remove %s and keep it ignored in .gitignore", name),
			})
		}
	}

	gitignore := filepath.Join(root, ".gitignore")
	if hasFile(gitignore) {
		content, err := os.ReadFile(gitignore)
		if err != nil {
			issues = append(issues, doctorIssue{
				Code:    "DX009",
				Message: "failed to read .gitignore",
				Fix:     err.Error(),
			})
		} else {
			ignoreText := string(content)
			required := []string{"/web", "/worker", "/seed", "/ship", "/ship-mcp"}
			for _, entry := range required {
				if !strings.Contains(ignoreText, entry) {
					issues = append(issues, doctorIssue{
						Code:    "DX009",
						Message: fmt.Sprintf(".gitignore missing required artifact entry: %s", entry),
						Fix:     "add required root binary ignore entries to .gitignore",
					})
				}
			}
		}
	}

	router := filepath.Join(root, "apps", "site", "router.go")
	if hasFile(router) {
		b, err := os.ReadFile(router)
		if err != nil {
			issues = append(issues, doctorIssue{
				Code:    "DX004",
				Message: "failed to read router.go for marker checks",
				Fix:     err.Error(),
			})
		} else {
			content := string(b)
			markers := []string{
				"// ship:routes:public:start",
				"// ship:routes:public:end",
				"// ship:routes:auth:start",
				"// ship:routes:auth:end",
			}
			for _, marker := range markers {
				if !strings.Contains(content, marker) {
					issues = append(issues, doctorIssue{
						Code:    "DX005",
						Message: fmt.Sprintf("missing router marker: %s", marker),
						Fix:     "restore route markers in apps/site/router.go to keep generator wiring deterministic",
					})
				}
			}

			type markerPair struct {
				start string
				end   string
			}
			pairs := []markerPair{
				{start: "// ship:routes:public:start", end: "// ship:routes:public:end"},
				{start: "// ship:routes:auth:start", end: "// ship:routes:auth:end"},
			}
			for _, pair := range pairs {
				startIdx := strings.Index(content, pair.start)
				endIdx := strings.Index(content, pair.end)
				if startIdx >= 0 && endIdx >= 0 && startIdx > endIdx {
					issues = append(issues, doctorIssue{
						Code:    "DX011",
						Message: fmt.Sprintf("router marker order invalid: %s appears after %s", pair.start, pair.end),
						Fix:     "place start marker before end marker to keep --wire deterministic",
					})
				}
			}
		}
	}

	issues = append(issues, checkPackageNaming(root, filepath.Join("apps", "site", "web", "ui"), "ui")...)
	issues = append(issues, checkPackageNaming(root, filepath.Join("apps", "site", "web", "viewmodels"), "viewmodels")...)
	issues = append(issues, checkTopLevelDirs(root)...)
	issues = append(issues, checkFileLengthBudget(root, 500)...)
	issues = append(issues, checkCLIDocsCoverage(root)...)
	issues = append(issues, checkGoWorkModules(root)...)
	issues = append(issues, checkDockerIgnoreCoverage(root)...)
	issues = append(issues, checkDockerLocalReplaceOrder(root)...)

	return issues
}

func checkPackageNaming(root, relDir, expected string) []doctorIssue {
	issues := make([]doctorIssue, 0)
	dir := filepath.Join(root, relDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return issues
		}
		return append(issues, doctorIssue{
			Code:    "DX006",
			Message: fmt.Sprintf("failed reading package directory %s", relDir),
			Fix:     err.Error(),
		})
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		filePath := filepath.Join(dir, entry.Name())
		pkg, err := readPackageClause(filePath)
		if err != nil {
			issues = append(issues, doctorIssue{
				Code:    "DX006",
				Message: fmt.Sprintf("failed reading package clause in %s", filepath.ToSlash(filepath.Join(relDir, entry.Name()))),
				Fix:     err.Error(),
			})
			continue
		}

		allowed := map[string]struct{}{expected: {}}
		if strings.HasSuffix(entry.Name(), "_test.go") {
			allowed[expected+"_test"] = struct{}{}
		}
		if _, ok := allowed[pkg]; !ok {
			issues = append(issues, doctorIssue{
				Code:    "DX007",
				Message: fmt.Sprintf("package mismatch in %s: got %q, want %q (or %q for tests)", filepath.ToSlash(filepath.Join(relDir, entry.Name())), pkg, expected, expected+"_test"),
				Fix:     "align package name with directory convention",
			})
		}
	}

	return issues
}

func readPackageClause(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(b), "\n")
	for _, line := range lines {
		s := strings.TrimSpace(line)
		if s == "" || strings.HasPrefix(s, "//") {
			continue
		}
		if strings.HasPrefix(s, "package ") {
			return strings.TrimSpace(strings.TrimPrefix(s, "package ")), nil
		}
		break
	}
	return "", fmt.Errorf("package clause not found")
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func checkFileLengthBudget(root string, maxLines int) []doctorIssue {
	issues := make([]doctorIssue, 0)
	allowlist := map[string]struct{}{
		filepath.ToSlash(filepath.Join("tools", "cli", "ship", "cli.go")):               {},
		filepath.ToSlash(filepath.Join("tools", "cli", "ship", "cli_test.go")):          {},
		filepath.ToSlash(filepath.Join("tools", "cli", "ship", "doctor.go")):            {},
		filepath.ToSlash(filepath.Join("tools", "cli", "ship", "generate_resource.go")): {},
		filepath.ToSlash(filepath.Join("apps", "site", "app", "profiles", "repo.go")):   {},
	}

	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)

		if d.IsDir() {
			if rel == ".git" || rel == "node_modules" || rel == ".cache" || rel == filepath.ToSlash(filepath.Join("apps", "db", "ent")) || strings.HasPrefix(rel, filepath.ToSlash(filepath.Join("apps", "db", "ent"))+"/") {
				return filepath.SkipDir
			}
			if strings.HasSuffix(rel, "/gen") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(rel, ".go") {
			return nil
		}
		if _, ok := allowlist[rel]; ok {
			return nil
		}
		lines, lineErr := countLines(path)
		if lineErr != nil {
			issues = append(issues, doctorIssue{
				Code:    "DX010",
				Message: fmt.Sprintf("failed counting lines for %s", rel),
				Fix:     lineErr.Error(),
			})
			return nil
		}
		if lines > maxLines {
			issues = append(issues, doctorIssue{
				Code:    "DX010",
				Message: fmt.Sprintf("file exceeds line budget (%d > %d): %s", lines, maxLines, rel),
				Fix:     "split by responsibility to keep files LLM-friendly",
			})
		}
		return nil
	})

	return issues
}

func checkTopLevelDirs(root string) []doctorIssue {
	issues := make([]doctorIssue, 0)
	entries, err := os.ReadDir(root)
	if err != nil {
		return append(issues, doctorIssue{
			Code:    "DX013",
			Message: "failed to read repository root",
			Fix:     err.Error(),
		})
	}

	allowed := map[string]struct{}{
		".cache":     {},
		".git":       {},
		".github":    {},
		".kamal":     {},
		".vscode":    {},
		"apps":       {},
		"cmd":        {},
		"config":     {},
		"data":       {},
		"dbs":        {},
		"docs":       {},
		"ent":        {},
		"framework":  {},
		"infra":      {},
		"javascript": {},
		"modules":    {},
		"tests":      {},
		"tmp":        {},
		"tools":      {},
		"frontend":   {},
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if _, ok := allowed[name]; !ok {
			issues = append(issues, doctorIssue{
				Code:    "DX013",
				Message: fmt.Sprintf("unexpected top-level directory: %s", name),
				Fix:     "move it under apps/, modules/, framework/, tools/, infra/, tests/, or mark as intentional in doctor allow-list",
			})
		}
	}

	return issues
}

func countLines(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	lines := 0
	for s.Scan() {
		lines++
	}
	if err := s.Err(); err != nil {
		return 0, err
	}
	return lines, nil
}

func checkCLIDocsCoverage(root string) []doctorIssue {
	issues := make([]doctorIssue, 0)
	cliRefPath := filepath.Join(root, "docs", "reference", "01-cli.md")
	if !hasFile(cliRefPath) {
		return issues
	}
	b, err := os.ReadFile(cliRefPath)
	if err != nil {
		return append(issues, doctorIssue{
			Code:    "DX012",
			Message: "failed to read docs/reference/01-cli.md",
			Fix:     err.Error(),
		})
	}
	text := string(b)
	requiredSections := []string{
		"## Minimal V1 Command Set",
		"## Implementation Mapping (Current Repo)",
		"## Generator test strategy",
	}
	for _, section := range requiredSections {
		if !strings.Contains(text, section) {
			issues = append(issues, doctorIssue{
				Code:    "DX012",
				Message: fmt.Sprintf("cli docs missing required section: %q", section),
				Fix:     "restore required sections in docs/reference/01-cli.md",
			})
		}
	}

	required := []string{
		"ship doctor",
		"ship new <app>",
		"ship upgrade",
		"ship make:resource",
		"ship make:model",
		"ship make:controller",
		"ship make:scaffold",
		"ship make:module",
		"ship db:migrate",
		"ship test --integration",
	}
	for _, token := range required {
		if !strings.Contains(text, token) {
			issues = append(issues, doctorIssue{
				Code:    "DX012",
				Message: fmt.Sprintf("cli docs missing required command token: %q", token),
				Fix:     "update docs/reference/01-cli.md to cover implemented core commands",
			})
		}
	}
	return issues
}

func checkGoWorkModules(root string) []doctorIssue {
	issues := make([]doctorIssue, 0)
	goWorkPath := filepath.Join(root, "go.work")
	if !hasFile(goWorkPath) {
		return issues
	}
	b, err := os.ReadFile(goWorkPath)
	if err != nil {
		return append(issues, doctorIssue{
			Code:    "DX014",
			Message: "failed to read go.work",
			Fix:     err.Error(),
		})
	}
	modules := parseGoWorkUseModules(string(b))
	for _, modPath := range modules {
		p := filepath.Clean(filepath.Join(root, filepath.FromSlash(modPath)))
		if !hasFile(filepath.Join(p, "go.mod")) {
			issues = append(issues, doctorIssue{
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

func checkDockerIgnoreCoverage(root string) []doctorIssue {
	issues := make([]doctorIssue, 0)
	if !hasFile(filepath.Join(root, "infra", "docker", "Dockerfile")) {
		return issues
	}
	path := filepath.Join(root, ".dockerignore")
	if !hasFile(path) {
		return append(issues, doctorIssue{
			Code:    "DX015",
			Message: "missing .dockerignore",
			Fix:     "add .dockerignore with heavy-path exclusions to keep docker build context small",
		})
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return append(issues, doctorIssue{
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
		"tmp",
		"tools/scripts/venv",
	}
	for _, entry := range requiredEntries {
		if !containsDockerIgnoreEntry(text, entry) {
			issues = append(issues, doctorIssue{
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

func checkDockerLocalReplaceOrder(root string) []doctorIssue {
	issues := make([]doctorIssue, 0)
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
		return append(issues, doctorIssue{
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
		return append(issues, doctorIssue{
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
			issues = append(issues, doctorIssue{
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
		filepath.Join(root, "apps", "go.mod"),
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
