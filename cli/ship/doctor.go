package ship

import (
	"fmt"
	"os"
	"path/filepath"
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
		filepath.Join("apps", "goship"),
		filepath.Join("apps", "goship", "app"),
		filepath.Join("apps", "goship", "foundation"),
		filepath.Join("apps", "goship", "web", "controllers"),
		filepath.Join("apps", "goship", "web", "middleware"),
		filepath.Join("apps", "goship", "web", "ui"),
		filepath.Join("apps", "goship", "web", "viewmodels"),
		filepath.Join("apps", "goship", "jobs"),
		filepath.Join("apps", "goship", "views"),
		filepath.Join("apps", "goship", "db", "schema"),
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
		filepath.Join("apps", "goship", "router.go"),
		filepath.Join("apps", "goship", "foundation", "container.go"),
		filepath.Join("apps", "goship", "web", "routenames", "routenames.go"),
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
		filepath.Join("app", "goship"),
		filepath.Join("apps", "goship", "bootstrap"),
		filepath.Join("apps", "goship", "domains"),
		filepath.Join("apps", "goship", "tasks"),
		filepath.Join("apps", "goship", "types"),
		filepath.Join("apps", "goship", "webui"),
		filepath.Join("apps", "goship", "middleware"),
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

	router := filepath.Join(root, "apps", "goship", "router.go")
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
						Fix:     "restore route markers in apps/goship/router.go to keep generator wiring deterministic",
					})
				}
			}
		}
	}

	issues = append(issues, checkPackageNaming(root, filepath.Join("apps", "goship", "web", "ui"), "ui")...)
	issues = append(issues, checkPackageNaming(root, filepath.Join("apps", "goship", "web", "viewmodels"), "viewmodels")...)

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

