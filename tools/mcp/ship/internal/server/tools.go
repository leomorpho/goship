package server

import (
	"encoding/json"
	"fmt"
	"os/exec"
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
			"name":        "ship_scaffold",
			"description": "Run `ship make:scaffold` and report the files that were touched.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"resource": map[string]any{
						"type":        "string",
						"description": "PascalCase name for the resource to scaffold.",
					},
					"fields": map[string]any{
						"type":        "array",
						"description": "List of field definitions for the model.",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"name": map[string]any{"type": "string"},
								"type": map[string]any{"type": "string"},
							},
							"required": []string{"name", "type"},
						},
					},
				},
				"required": []string{"resource"},
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
	case "ship_scaffold":
		return s.callShipScaffold(params.Arguments)
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

type shipScaffoldField struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type shipScaffoldInput struct {
	Resource string              `json:"resource"`
	Fields   []shipScaffoldField `json:"fields,omitempty"`
}

type shipScaffoldResult struct {
	OK           bool     `json:"ok"`
	FilesCreated []string `json:"files_created"`
	Errors       []string `json:"errors"`
}

var (
	lookPathShip = exec.LookPath
	runShipJSON  = func(name string, args ...string) ([]byte, error) {
		return exec.Command(name, args...).CombinedOutput()
	}
	runGitStatus = func(dir string) (map[string]string, error) {
		cmd := exec.Command("git", "status", "--short")
		if dir != "" {
			cmd.Dir = dir
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("git status --short: %w (%s)", err, strings.TrimSpace(string(out)))
		}
		return parseGitStatusOutput(out), nil
	}
)

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
