package generators

import (
	"fmt"
	"strings"
	"unicode"
)

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
