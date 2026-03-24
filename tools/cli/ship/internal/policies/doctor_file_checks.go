package policies

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func checkFileSizes(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	hardCapAllowlist := map[string]struct{}{
		filepath.ToSlash(filepath.Join("tools", "cli", "ship", "internal", "policies", "doctor.go")):  {},
		filepath.ToSlash(filepath.Join("config", "config.go")):                                        {},
		filepath.ToSlash(filepath.Join("app", "views", "web", "pages", "home_feed.templ")):            {},
		filepath.ToSlash(filepath.Join("app", "views", "web", "pages", "landing_page.templ")):         {},
		filepath.ToSlash(filepath.Join("app", "views", "web", "pages", "preferences.templ")):          {},
		filepath.ToSlash(filepath.Join("app", "views", "emails", "password_reset.templ")):             {},
		filepath.ToSlash(filepath.Join("app", "views", "emails", "registration_confirmation.templ")): {},
		filepath.ToSlash(filepath.Join("app", "views", "emails", "update.templ")):                     {},
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
