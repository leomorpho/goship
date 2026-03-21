package policies

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

func checkFileSizes(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	hardCapAllowlist := map[string]struct{}{
		filepath.ToSlash(filepath.Join("tools", "cli", "ship", "internal", "policies", "doctor.go")):             {},
		filepath.ToSlash(filepath.Join("tools", "cli", "ship", "internal", "policies", "doctor_repo_checks.go")): {},
		filepath.ToSlash(filepath.Join("config", "config.go")):                                                   {},
		filepath.ToSlash(filepath.Join("app", "views", "web", "pages", "home_feed.templ")):                       {},
		filepath.ToSlash(filepath.Join("app", "views", "web", "pages", "landing_page.templ")):                    {},
		filepath.ToSlash(filepath.Join("app", "views", "web", "pages", "preferences.templ")):                     {},
		filepath.ToSlash(filepath.Join("app", "views", "emails", "password_reset.templ")):                        {},
		filepath.ToSlash(filepath.Join("app", "views", "emails", "registration_confirmation.templ")):             {},
		filepath.ToSlash(filepath.Join("app", "views", "emails", "update.templ")):                                {},
	}

	scanRoots := []string{
		filepath.Join(root, "app"),
		filepath.Join(root, "framework"),
		filepath.Join(root, "tools"),
		filepath.Join(root, "config"),
	}
	for _, scanRoot := range scanRoots {
		if !isDir(scanRoot) {
			continue
		}
		_ = filepath.WalkDir(scanRoot, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			rel := filepath.ToSlash(mustRel(root, path))

			if d.IsDir() {
				if rel == "vendor" ||
					strings.HasPrefix(rel, "vendor/") ||
					rel == ".git" ||
					rel == "node_modules" ||
					rel == ".cache" ||
					filepath.Base(rel) == ".cache" ||
					strings.Contains(rel, "/.cache/") ||
					strings.HasSuffix(rel, "/gen") {
					return filepath.SkipDir
				}
				return nil
			}

			kind, warnThreshold, errorThreshold, skip := doctorFileSizeKind(rel)
			if skip {
				return nil
			}

			lines, lineErr := countNonBlankLines(path)
			if lineErr != nil {
				issues = append(issues, DoctorIssue{
					Code:    "DX010",
					File:    rel,
					Message: fmt.Sprintf("failed counting non-blank lines for %s", rel),
					Fix:     lineErr.Error(),
				})
				return nil
			}
			if lines <= warnThreshold {
				return nil
			}

			severity := "warning"
			message := fmt.Sprintf("%s file exceeds recommended size (%d > %d non-blank lines): %s", kind, lines, warnThreshold, rel)
			if lines > errorThreshold {
				if _, ok := hardCapAllowlist[rel]; ok {
					message = fmt.Sprintf("%s file exceeds hard size cap but is grandfathered (%d > %d non-blank lines): %s", kind, lines, errorThreshold, rel)
				} else {
					severity = "error"
					message = fmt.Sprintf("%s file exceeds hard size cap (%d > %d non-blank lines): %s", kind, lines, errorThreshold, rel)
				}
			}

			issues = append(issues, DoctorIssue{
				Code:     "DX010",
				File:     rel,
				Message:  message,
				Fix:      "split by responsibility to keep files LLM-friendly",
				Severity: severity,
			})
			return nil
		})
	}

	return issues
}

func doctorFileSizeKind(rel string) (kind string, warnThreshold int, errorThreshold int, skip bool) {
	switch {
	case strings.HasSuffix(rel, ".go"):
		if strings.HasSuffix(rel, "_test.go") ||
			strings.HasSuffix(rel, ".templ.go") ||
			strings.HasSuffix(rel, "_sql.go") ||
			strings.HasPrefix(filepath.Base(rel), "bob_") {
			return "", 0, 0, true
		}
		return "Go", 800, 1000, false
	case strings.HasSuffix(rel, ".templ"):
		return "templ", 600, 800, false
	default:
		return "", 0, 0, true
	}
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
		".doombox":   {},
		".docket":    {},
		".git":       {},
		".github":    {},
		".githooks":  {},
		".kamal":     {},
		".local":     {},
		".worktrees": {},
		".vscode":    {},
		"app":        {},
		"db":         {},
		"cmd":        {},
		"config":     {},
		"docs":       {},
		"framework":  {},
		"infra":      {},
		"locales":    {},
		"modules":    {},
		"static":     {},
		"styles":     {},
		"tests":      {},
		"tools":      {},
		"frontend":   {},
		"tmp":        {},
		"uploads":    {},
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

func CheckCanonicalRepoTopLevelPaths(root string) []DoctorIssue {
	if !looksLikeCanonicalFrameworkRepo(root) {
		return nil
	}

	issues := make([]DoctorIssue, 0)
	required := []string{
		"cmd",
		"config",
		"container.go",
		"db",
		"docs",
		"framework",
		"frontend",
		"go.mod",
		"go.work",
		"infra",
		"locales",
		"modules",
		"router.go",
		"schedules.go",
		"static",
		"styles",
		"testdata",
		"tests",
		"tools",
	}
	for _, rel := range required {
		if !pathExists(filepath.Join(root, rel)) {
			issues = append(issues, DoctorIssue{
				Code:    "DX013",
				File:    filepath.ToSlash(rel),
				Message: fmt.Sprintf("missing canonical top-level path: %s", filepath.ToSlash(rel)),
				Fix:     "restore the canonical GoShip repo layout before merging",
			})
		}
	}

	forbidden := []string{
		"app",
		"javascript",
	}
	for _, rel := range forbidden {
		if !pathExists(filepath.Join(root, rel)) {
			continue
		}
		issues = append(issues, DoctorIssue{
			Code:    "DX013",
			File:    filepath.ToSlash(rel),
			Message: fmt.Sprintf("forbidden top-level path present: %s", filepath.ToSlash(rel)),
			Fix:     "remove the old competing top-level path and keep the canonical GoShip repo shape",
		})
	}

	return issues
}

func looksLikeCanonicalFrameworkRepo(root string) bool {
	return isDir(filepath.Join(root, "tools", "cli", "ship")) ||
		hasFile(filepath.Join(root, "container.go")) ||
		hasFile(filepath.Join(root, "router.go")) ||
		hasFile(filepath.Join(root, "schedules.go"))
}

func countNonBlankLines(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	lines := 0
	for s.Scan() {
		if strings.TrimSpace(s.Text()) == "" {
			continue
		}
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
		"ship verify",
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
	if looksLikeCanonicalFrameworkRepo(root) {
		for _, token := range []string{"extension-zone manifest", "`container.go`", "`router.go`", "`schedules.go`"} {
			if strings.Contains(text, token) {
				continue
			}
			issues = append(issues, DoctorIssue{
				Code:    "DX012",
				Message: fmt.Sprintf("cli docs missing framework repo command token: %q", token),
				Fix:     "update docs/reference/01-cli.md to describe canonical root-runtime repo enforcement",
			})
		}
	}
	return issues
}

func checkExtensionZoneManifest(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	rel := filepath.ToSlash(filepath.Join("docs", "architecture", "10-extension-zones.md"))
	path := filepath.Join(root, filepath.FromSlash(rel))
	if !hasFile(path) {
		return append(issues, DoctorIssue{
			Code:    "DX031",
			File:    rel,
			Message: "missing extension-zone manifest",
			Fix:     "add docs/architecture/10-extension-zones.md with extension and protected contract zone definitions",
		})
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return append(issues, DoctorIssue{
			Code:    "DX031",
			File:    rel,
			Message: "failed to read extension-zone manifest",
			Fix:     err.Error(),
		})
	}

	text := string(b)
	requiredTokens := []string{
		"## Extension Zones",
		"## Protected Contract Zones",
		"`framework/`",
		"`config/modules.yaml`",
		"`tools/agent-policy/allowed-commands.yaml`",
	}
	if looksLikeCanonicalFrameworkRepo(root) {
		requiredTokens = append(requiredTokens,
			"`container.go`",
			"`router.go`",
			"`schedules.go`",
		)
	} else {
		requiredTokens = append(requiredTokens,
			"`app/`",
			"`app/router.go`",
			"`app/foundation/container.go`",
		)
	}
	for _, token := range requiredTokens {
		if strings.Contains(text, token) {
			continue
		}
		issues = append(issues, DoctorIssue{
			Code:    "DX031",
			File:    rel,
			Message: fmt.Sprintf("extension-zone manifest missing required token: %q", token),
			Fix:     "restore the canonical extension-zone manifest sections and protected seam entries",
		})
	}

	if looksLikeCanonicalFrameworkRepo(root) {
		staleTokens := []string{
			"`app/`",
			"`app/router.go`",
			"`app/foundation/container.go`",
		}
		for _, token := range staleTokens {
			if !strings.Contains(text, token) {
				continue
			}
			issues = append(issues, DoctorIssue{
				Code:    "DX031",
				File:    rel,
				Message: fmt.Sprintf("extension-zone manifest contains stale framework-shell token: %q", token),
				Fix:     "document the canonical root runtime seams and remove deleted internal app-shell references",
			})
		}
	}

	return issues
}

func checkCanonicalDocsHardReset(root string) []DoctorIssue {
	return CheckHardCutDocWording(root)
}

type hardCutPhraseRule struct {
	Phrase  string
	Replace string
}

type hardCutAllowlistEntry struct {
	Path   string
	Phrase string
}

var hardCutCanonicalDocs = []string{
	filepath.ToSlash(filepath.Join("docs", "architecture", "01-architecture.md")),
	filepath.ToSlash(filepath.Join("docs", "architecture", "02-structure-and-boundaries.md")),
	filepath.ToSlash(filepath.Join("docs", "architecture", "03-project-scope-analysis.md")),
	filepath.ToSlash(filepath.Join("docs", "architecture", "04-http-routes.md")),
	filepath.ToSlash(filepath.Join("docs", "architecture", "05-data-model.md")),
	filepath.ToSlash(filepath.Join("docs", "architecture", "06-known-gaps-and-risks.md")),
	filepath.ToSlash(filepath.Join("docs", "architecture", "07-core-interfaces.md")),
	filepath.ToSlash(filepath.Join("docs", "architecture", "08-cognitive-model.md")),
	filepath.ToSlash(filepath.Join("docs", "architecture", "09-standalone-and-managed-mode.md")),
	filepath.ToSlash(filepath.Join("docs", "reference", "01-cli.md")),
	filepath.ToSlash(filepath.Join("docs", "roadmap", "01-framework-plan.md")),
}

var hardCutPhraseRules = []hardCutPhraseRule{
	{
		Phrase:  "active transitional state",
		Replace: "describe the current hard-cut model directly",
	},
	{
		Phrase:  "transition-era",
		Replace: "describe the current hard-cut model directly",
	},
	{
		Phrase:  "legacy compatibility path",
		Replace: "describe the single canonical command/runtime path",
	},
	{
		Phrase:  "compatibility window",
		Replace: "document upgrade guidance or migration notes instead of a dual-path window",
	},
	{
		Phrase:  "deprecation period",
		Replace: "describe the current canonical behavior without transition language",
	},
	{
		Phrase:  "deprecated alias",
		Replace: "remove alias wording and document the canonical command only",
	},
}

func CheckHardCutDocWording(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	docsRoot := filepath.Join(root, "docs")
	if !isDir(docsRoot) {
		return issues
	}

	allowlist, allowlistIssues := readHardCutAllowlist(root)
	issues = append(issues, allowlistIssues...)

	_ = filepath.WalkDir(docsRoot, func(currentPath string, d os.DirEntry, err error) error {
		if err != nil {
			rel := filepath.ToSlash(strings.TrimPrefix(currentPath, root+string(filepath.Separator)))
			issues = append(issues, DoctorIssue{
				Code:    "DX030",
				File:    rel,
				Message: fmt.Sprintf("failed to walk docs path %s", rel),
				Fix:     err.Error(),
			})
			return nil
		}
		if d.IsDir() || filepath.Ext(currentPath) != ".md" {
			return nil
		}
		rel, relErr := filepath.Rel(root, currentPath)
		if relErr != nil {
			issues = append(issues, DoctorIssue{
				Code:    "DX030",
				Message: "failed to resolve docs path for hard-cut wording check",
				Fix:     relErr.Error(),
			})
			return nil
		}
		issues = append(issues, scanHardCutDocFile(root, filepath.ToSlash(rel), allowlist)...)
		return nil
	})

	return issues
}

func scanHardCutDocFile(root, rel string, allowlist []hardCutAllowlistEntry) []DoctorIssue {
	path := filepath.Join(root, filepath.FromSlash(rel))
	content, err := os.ReadFile(path)
	if err != nil {
		return []DoctorIssue{{
			Code:    "DX030",
			File:    rel,
			Message: fmt.Sprintf("failed to read docs file %s", rel),
			Fix:     err.Error(),
		}}
	}

	isCanonical := false
	for _, canonical := range hardCutCanonicalDocs {
		if canonical == rel {
			isCanonical = true
			break
		}
	}

	lines := strings.Split(string(content), "\n")
	issues := make([]DoctorIssue, 0)
	for i, raw := range lines {
		line := strings.ToLower(raw)
		for _, rule := range hardCutPhraseRules {
			if !strings.Contains(line, rule.Phrase) {
				continue
			}
			if !isCanonical && allowHardCutPhrase(rel, rule.Phrase, allowlist) {
				continue
			}
			location := fmt.Sprintf("%s:%d", rel, i+1)
			issues = append(issues, DoctorIssue{
				Code:    "DX030",
				File:    location,
				Message: fmt.Sprintf("%s contains forbidden compatibility/deprecation wording %q", location, rule.Phrase),
				Fix:     fmt.Sprintf("%s; %s", rule.Replace, hardCutAllowlistFix(rel, rule.Phrase, isCanonical)),
			})
		}
	}
	return issues
}

func hardCutAllowlistFix(rel, phrase string, isCanonical bool) string {
	if isCanonical {
		return "rewrite canonical docs to describe the current hard-cut model only"
	}
	return fmt.Sprintf("if this is an intentional historical reference outside the canonical docs, add %q to docs/policies/02-transition-wording-allowlist.txt", rel+"|"+phrase)
}

func allowHardCutPhrase(rel, phrase string, allowlist []hardCutAllowlistEntry) bool {
	for _, entry := range allowlist {
		if path.Clean(entry.Path) == path.Clean(rel) && strings.EqualFold(entry.Phrase, phrase) {
			return true
		}
	}
	return false
}

func readHardCutAllowlist(root string) ([]hardCutAllowlistEntry, []DoctorIssue) {
	rel := filepath.ToSlash(filepath.Join("docs", "policies", "02-transition-wording-allowlist.txt"))
	path := filepath.Join(root, filepath.FromSlash(rel))
	if !hasFile(path) {
		return nil, nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, []DoctorIssue{{
			Code:    "DX030",
			File:    rel,
			Message: "failed to read hard-cut wording allowlist",
			Fix:     err.Error(),
		}}
	}

	entries := make([]hardCutAllowlistEntry, 0)
	issues := make([]DoctorIssue, 0)
	for idx, raw := range strings.Split(string(content), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		pathPart, phrasePart, ok := strings.Cut(line, "|")
		if !ok || strings.TrimSpace(pathPart) == "" || strings.TrimSpace(phrasePart) == "" {
			issues = append(issues, DoctorIssue{
				Code:    "DX030",
				File:    fmt.Sprintf("%s:%d", rel, idx+1),
				Message: "invalid hard-cut wording allowlist entry",
				Fix:     `use "docs/path.md|phrase" entries`,
			})
			continue
		}
		entries = append(entries, hardCutAllowlistEntry{
			Path:   filepath.ToSlash(strings.TrimSpace(pathPart)),
			Phrase: strings.ToLower(strings.TrimSpace(phrasePart)),
		})
	}
	return entries, issues
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

func defaultDoctorRunCmd(dir string, name string, args ...string) (int, string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), string(out), nil
		}
		return 1, string(out), err
	}
	return 0, string(out), nil
}

func printDoctorHelp(w io.Writer) {
	fmt.Fprintln(w, "ship doctor commands:")
	fmt.Fprintln(w, "  ship doctor [--json]")
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
