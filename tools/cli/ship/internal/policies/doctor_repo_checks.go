package policies

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type workspaceKind string

const (
	workspaceKindUnknown             workspaceKind = "unknown"
	workspaceKindGeneratedApp        workspaceKind = "generated-app"
	workspaceKindGeneratedAPIOnlyApp workspaceKind = "generated-api-only-app"
	workspaceKindFrameworkRepo       workspaceKind = "framework-repo"
)

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
		"app",
		filepath.ToSlash(filepath.Join("app", "container.go")),
		filepath.ToSlash(filepath.Join("app", "router.go")),
		filepath.ToSlash(filepath.Join("app", "schedules.go")),
		"cmd",
		"config",
		"db",
		"docs",
		"framework",
		"frontend",
		"go.mod",
		"go.work",
		"infra",
		"locales",
		"modules",
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

	forbidden := []string{"javascript"}
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

func checkFrameworkCIVerifyGate(root string) []DoctorIssue {
	if !looksLikeCanonicalFrameworkRepo(root) {
		return nil
	}
	workflowRel := filepath.ToSlash(filepath.Join(".github", "workflows", "test.yml"))
	workflowPath := filepath.Join(root, filepath.FromSlash(workflowRel))
	if !hasFile(workflowPath) {
		return []DoctorIssue{{
			Code:    "DX013",
			File:    workflowRel,
			Message: "missing CI workflow gate for strict framework verify profile",
			Fix:     "add .github/workflows/test.yml with a verify_strict job that runs `go run ./tools/cli/ship/cmd/ship verify --profile strict`",
		}}
	}
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		return []DoctorIssue{{
			Code:    "DX013",
			File:    workflowRel,
			Message: "failed to read CI workflow for strict verify gate",
			Fix:     err.Error(),
		}}
	}
	text := string(content)
	var issues []DoctorIssue
	if !strings.Contains(text, "verify_strict:") {
		issues = append(issues, DoctorIssue{
			Code:    "DX013",
			File:    workflowRel,
			Message: "CI workflow missing verify_strict job",
			Fix:     "restore verify_strict job in .github/workflows/test.yml",
		})
	}
	if !strings.Contains(text, "go run ./tools/cli/ship/cmd/ship verify --profile strict") {
		issues = append(issues, DoctorIssue{
			Code:    "DX013",
			File:    workflowRel,
			Message: "CI workflow missing strict verify command",
			Fix:     "run `go run ./tools/cli/ship/cmd/ship verify --profile strict` in verify_strict job",
		})
	}
	return issues
}

func detectWorkspaceKind(root string) workspaceKind {
	if looksLikeCanonicalFrameworkRepo(root) {
		return workspaceKindFrameworkRepo
	}
	if looksLikeGeneratedAPIOnlyWorkspace(root) {
		return workspaceKindGeneratedAPIOnlyApp
	}
	if looksLikeGeneratedAppWorkspace(root) {
		return workspaceKindGeneratedApp
	}
	return workspaceKindUnknown
}

func checkGeneratedAppRequiredPaths(root string) []DoctorIssue {
	kind := detectWorkspaceKind(root)
	if kind != workspaceKindGeneratedApp && kind != workspaceKindGeneratedAPIOnlyApp {
		return nil
	}

	issues := make([]DoctorIssue, 0)
	requiredDirs := []string{
		filepath.Join("app"),
		filepath.Join("app", "foundation"),
		filepath.Join("app", "web", "controllers"),
		filepath.Join("app", "web", "middleware"),
		filepath.Join("app", "web", "ui"),
		filepath.Join("app", "web", "viewmodels"),
		filepath.Join("app", "jobs"),
		filepath.Join("db", "queries"),
		filepath.Join("db", "migrate", "migrations"),
	}
	if kind == workspaceKindGeneratedApp {
		requiredDirs = append(requiredDirs, filepath.Join("app", "views"))
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

	return issues
}

func looksLikeCanonicalFrameworkRepo(root string) bool {
	signals := []string{
		filepath.Join("tools", "cli", "ship"),
		filepath.Join("tools", "mcp", "ship"),
		"framework",
		"examples",
		"testdata",
		"go.work",
	}

	hits := 0
	for _, rel := range signals {
		if pathExists(filepath.Join(root, rel)) {
			hits++
		}
	}

	return hits >= 2
}

func looksLikeGeneratedAPIOnlyWorkspace(root string) bool {
	if !looksLikeGeneratedAppWorkspace(root) {
		return false
	}

	webMainPath := filepath.Join(root, "cmd", "web", "main.go")
	content, err := os.ReadFile(webMainPath)
	if err != nil {
		return false
	}

	text := string(content)
	return strings.Contains(text, "starter api web listening") &&
		strings.Contains(text, "writeJSON(w, http.StatusOK")
}
