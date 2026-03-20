package server

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func (s *mcpServer) callDocsGet(arguments json.RawMessage) (toolCallResult, error) {
	var in struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(arguments, &in); err != nil {
		return toolCallResult{}, fmt.Errorf("invalid docs_get arguments: %w", err)
	}
	if strings.TrimSpace(in.Path) == "" {
		return toolCallResult{}, errors.New("docs_get path is required")
	}

	absPath, relPath, err := resolveDocPath(s.docsRoot, in.Path)
	if err != nil {
		return toolCallResult{}, err
	}

	body, err := os.ReadFile(absPath)
	if err != nil {
		return toolCallResult{}, fmt.Errorf("read %s: %w", relPath, err)
	}

	text := string(body)
	const maxChars = 24000
	if len(text) > maxChars {
		text = text[:maxChars] + "\n\n[truncated]"
	}

	return toolCallResult{Content: []toolContent{{
		Type: "text",
		Text: fmt.Sprintf("# %s\n\n%s", relPath, text),
	}}}, nil
}

func (s *mcpServer) callDocsSearch(arguments json.RawMessage) (toolCallResult, error) {
	var in struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(arguments, &in); err != nil {
		return toolCallResult{}, fmt.Errorf("invalid docs_search arguments: %w", err)
	}

	query := strings.TrimSpace(in.Query)
	if query == "" {
		return toolCallResult{}, errors.New("docs_search query is required")
	}

	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	matches, err := searchDocs(s.docsRoot, query, limit)
	if err != nil {
		return toolCallResult{}, err
	}
	if len(matches) == 0 {
		return toolCallResult{Content: []toolContent{{Type: "text", Text: "No matches."}}}, nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Matches for %q:\n\n", query)
	for _, m := range matches {
		fmt.Fprintf(&b, "- %s:%d %s\n", m.Path, m.Line, m.Text)
	}
	return toolCallResult{Content: []toolContent{{Type: "text", Text: b.String()}}}, nil
}

type searchMatch struct {
	Path string
	Line int
	Text string
}

func searchDocs(docsRoot, query string, limit int) ([]searchMatch, error) {
	query = strings.ToLower(query)
	matches := make([]searchMatch, 0, limit)

	err := filepath.WalkDir(docsRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}

		rel, err := filepath.Rel(docsRoot, path)
		if err != nil {
			return err
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		s := bufio.NewScanner(f)
		lineNo := 0
		for s.Scan() {
			lineNo++
			line := s.Text()
			if strings.Contains(strings.ToLower(line), query) {
				matches = append(matches, searchMatch{
					Path: filepath.ToSlash(rel),
					Line: lineNo,
					Text: strings.TrimSpace(line),
				})
				if len(matches) >= limit {
					return ioEOFStop
				}
			}
		}
		if err := s.Err(); err != nil {
			return err
		}
		return nil
	})

	if err != nil && !errors.Is(err, ioEOFStop) {
		return nil, err
	}
	return matches, nil
}

var ioEOFStop = errors.New("stop walk")

func resolveDocPath(docsRoot, input string) (absPath, relPath string, err error) {
	p := strings.TrimSpace(filepath.ToSlash(input))
	p = strings.TrimPrefix(p, "docs/")
	if p == "" {
		return "", "", errors.New("path is empty")
	}
	if strings.HasPrefix(p, "../") || strings.Contains(p, "/../") {
		return "", "", errors.New("path must stay within docs/")
	}

	clean := filepath.Clean(filepath.FromSlash(p))
	if clean == "." || clean == "" {
		return "", "", errors.New("path is empty")
	}
	if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return "", "", errors.New("path must stay within docs/")
	}

	absRoot, err := filepath.Abs(docsRoot)
	if err != nil {
		return "", "", err
	}

	abs := filepath.Join(absRoot, clean)
	abs = filepath.Clean(abs)
	if !strings.HasPrefix(abs, absRoot+string(filepath.Separator)) && abs != absRoot {
		return "", "", errors.New("path must stay within docs/")
	}

	rel := filepath.ToSlash(clean)
	if filepath.Ext(rel) == "" {
		rel += ".md"
		abs += ".md"
	}

	return abs, rel, nil
}

func toInt(v any, def int) int {
	switch t := v.(type) {
	case int:
		return t
	case float64:
		return int(t)
	case string:
		n, err := strconv.Atoi(t)
		if err == nil {
			return n
		}
	}
	return def
}
