package generators

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"unicode"
)

type generatorPreview struct {
	Title string
	Body  string
}

func writeGeneratorReport(w io.Writer, kind string, dryRun bool, created, updated []string, previews []generatorPreview, next []string) {
	mode := ""
	if dryRun {
		mode = " (dry-run)"
	}
	fmt.Fprintf(w, "make:%s result%s\n", kind, mode)
	writeGeneratorPathSection(w, "Created", created)
	writeGeneratorPathSection(w, "Updated", updated)
	for _, preview := range previews {
		title := strings.TrimSpace(preview.Title)
		if title == "" && strings.TrimSpace(preview.Body) == "" {
			continue
		}
		fmt.Fprintln(w, "Preview:")
		if title != "" {
			fmt.Fprintf(w, "%s:\n", title)
		}
		body := strings.TrimSpace(preview.Body)
		if body != "" {
			fmt.Fprintln(w, body)
		}
	}
	writeGeneratorPathSection(w, "Next", next)
}

func writeGeneratorPathSection(w io.Writer, heading string, values []string) {
	if len(values) == 0 {
		return
	}
	fmt.Fprintf(w, "%s:\n", heading)
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		fmt.Fprintf(w, "- %s\n", value)
	}
}

func splitWords(input string) []string {
	clean := strings.TrimSpace(input)
	if clean == "" {
		return nil
	}
	clean = strings.ReplaceAll(clean, "-", " ")
	clean = strings.ReplaceAll(clean, "_", " ")
	parts := strings.Fields(clean)
	if len(parts) > 1 {
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.ToLower(p)
			if p != "" {
				out = append(out, p)
			}
		}
		return out
	}
	word := parts[0]
	var out []string
	var cur []rune
	for i, r := range word {
		if i > 0 && unicode.IsUpper(r) && len(cur) > 0 {
			out = append(out, strings.ToLower(string(cur)))
			cur = cur[:0]
		}
		cur = append(cur, r)
	}
	if len(cur) > 0 {
		out = append(out, strings.ToLower(string(cur)))
	}
	return out
}

func toPascalFromParts(parts []string) string {
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]))
		if len(p) > 1 {
			b.WriteString(strings.ToLower(p[1:]))
		}
	}
	return b.String()
}

func toLowerCamel(pascal string) string {
	if pascal == "" {
		return pascal
	}
	return strings.ToLower(pascal[:1]) + pascal[1:]
}

func insertAfterAnchor(src, anchor, snippet string) (string, bool, error) {
	if strings.Contains(src, snippet) {
		return src, false, nil
	}
	idx := strings.Index(src, anchor)
	if idx == -1 {
		return "", false, fmt.Errorf("anchor %q not found", anchor)
	}
	pos := idx + len(anchor)
	insert := "\n" + snippet
	return src[:pos] + insert + src[pos:], true, nil
}

func insertBeforeAnchor(src, anchor, snippet string) (string, bool, error) {
	if strings.Contains(src, snippet) {
		return src, false, nil
	}
	idx := strings.Index(src, anchor)
	if idx == -1 {
		return "", false, fmt.Errorf("anchor %q not found", anchor)
	}
	insert := snippet
	if !strings.HasSuffix(insert, "\n") {
		insert += "\n"
	}
	return src[:idx] + insert + src[idx:], true, nil
}

func normalizeOwnedGeneratorPath(raw, owner string) (string, error) {
	cleanOwner := filepath.ToSlash(filepath.Clean(strings.TrimSpace(owner)))
	cleanPath := filepath.ToSlash(filepath.Clean(strings.TrimSpace(raw)))

	if strings.TrimSpace(raw) == "" || cleanPath == "." {
		return "", errors.New("path cannot be empty")
	}
	if filepath.IsAbs(strings.TrimSpace(raw)) {
		return "", fmt.Errorf("path %q escapes canonical %s-owned location %q", raw, cleanOwner, cleanOwner)
	}
	if cleanPath == ".." || strings.HasPrefix(cleanPath, "../") {
		return "", fmt.Errorf("path %q escapes canonical %s-owned location %q", raw, cleanOwner, cleanOwner)
	}
	if cleanPath != cleanOwner {
		return "", fmt.Errorf("path %q escapes canonical %s-owned location %q", raw, cleanOwner, cleanOwner)
	}
	return cleanOwner, nil
}
