package server

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
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

type shipDescribeRoute struct {
	Method  string `json:"method"`
	Path    string `json:"path"`
	Handler string `json:"handler"`
	Auth    bool   `json:"auth"`
	File    string `json:"file"`
}

type shipDescribeModule struct {
	ID         string `json:"id"`
	Installed  bool   `json:"installed"`
	Routes     int    `json:"routes"`
	Migrations int    `json:"migrations"`
}

type shipDescribeResult struct {
	Routes  []shipDescribeRoute  `json:"routes"`
	Modules []shipDescribeModule `json:"modules"`
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
			"name":        "ship_doctor",
			"description": "Run `ship doctor --json` and return the structured result.",
			"inputSchema": map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			"name":        "ship_routes",
			"description": "Run `ship describe` and return the route inventory.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"filter": map[string]any{
						"type":        "string",
						"enum":        []string{"public", "auth", "admin"},
						"description": "Optional route auth filter.",
					},
				},
			},
		},
		{
			"name":        "ship_modules",
			"description": "Run `ship describe` and return the installed module list.",
			"inputSchema": map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			"name":        "ship_verify",
			"description": "Run `ship verify --json` and return the step-by-step verification result.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"skip_tests": map[string]any{
						"type":        "boolean",
						"description": "Skip the final go test ./... step.",
					},
				},
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
	case "ship_doctor":
		return s.callShipDoctor(params.Arguments)
	case "ship_routes":
		return s.callShipRoutes(params.Arguments)
	case "ship_modules":
		return s.callShipModules(params.Arguments)
	case "ship_verify":
		return s.callShipVerify(params.Arguments)
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

type shipDoctorIssue struct {
	Type     string `json:"type"`
	File     string `json:"file"`
	Detail   string `json:"detail"`
	Severity string `json:"severity"`
}

type shipDoctorResult struct {
	OK     bool              `json:"ok"`
	Issues []shipDoctorIssue `json:"issues"`
}

type shipVerifyStep struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Output string `json:"output"`
}

type shipVerifyResult struct {
	OK    bool             `json:"ok"`
	Steps []shipVerifyStep `json:"steps"`
}

var (
	lookPathShip = exec.LookPath
	runShipJSON  = func(name string, args ...string) ([]byte, error) {
		return exec.Command(name, args...).CombinedOutput()
	}
)

func (s *mcpServer) callShipDoctor(arguments json.RawMessage) (toolCallResult, error) {
	if len(arguments) > 0 && string(arguments) != "{}" {
		var in map[string]any
		if err := json.Unmarshal(arguments, &in); err != nil {
			return toolCallResult{}, fmt.Errorf("invalid ship_doctor arguments: %w", err)
		}
		if len(in) > 0 {
			return toolCallResult{}, errors.New("ship_doctor does not accept arguments")
		}
	}

	shipPath, err := lookPathShip("ship")
	if err != nil {
		return toolCallResult{Content: []toolContent{{Type: "text", Text: marshalShipDoctorResult(shipBinaryMissingDoctorResult())}}}, nil
	}

	out, err := runShipJSON(shipPath, "doctor", "--json")
	if err != nil {
		return toolCallResult{Content: []toolContent{{
			Type: "text",
			Text: marshalShipDoctorResult(shipDoctorResult{
				OK: false,
				Issues: []shipDoctorIssue{{
					Type:     "config",
					File:     "",
					Detail:   fmt.Sprintf("failed to run ship doctor --json: %v", err),
					Severity: "error",
				}},
			}),
		}}}, nil
	}

	var payload shipDoctorResult
	if err := json.Unmarshal(out, &payload); err != nil {
		return toolCallResult{Content: []toolContent{{
			Type: "text",
			Text: marshalShipDoctorResult(shipDoctorResult{
				OK: false,
				Issues: []shipDoctorIssue{{
					Type:     "config",
					File:     "",
					Detail:   fmt.Sprintf("invalid ship doctor JSON output: %s", strings.TrimSpace(string(out))),
					Severity: "error",
				}},
			}),
		}}}, nil
	}

	return toolCallResult{Content: []toolContent{{
		Type: "text",
		Text: marshalShipDoctorResult(payload),
	}}}, nil
}

func (s *mcpServer) callShipRoutes(arguments json.RawMessage) (toolCallResult, error) {
	var in struct {
		Filter string `json:"filter"`
	}
	if len(arguments) > 0 {
		if err := json.Unmarshal(arguments, &in); err != nil {
			return toolCallResult{}, fmt.Errorf("invalid ship_routes arguments: %w", err)
		}
	}
	filter := strings.TrimSpace(strings.ToLower(in.Filter))
	if filter != "" && filter != "public" && filter != "auth" && filter != "admin" {
		return toolCallResult{}, errors.New("ship_routes filter must be one of public, auth, admin")
	}

	payload, toolErr := runShipDescribePayload()
	if toolErr != nil {
		return toolCallResult{
			Content: []toolContent{{Type: "text", Text: `{"routes":[]}`}},
			IsError: true,
		}, nil
	}

	routes := payload.Routes
	if filter != "" {
		filtered := make([]shipDescribeRoute, 0, len(routes))
		for _, route := range routes {
			switch filter {
			case "public":
				if !route.Auth {
					filtered = append(filtered, route)
				}
			case "auth":
				if route.Auth {
					filtered = append(filtered, route)
				}
			case "admin":
				// Current describe schema does not distinguish admin routes yet.
			}
		}
		routes = filtered
	}

	b, err := json.Marshal(map[string]any{"routes": routes})
	if err != nil {
		return toolCallResult{}, err
	}
	return toolCallResult{Content: []toolContent{{Type: "text", Text: string(b)}}}, nil
}

func (s *mcpServer) callShipModules(arguments json.RawMessage) (toolCallResult, error) {
	if len(arguments) > 0 && string(arguments) != "{}" {
		var in map[string]any
		if err := json.Unmarshal(arguments, &in); err != nil {
			return toolCallResult{}, fmt.Errorf("invalid ship_modules arguments: %w", err)
		}
		if len(in) > 0 {
			return toolCallResult{}, errors.New("ship_modules does not accept arguments")
		}
	}

	payload, toolErr := runShipDescribePayload()
	if toolErr != nil {
		return toolCallResult{
			Content: []toolContent{{Type: "text", Text: `{"modules":[]}`}},
			IsError: true,
		}, nil
	}

	b, err := json.Marshal(map[string]any{"modules": payload.Modules})
	if err != nil {
		return toolCallResult{}, err
	}
	return toolCallResult{Content: []toolContent{{Type: "text", Text: string(b)}}}, nil
}

func (s *mcpServer) callShipVerify(arguments json.RawMessage) (toolCallResult, error) {
	var in struct {
		SkipTests bool `json:"skip_tests"`
	}
	if len(arguments) > 0 {
		if err := json.Unmarshal(arguments, &in); err != nil {
			return toolCallResult{}, fmt.Errorf("invalid ship_verify arguments: %w", err)
		}
	}

	shipPath, err := lookPathShip("ship")
	if err != nil {
		return toolCallResult{Content: []toolContent{{
			Type: "text",
			Text: marshalShipVerifyResult(shipBinaryMissingVerifyResult()),
		}}}, nil
	}

	args := []string{"verify", "--json"}
	if in.SkipTests {
		args = append(args, "--skip-tests")
	}

	out, err := runShipJSON(shipPath, args...)
	if err != nil {
		return toolCallResult{Content: []toolContent{{
			Type: "text",
			Text: marshalShipVerifyResult(shipVerifyResult{
				OK: false,
				Steps: []shipVerifyStep{{
					Name:   strings.Join(append([]string{"ship"}, args...), " "),
					OK:     false,
					Output: fmt.Sprintf("failed to run %s: %v", strings.Join(args, " "), err),
				}},
			}),
		}}}, nil
	}

	var payload shipVerifyResult
	if err := json.Unmarshal(out, &payload); err != nil {
		return toolCallResult{Content: []toolContent{{
			Type: "text",
			Text: marshalShipVerifyResult(shipVerifyResult{
				OK: false,
				Steps: []shipVerifyStep{{
					Name:   strings.Join(append([]string{"ship"}, args...), " "),
					OK:     false,
					Output: fmt.Sprintf("invalid ship verify JSON output: %s", strings.TrimSpace(string(out))),
				}},
			}),
		}}}, nil
	}

	return toolCallResult{Content: []toolContent{{
		Type: "text",
		Text: marshalShipVerifyResult(payload),
	}}}, nil
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

func marshalShipDoctorResult(result shipDoctorResult) string {
	b, err := json.Marshal(result)
	if err != nil {
		return `{"ok":false,"issues":[{"type":"config","file":"","detail":"failed to encode ship doctor result","severity":"error"}]}`
	}
	return string(b)
}

func shipBinaryMissingDoctorResult() shipDoctorResult {
	return shipDoctorResult{
		OK: false,
		Issues: []shipDoctorIssue{{
			Type:     "config",
			File:     "",
			Detail:   "ship binary not found in PATH",
			Severity: "error",
		}},
	}
}

func marshalShipVerifyResult(result shipVerifyResult) string {
	b, err := json.Marshal(result)
	if err != nil {
		return `{"ok":false,"steps":[{"name":"ship verify --json","ok":false,"output":"failed to encode ship verify result"}]}`
	}
	return string(b)
}

func shipBinaryMissingVerifyResult() shipVerifyResult {
	return shipVerifyResult{
		OK: false,
		Steps: []shipVerifyStep{{
			Name:   "ship verify --json",
			OK:     false,
			Output: "ship binary not found in PATH",
		}},
	}
}

func runShipDescribePayload() (shipDescribeResult, error) {
	shipPath, err := lookPathShip("ship")
	if err != nil {
		return shipDescribeResult{}, err
	}
	out, err := runShipJSON(shipPath, "describe")
	if err != nil {
		return shipDescribeResult{}, err
	}
	var payload shipDescribeResult
	if err := json.Unmarshal(out, &payload); err != nil {
		return shipDescribeResult{}, fmt.Errorf("invalid ship describe JSON output: %s", strings.TrimSpace(string(out)))
	}
	return payload, nil
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
