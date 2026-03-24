package commands

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type DestroyDeps struct {
	Out io.Writer
	Err io.Writer
	Cwd string
}

func RunDestroy(args []string, d DestroyDeps) int {
	if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
		fmt.Fprintln(d.Err, "usage: ship destroy resource:<name>")
		return 1
	}

	kind, rawName, ok := strings.Cut(strings.TrimSpace(args[0]), ":")
	if !ok || strings.TrimSpace(kind) == "" || strings.TrimSpace(rawName) == "" {
		fmt.Fprintln(d.Err, "invalid artifact format: expected <kind>:<name> (example: resource:contact)")
		return 1
	}

	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "resource":
		return runDestroyResource(strings.TrimSpace(rawName), d)
	default:
		fmt.Fprintf(d.Err, "unsupported destroy artifact kind %q; supported kinds: resource\n", kind)
		return 1
	}
}

func runDestroyResource(name string, d DestroyDeps) int {
	norm, err := normalizeDestroyResourceName(name)
	if err != nil {
		fmt.Fprintf(d.Err, "invalid resource name %q: %v\n", name, err)
		return 1
	}

	cwd := strings.TrimSpace(d.Cwd)
	if cwd == "" {
		cwd, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(d.Err, "failed to resolve working directory: %v\n", err)
			return 1
		}
	}

	actions := []destroyAction{
		{
			Target: filepath.ToSlash(filepath.Join("app", "router.go")),
			Apply: func(absPath string) (string, error) {
				content, readErr := os.ReadFile(absPath)
				if readErr != nil {
					if os.IsNotExist(readErr) {
						return "skipped (path not found)", nil
					}
					return "", readErr
				}
				updated, removed := removeGeneratedRouteBlock(string(content), norm.Snake)
				if !removed {
					return "skipped (no generator route marker found)", nil
				}
				if err := os.WriteFile(absPath, []byte(updated), 0o644); err != nil {
					return "", err
				}
				return "removed generated route registration", nil
			},
		},
		{
			Target: filepath.ToSlash(filepath.Join("app", "web", "routenames", "routenames.go")),
			Apply: func(absPath string) (string, error) {
				content, readErr := os.ReadFile(absPath)
				if readErr != nil {
					if os.IsNotExist(readErr) {
						return "skipped (path not found)", nil
					}
					return "", readErr
				}
				updated, removed := removeRouteNameConstant(string(content), "RouteName"+norm.Pascal, norm.Snake)
				if !removed {
					return "skipped (no generator-managed route constant found)", nil
				}
				if err := os.WriteFile(absPath, []byte(updated), 0o644); err != nil {
					return "", err
				}
				return "removed generated route name constant", nil
			},
		},
		{
			Target: filepath.ToSlash(filepath.Join("app", "views", "web", "pages", norm.Snake+".templ")),
			Apply: func(absPath string) (string, error) {
				return removeManagedFile(absPath, []string{
					"templ " + norm.Pascal + "Page(",
					"TODO: implement " + norm.Kebab + " page.",
				})
			},
		},
		{
			Target: filepath.ToSlash(filepath.Join("app", "web", "controllers", norm.Snake+"_test.go")),
			Apply: func(absPath string) (string, error) {
				return removeManagedFile(absPath, []string{
					"func Test" + norm.Pascal + "Route_Get",
					"SCAFFOLD: implement " + norm.Pascal + " show",
				})
			},
		},
		{
			Target: filepath.ToSlash(filepath.Join("app", "web", "controllers", norm.Snake+".go")),
			Apply: func(absPath string) (string, error) {
				return removeManagedFile(absPath, []string{
					"type " + norm.LowerCamel + " struct {",
					"func New" + norm.Pascal + "Route(",
				})
			},
		},
	}

	hadMutation := false
	for _, action := range actions {
		abs := filepath.Join(cwd, filepath.FromSlash(action.Target))
		status, actionErr := action.Apply(abs)
		if actionErr != nil {
			fmt.Fprintf(d.Err, "destroy failed for %s: %v\n", action.Target, actionErr)
			return 1
		}
		fmt.Fprintf(d.Out, "%s: %s\n", action.Target, status)
		if strings.HasPrefix(status, "removed ") || status == "deleted file" {
			hadMutation = true
		}
	}

	if !hadMutation {
		fmt.Fprintf(d.Err, "refusing to destroy %q: no generator-managed targets matched\n", name)
		return 1
	}

	return 0
}

type destroyResourceName struct {
	Snake      string
	Kebab      string
	Pascal     string
	LowerCamel string
}

func normalizeDestroyResourceName(raw string) (destroyResourceName, error) {
	var out destroyResourceName
	tokens := tokenizeDestroyName(raw)
	if len(tokens) == 0 {
		return out, errors.New("resource name must contain at least one letter or number")
	}

	out.Snake = strings.Join(tokens, "_")
	out.Kebab = strings.Join(tokens, "-")

	pascalParts := make([]string, 0, len(tokens))
	for _, token := range tokens {
		pascalParts = append(pascalParts, strings.ToUpper(token[:1])+token[1:])
	}
	out.Pascal = strings.Join(pascalParts, "")
	out.LowerCamel = strings.ToLower(out.Pascal[:1]) + out.Pascal[1:]
	return out, nil
}

func tokenizeDestroyName(raw string) []string {
	var tokens []string
	var current []rune
	runes := []rune(strings.TrimSpace(raw))

	flush := func() {
		if len(current) == 0 {
			return
		}
		tokens = append(tokens, strings.ToLower(string(current)))
		current = current[:0]
	}

	for i, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush()
			continue
		}

		if unicode.IsUpper(r) && len(current) > 0 {
			prev := runes[i-1]
			var next rune
			if i+1 < len(runes) {
				next = runes[i+1]
			}
			if unicode.IsLower(prev) || (unicode.IsUpper(prev) && next != 0 && unicode.IsLower(next)) || unicode.IsDigit(prev) {
				flush()
			}
		}

		current = append(current, unicode.ToLower(r))
	}
	flush()
	return tokens
}

type destroyAction struct {
	Target string
	Apply  func(absPath string) (string, error)
}

func removeManagedFile(path string, ownershipSignals []string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "skipped (path not found)", nil
		}
		return "", err
	}

	text := string(content)
	for _, signal := range ownershipSignals {
		if !strings.Contains(text, signal) {
			return fmt.Sprintf("skipped (missing ownership signal: %s)", signal), nil
		}
	}

	if err := os.Remove(path); err != nil {
		return "", err
	}
	return "deleted file", nil
}

func removeGeneratedRouteBlock(content, snake string) (string, bool) {
	lines := strings.Split(content, "\n")
	marker := "\t// ship:generated:" + snake
	out := make([]string, 0, len(lines))
	removed := false

	for i := 0; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == strings.TrimSpace(marker) {
			removed = true
			i++
			for i < len(lines) && strings.TrimSpace(lines[i]) != "" {
				i++
			}
			if i < len(lines) && strings.TrimSpace(lines[i]) == "" {
				// Drop the separator line directly after the generated block.
			} else {
				i--
			}
			continue
		}
		out = append(out, lines[i])
	}

	return strings.Join(out, "\n"), removed
}

func removeRouteNameConstant(content, constName, constValue string) (string, bool) {
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	needle := constName + " = " + fmt.Sprintf("%q", constValue)
	removed := false

	for _, line := range lines {
		if strings.Contains(line, needle) {
			removed = true
			continue
		}
		out = append(out, line)
	}

	return strings.Join(out, "\n"), removed
}
