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

type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type toolCallResult struct {
	Content []toolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

func toolDefinitions() []map[string]any {
	return []map[string]any{
		{
			"name":        "ship_help",
			"description": "Get usage/help text for ship CLI commands.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"topic": map[string]any{
						"type":        "string",
						"enum":        []string{"general", "dev", "test", "db"},
						"description": "Optional help topic.",
					},
				},
			},
		},
		{
			"name":        "docs_search",
			"description": "Search markdown docs under docs/ and return matching lines.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Case-insensitive text to find.",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum number of matches (default 20, max 50).",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			"name":        "docs_get",
			"description": "Read one markdown doc by relative path under docs/.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Path under docs/, for example architecture/01-architecture.md.",
					},
				},
				"required": []string{"path"},
			},
		},
	}
}

func (s *mcpServer) handleToolsCall(paramsJSON json.RawMessage) (toolCallResult, error) {
	var params toolCallParams
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return toolCallResult{}, fmt.Errorf("invalid tool call params: %w", err)
	}

	switch params.Name {
	case "ship_help":
		return s.callShipHelp(params.Arguments)
	case "docs_search":
		return s.callDocsSearch(params.Arguments)
	case "docs_get":
		return s.callDocsGet(params.Arguments)
	default:
		return toolCallResult{}, fmt.Errorf("unknown tool: %s", params.Name)
	}
}

func (s *mcpServer) callShipHelp(arguments json.RawMessage) (toolCallResult, error) {
	var in struct {
		Topic string `json:"topic"`
	}
	if len(arguments) > 0 {
		if err := json.Unmarshal(arguments, &in); err != nil {
			return toolCallResult{}, fmt.Errorf("invalid ship_help arguments: %w", err)
		}
	}

	text := shipHelpByTopic(strings.TrimSpace(strings.ToLower(in.Topic)))
	return toolCallResult{Content: []toolContent{{Type: "text", Text: text}}}, nil
}

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

func shipHelpByTopic(topic string) string {
	switch topic {
	case "dev":
		return "ship dev commands:\n  ship dev\n  ship dev worker\n  ship dev all\n  ship dev --worker\n  ship dev --all"
	case "test":
		return "ship test commands:\n  ship test\n  ship test --integration"
	case "db":
		return "ship db commands:\n  ship db create\n  ship db migrate\n  ship db rollback [amount]\n  ship db seed"
	default:
		return "ship - GoShip CLI\n\nUsage:\n  ship dev [worker|all] [--worker|--all]\n  ship test [--integration]\n  ship db <create|migrate|rollback|seed>"
	}
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
