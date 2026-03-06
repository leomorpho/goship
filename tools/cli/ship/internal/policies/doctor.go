package policies

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	rt "github.com/leomorpho/goship/tools/cli/ship/internal/runtime"
)

type DoctorIssue struct {
	Code    string
	Message string
	Fix     string
}

type DoctorDeps struct {
	Out          io.Writer
	Err          io.Writer
	FindGoModule func(start string) (string, string, error)
}

func RunDoctor(args []string, d DoctorDeps) int {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			printDoctorHelp(d.Out)
			return 0
		}
	}
	if len(args) > 0 {
		fmt.Fprintf(d.Err, "unexpected doctor arguments: %v\n", args)
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

	issues := RunDoctorChecks(root)
	if len(issues) == 0 {
		fmt.Fprintf(d.Out, "ship doctor: OK (%s)\n", root)
		return 0
	}

	fmt.Fprintf(d.Err, "ship doctor: found %d issue(s)\n", len(issues))
	for _, issue := range issues {
		fmt.Fprintf(d.Err, "- [%s] %s\n", issue.Code, issue.Message)
		if issue.Fix != "" {
			fmt.Fprintf(d.Err, "  fix: %s\n", issue.Fix)
		}
	}
	return 1
}

func RunDoctorChecks(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)

	requiredDirs := []string{
		filepath.Join("app"),
		filepath.Join("app", "foundation"),
		filepath.Join("app", "web", "controllers"),
		filepath.Join("app", "web", "middleware"),
		filepath.Join("app", "web", "ui"),
		filepath.Join("app", "web", "viewmodels"),
		filepath.Join("app", "jobs"),
		filepath.Join("app", "views"),
		filepath.Join("db", "queries"),
		filepath.Join("db", "migrate", "migrations"),
	}
	for _, rel := range requiredDirs {
		if !isDir(filepath.Join(root, rel)) {
			issues = append(issues, DoctorIssue{
				Code:    "DX001",
				Message: fmt.Sprintf("missing required directory: %s", rel),
				Fix:     fmt.Sprintf("create %s or regenerate the app scaffold with `ship new`", rel),
			})
		}
	}

	requiredFiles := []string{
		filepath.Join("app", "router.go"),
		filepath.Join("app", "foundation", "container.go"),
		filepath.Join("app", "web", "routenames", "routenames.go"),
		filepath.Join("db", "bobgen.yaml"),
		filepath.Join("config", "modules.yaml"),
		filepath.Join("docs", "00-index.md"),
		filepath.Join("docs", "architecture", "01-architecture.md"),
		filepath.Join("docs", "architecture", "08-cognitive-model.md"),
	}
	for _, rel := range requiredFiles {
		if !hasFile(filepath.Join(root, rel)) {
			issues = append(issues, DoctorIssue{
				Code:    "DX002",
				Message: fmt.Sprintf("missing required file: %s", rel),
				Fix:     "restore missing documentation or scaffold files",
			})
		}
	}

	forbidden := []string{
		filepath.Join("app", "site"),
		filepath.Join("app", "bootstrap"),
		filepath.Join("app", "domains"),
		filepath.Join("app", "tasks"),
		filepath.Join("app", "types"),
		filepath.Join("app", "webui"),
		filepath.Join("app", "middleware"),
	}
	for _, rel := range forbidden {
		if pathExists(filepath.Join(root, rel)) {
			issues = append(issues, DoctorIssue{
				Code:    "DX003",
				Message: fmt.Sprintf("forbidden legacy path present: %s", rel),
				Fix:     "remove or migrate legacy paths to canonical app layout",
			})
		}
	}

	rootBinaries := []string{"web", "worker", "seed", "ship", "ship-mcp"}
	for _, name := range rootBinaries {
		if hasFile(filepath.Join(root, name)) {
			issues = append(issues, DoctorIssue{
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
			issues = append(issues, DoctorIssue{
				Code:    "DX009",
				Message: "failed to read .gitignore",
				Fix:     err.Error(),
			})
		} else {
			ignoreText := string(content)
			required := []string{"/web", "/worker", "/seed", "/ship", "/ship-mcp"}
			for _, entry := range required {
				if !strings.Contains(ignoreText, entry) {
					issues = append(issues, DoctorIssue{
						Code:    "DX009",
						Message: fmt.Sprintf(".gitignore missing required artifact entry: %s", entry),
						Fix:     "add required root binary ignore entries to .gitignore",
					})
				}
			}
		}
	}

	router := filepath.Join(root, "app", "router.go")
	if hasFile(router) {
		b, err := os.ReadFile(router)
		if err != nil {
			issues = append(issues, DoctorIssue{
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
					issues = append(issues, DoctorIssue{
						Code:    "DX005",
						Message: fmt.Sprintf("missing router marker: %s", marker),
						Fix:     "restore route markers in app/router.go to keep generator wiring deterministic",
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
					issues = append(issues, DoctorIssue{
						Code:    "DX011",
						Message: fmt.Sprintf("router marker order invalid: %s appears after %s", pair.start, pair.end),
						Fix:     "place start marker before end marker to keep --wire deterministic",
					})
				}
			}
		}
	}

	issues = append(issues, checkPackageNaming(root, filepath.Join("app", "web", "ui"), "ui")...)
	issues = append(issues, checkPackageNaming(root, filepath.Join("app", "web", "viewmodels"), "viewmodels")...)
	issues = append(issues, checkTopLevelDirs(root)...)
	issues = append(issues, checkFileLengthBudget(root, 500)...)
	issues = append(issues, checkCLIDocsCoverage(root)...)
	issues = append(issues, checkGoWorkModules(root)...)
	issues = append(issues, checkDockerIgnoreCoverage(root)...)
	issues = append(issues, checkDockerLocalReplaceOrder(root)...)
	issues = append(issues, checkAgentPolicyArtifacts(root)...)
	issues = append(issues, checkModulesManifestFormat(root)...)
	issues = append(issues, checkEnabledModuleDBArtifacts(root)...)
	issues = append(issues, checkForbiddenCrossBoundaryImports(root)...)

	return issues
}

func checkModulesManifestFormat(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	path := filepath.Join(root, "config", "modules.yaml")
	if !hasFile(path) {
		return issues
	}
	_, err := rt.LoadModulesManifest(path)
	if err != nil {
		issues = append(issues, DoctorIssue{
			Code:    "DX018",
			Message: "invalid config/modules.yaml format",
			Fix:     fmt.Sprintf("use YAML shape `modules: []` with tokens [a-z0-9_-]: %v", err),
		})
	}
	return issues
}

func checkEnabledModuleDBArtifacts(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	path := filepath.Join(root, "config", "modules.yaml")
	if !hasFile(path) {
		return issues
	}
	manifest, err := rt.LoadModulesManifest(path)
	if err != nil {
		return issues
	}

	for _, name := range manifest.Modules {
		moduleRoot := filepath.Join(root, "modules", name)
		if !isDir(moduleRoot) {
			issues = append(issues, DoctorIssue{
				Code:    "DX019",
				Message: fmt.Sprintf("enabled module directory missing: modules/%s", name),
				Fix:     fmt.Sprintf("add modules/%s or remove %q from config/modules.yaml", name, name),
			})
			continue
		}

		migrationsDir := filepath.Join(moduleRoot, "db", "migrate", "migrations")
		if !isDir(migrationsDir) {
			issues = append(issues, DoctorIssue{
				Code:    "DX019",
				Message: fmt.Sprintf("enabled module missing migrations directory: modules/%s/db/migrate/migrations", name),
				Fix:     fmt.Sprintf("add module migrations under modules/%s/db/migrate/migrations", name),
			})
		}

		bobgenPath := filepath.Join(moduleRoot, "db", "bobgen.yaml")
		if !hasFile(bobgenPath) {
			issues = append(issues, DoctorIssue{
				Code:    "DX019",
				Message: fmt.Sprintf("enabled module missing bobgen config: modules/%s/db/bobgen.yaml", name),
				Fix:     fmt.Sprintf("add modules/%s/db/bobgen.yaml", name),
			})
		}
	}

	return issues
}

func checkForbiddenCrossBoundaryImports(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)

	// Controllers must not directly import Ent packages.
	controllerDir := filepath.Join(root, "app", "web", "controllers")
	issues = append(issues, checkImportPrefixForbidden(controllerDir, "github.com/leomorpho/goship/db/ent", "DX020",
		"controller db boundary violated: app/web/controllers must not import db/ent directly",
		"move DB access behind foundation/service seams or auth/profile helpers")...)

	// Controllers must not call QueryProfile() directly.
	issues = append(issues, checkTextForbiddenInDir(controllerDir, "QueryProfile(", "DX020",
		"controller auth boundary violated: direct QueryProfile() usage is not allowed in app/web/controllers",
		"use middleware auth identity keys + service/store lookup by id")...)

	// Jobs SQL path must stay Ent-free.
	issues = append(issues, checkTextForbidden(filepath.Join(root, "modules", "jobs", "config.go"), "EntClient", "DX020",
		"jobs SQL boundary violated: EntClient found in modules/jobs/config.go",
		"keep jobs SQL path DB-first (*sql.DB) and adapter-agnostic")...)
	issues = append(issues, checkTextForbidden(filepath.Join(root, "modules", "jobs", "module.go"), "EntClient", "DX020",
		"jobs SQL boundary violated: EntClient found in modules/jobs/module.go",
		"remove Ent coupling from jobs module runtime config")...)
	issues = append(issues, checkTextForbidden(filepath.Join(root, "modules", "jobs", "drivers", "sql", "client.go"), "EntClient", "DX020",
		"jobs SQL boundary violated: EntClient found in modules/jobs/drivers/sql/client.go",
		"keep jobs SQL driver independent from Ent")...)
	for _, path := range []string{
		filepath.Join(root, "modules", "jobs", "config.go"),
		filepath.Join(root, "modules", "jobs", "module.go"),
		filepath.Join(root, "modules", "jobs", "drivers", "sql", "client.go"),
	} {
		issues = append(issues, checkTextForbidden(path, "github.com/leomorpho/goship/db/ent", "DX020",
			fmt.Sprintf("jobs SQL boundary violated: db/ent import found in %s", filepath.ToSlash(mustRel(root, path))),
			"remove Ent imports from jobs SQL path")...)
	}

	// Notifications module must not depend on framework/core directly for pubsub contracts.
	for _, path := range []string{
		filepath.Join(root, "modules", "notifications", "module.go"),
		filepath.Join(root, "modules", "notifications", "notifier.go"),
		filepath.Join(root, "modules", "notifications", "notifier_test.go"),
	} {
		issues = append(issues, checkTextForbidden(path, "github.com/leomorpho/goship/framework/core", "DX020",
			fmt.Sprintf("notifications pubsub boundary violated: framework/core import found in %s", filepath.ToSlash(mustRel(root, path))),
			"use module-local contracts and app-level bridge adapters")...)
	}

	// Module isolation: no direct imports from root app/framework packages, except explicit allowlist paths.
	issues = append(issues, checkModuleSourceIsolation(root)...)

	return issues
}

func checkModuleSourceIsolation(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	modulesRoot := filepath.Join(root, "modules")
	if !isDir(modulesRoot) {
		return issues
	}
	allowlist := loadModuleIsolationAllowlist(filepath.Join(root, "tools", "scripts", "test", "module-isolation-allowlist.txt"))
	_ = filepath.WalkDir(modulesRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel := filepath.ToSlash(mustRel(root, path))
		if _, ok := allowlist[rel]; ok {
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			issues = append(issues, DoctorIssue{
				Code:    "DX020",
				Message: fmt.Sprintf("failed reading file for module isolation check: %s", rel),
				Fix:     readErr.Error(),
			})
			return nil
		}
		if strings.Contains(string(b), "\"github.com/leomorpho/goship/") {
			issues = append(issues, DoctorIssue{
				Code:    "DX020",
				Message: fmt.Sprintf("module isolation violated: forbidden root import in %s", rel),
				Fix:     "remove direct github.com/leomorpho/goship/* imports from module runtime code or add a deliberate allowlist entry",
			})
		}
		return nil
	})
	return issues
}

func loadModuleIsolationAllowlist(path string) map[string]struct{} {
	result := map[string]struct{}{}
	if !hasFile(path) {
		return result
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return result
	}
	for _, raw := range strings.Split(string(b), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		result[filepath.ToSlash(line)] = struct{}{}
	}
	return result
}

func checkImportPrefixForbidden(dir string, forbiddenPrefix string, code string, message string, fix string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	if !isDir(dir) {
		return issues
	}
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		if strings.Contains(string(b), "\""+forbiddenPrefix) {
			issues = append(issues, DoctorIssue{
				Code:    code,
				Message: message,
				Fix:     fix,
			})
		}
		return nil
	})
	return issues
}

func checkTextForbidden(path string, token string, code string, message string, fix string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	if !hasFile(path) {
		return issues
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return append(issues, DoctorIssue{
			Code:    code,
			Message: fmt.Sprintf("failed to read boundary file: %s", filepath.ToSlash(path)),
			Fix:     err.Error(),
		})
	}
	if strings.Contains(string(b), token) {
		issues = append(issues, DoctorIssue{
			Code:    code,
			Message: message,
			Fix:     fix,
		})
	}
	return issues
}

func checkTextForbiddenInDir(dir string, token string, code string, message string, fix string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	if !isDir(dir) {
		return issues
	}
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		if strings.Contains(string(b), token) {
			issues = append(issues, DoctorIssue{
				Code:    code,
				Message: message,
				Fix:     fix,
			})
		}
		return nil
	})
	return issues
}

func mustRel(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}

func checkPackageNaming(root, relDir, expected string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	dir := filepath.Join(root, relDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return issues
		}
		return append(issues, DoctorIssue{
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
			issues = append(issues, DoctorIssue{
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
			issues = append(issues, DoctorIssue{
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

func hasFile(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func checkFileLengthBudget(root string, maxLines int) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	allowlist := map[string]struct{}{
		filepath.ToSlash(filepath.Join("tools", "cli", "ship", "internal", "cli", "cli_test.go")):             {},
		filepath.ToSlash(filepath.Join("tools", "cli", "ship", "internal", "generators", "resource.go")):      {},
		filepath.ToSlash(filepath.Join("tools", "cli", "ship", "internal", "policies", "doctor.go")):          {},
		filepath.ToSlash(filepath.Join("app", "profiles", "repo.go")):                                         {},
		filepath.ToSlash(filepath.Join("tools", "cli", "ship", "internal", "commands", "project_new.go")):     {},
		filepath.ToSlash(filepath.Join("tools", "cli", "ship", "internal", "commands", "project_upgrade.go")): {},
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
			if rel == ".git" ||
				rel == "node_modules" ||
				rel == ".cache" ||
				filepath.Base(rel) == ".cache" ||
				strings.Contains(rel, "/.cache/") ||
				rel == filepath.ToSlash(filepath.Join("db", "ent")) ||
				strings.HasPrefix(rel, filepath.ToSlash(filepath.Join("db", "ent"))+"/") {
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
			issues = append(issues, DoctorIssue{
				Code:    "DX010",
				Message: fmt.Sprintf("failed counting lines for %s", rel),
				Fix:     lineErr.Error(),
			})
			return nil
		}
		if lines > maxLines {
			issues = append(issues, DoctorIssue{
				Code:    "DX010",
				Message: fmt.Sprintf("file exceeds line budget (%d > %d): %s", lines, maxLines, rel),
				Fix:     "split by responsibility to keep files LLM-friendly",
			})
		}
		return nil
	})

	return issues
}

func checkTopLevelDirs(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	entries, err := os.ReadDir(root)
	if err != nil {
		return append(issues, DoctorIssue{
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
		"app":        {},
		"db":         {},
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
			issues = append(issues, DoctorIssue{
				Code:    "DX013",
				Message: fmt.Sprintf("unexpected top-level directory: %s", name),
				Fix:     "move it under app/, db/, cmd/, modules/, framework/, tools/, infra/, tests/, or mark as intentional in doctor allow-list",
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

func checkCLIDocsCoverage(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	cliRefPath := filepath.Join(root, "docs", "reference", "01-cli.md")
	if !hasFile(cliRefPath) {
		return issues
	}
	b, err := os.ReadFile(cliRefPath)
	if err != nil {
		return append(issues, DoctorIssue{
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
			issues = append(issues, DoctorIssue{
				Code:    "DX012",
				Message: fmt.Sprintf("cli docs missing required section: %q", section),
				Fix:     "restore required sections in docs/reference/01-cli.md",
			})
		}
	}

	required := []string{
		"ship doctor",
		"ship agent:setup",
		"ship agent:check",
		"ship agent:status",
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
			issues = append(issues, DoctorIssue{
				Code:    "DX012",
				Message: fmt.Sprintf("cli docs missing required command token: %q", token),
				Fix:     "update docs/reference/01-cli.md to cover implemented core commands",
			})
		}
	}
	return issues
}

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
	path := filepath.Join(root, ".dockerignore")
	if !hasFile(path) {
		return append(issues, DoctorIssue{
			Code:    "DX015",
			Message: "missing .dockerignore",
			Fix:     "add .dockerignore with heavy-path exclusions to keep docker build context small",
		})
	}
	b, err := os.ReadFile(path)
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
		"tmp",
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

func checkAgentPolicyArtifacts(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	policyPath := filepath.Join(root, AgentPolicyFilePath)
	if !hasFile(policyPath) {
		return append(issues, DoctorIssue{
			Code:    "DX017",
			Message: fmt.Sprintf("missing agent policy file: %s", filepath.ToSlash(AgentPolicyFilePath)),
			Fix:     "add tools/agent-policy/allowed-commands.yaml and run ship agent:setup",
		})
	}
	policy, err := LoadPolicy(policyPath)
	if err != nil {
		return append(issues, DoctorIssue{
			Code:    "DX017",
			Message: "invalid agent policy file",
			Fix:     err.Error(),
		})
	}
	expected, err := RenderPolicyArtifacts(policy)
	if err != nil {
		return append(issues, DoctorIssue{
			Code:    "DX017",
			Message: "failed to render agent policy artifacts",
			Fix:     err.Error(),
		})
	}
	drifted, err := DiffPolicyArtifacts(root, expected)
	if err != nil {
		return append(issues, DoctorIssue{
			Code:    "DX017",
			Message: "failed to compare generated agent artifacts",
			Fix:     err.Error(),
		})
	}
	for _, rel := range drifted {
		issues = append(issues, DoctorIssue{
			Code:    "DX017",
			Message: fmt.Sprintf("agent artifact out of sync: %s", rel),
			Fix:     "run ship agent:setup",
		})
	}
	return issues
}

func printDoctorHelp(w io.Writer) {
	fmt.Fprintln(w, "ship doctor commands:")
	fmt.Fprintln(w, "  ship doctor")
	fmt.Fprintln(w, "  (validates canonical app structure and LLM/DX conventions)")
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
