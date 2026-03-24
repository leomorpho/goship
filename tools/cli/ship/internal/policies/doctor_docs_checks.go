package policies

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

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
		for _, stale := range []string{"`app/foundation/container.go`", "`app/schedules/schedules.go`"} {
			if !strings.Contains(text, stale) {
				continue
			}
			issues = append(issues, DoctorIssue{
				Code:    "DX012",
				Message: fmt.Sprintf("cli docs contain stale framework-shell link token: %q", stale),
				Fix:     "remove deleted internal app-shell file links and keep canonical root runtime seams in docs/reference/01-cli.md",
			})
		}
	}
	return issues
}

func checkExtensionZoneManifest(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	rel := filepath.ToSlash(filepath.Join("docs", "architecture", "10-extension-zones.md"))
	p := filepath.Join(root, filepath.FromSlash(rel))
	if !hasFile(p) {
		return append(issues, DoctorIssue{
			Code:    "DX031",
			File:    rel,
			Message: "missing extension-zone manifest",
			Fix:     "add docs/architecture/10-extension-zones.md with extension and protected contract zone definitions",
		})
	}

	b, err := os.ReadFile(p)
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
	p := filepath.Join(root, filepath.FromSlash(rel))
	content, err := os.ReadFile(p)
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
	p := filepath.Join(root, filepath.FromSlash(rel))
	if !hasFile(p) {
		return nil, nil
	}

	content, err := os.ReadFile(p)
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
