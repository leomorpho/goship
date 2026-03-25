package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
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

	shipCmd, shipArgs, err := resolveShipInvocation(s.repoRoot)
	if err != nil {
		return toolCallResult{Content: []toolContent{{Type: "text", Text: marshalShipDoctorResult(shipCLIMissingDoctorResult(err))}}}, nil
	}

	out, err := runShipJSON(shipCmd, append(shipArgs, "doctor", "--json")...)
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

	payload, toolErr := runShipDescribePayload(s.repoRoot)
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
			access := describeRouteAccessClass(route)
			switch filter {
			case "public":
				if access == "public" {
					filtered = append(filtered, route)
				}
			case "auth":
				if access == "auth" {
					filtered = append(filtered, route)
				}
			case "admin":
				if access == "admin" {
					filtered = append(filtered, route)
				}
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

func (s *mcpServer) callShipRuntimeReport(arguments json.RawMessage) (toolCallResult, error) {
	if len(arguments) > 0 && string(arguments) != "{}" {
		var in map[string]any
		if err := json.Unmarshal(arguments, &in); err != nil {
			return toolCallResult{}, fmt.Errorf("invalid ship_runtime_report arguments: %w", err)
		}
		if len(in) > 0 {
			return toolCallResult{}, errors.New("ship_runtime_report does not accept arguments")
		}
	}

	shipCmd, shipArgs, err := resolveShipInvocation(s.repoRoot)
	if err != nil {
		return toolCallResult{
			Content: []toolContent{{Type: "text", Text: fmt.Sprintf(`{"error":"ship runtime report unavailable: %s"}`, strings.ReplaceAll(err.Error(), `"`, `'`))}},
			IsError: true,
		}, nil
	}
	out, err := runShipJSON(shipCmd, append(shipArgs, "runtime:report", "--json")...)
	if err != nil {
		return toolCallResult{
			Content: []toolContent{{Type: "text", Text: fmt.Sprintf(`{"error":"failed to run ship runtime:report --json: %s"}`, strings.ReplaceAll(err.Error(), `"`, `'`))}},
			IsError: true,
		}, nil
	}
	if !json.Valid(out) {
		return toolCallResult{
			Content: []toolContent{{Type: "text", Text: fmt.Sprintf(`{"error":"invalid ship runtime report JSON output: %s"}`, strings.ReplaceAll(strings.TrimSpace(string(out)), `"`, `'`))}},
			IsError: true,
		}, nil
	}

	return toolCallResult{Content: []toolContent{{Type: "text", Text: string(out)}}}, nil
}

func describeRouteAccessClass(route shipDescribeRoute) string {
	access := strings.ToLower(strings.TrimSpace(route.Access))
	if access == "public" || access == "auth" || access == "admin" {
		return access
	}

	path := strings.ToLower(strings.Trim(strings.TrimSpace(route.Path), "`\""))
	if strings.Contains(path, "/auth/admin") || strings.HasPrefix(path, "/admin") {
		return "admin"
	}
	if route.Auth || strings.HasPrefix(path, "/auth") {
		return "auth"
	}
	return "public"
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

	payload, toolErr := runShipDescribePayload(s.repoRoot)
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

	shipCmd, shipArgs, err := resolveShipInvocation(s.repoRoot)
	if err != nil {
		return toolCallResult{Content: []toolContent{{
			Type: "text",
			Text: marshalShipVerifyResult(shipCLIMissingVerifyResult(err)),
		}}}, nil
	}

	args := []string{"verify", "--json"}
	if in.SkipTests {
		args = append(args, "--skip-tests")
	}

	out, err := runShipJSON(shipCmd, append(shipArgs, args...)...)
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

func (s *mcpServer) callShipScaffold(arguments json.RawMessage) (toolCallResult, error) {
	var in shipScaffoldInput
	if len(arguments) == 0 {
		return toolCallResult{}, errors.New("ship_scaffold requires arguments")
	}
	if err := json.Unmarshal(arguments, &in); err != nil {
		return toolCallResult{}, fmt.Errorf("invalid ship_scaffold arguments: %w", err)
	}
	in.Resource = strings.TrimSpace(in.Resource)
	if in.Resource == "" {
		return toolCallResult{}, errors.New("ship_scaffold resource is required")
	}
	args, err := buildShipScaffoldArgs(in)
	if err != nil {
		return toolCallResult{}, err
	}

	repoRoot := s.repoRoot
	if repoRoot == "" {
		repoRoot = "."
	}
	before, beforeErr := runGitStatus(repoRoot)

	shipCmd, shipArgs, lookErr := resolveShipInvocation(s.repoRoot)
	if lookErr != nil {
		res := shipScaffoldResult{
			OK:           false,
			FilesCreated: []string{},
			Errors:       []string{fmt.Sprintf("ship CLI is unavailable: %v", lookErr)},
		}
		if beforeErr != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("git status (before) failed: %v", beforeErr))
		}
		return toolCallResult{
			Content: []toolContent{{Type: "text", Text: marshalShipScaffoldResult(res)}},
			IsError: true,
		}, nil
	}

	output, cmdErr := runShipJSON(shipCmd, append(shipArgs, args...)...)
	after, afterErr := runGitStatus(repoRoot)
	files := diffGitStatus(before, after)

	errs := make([]string, 0, 3)
	if beforeErr != nil {
		errs = append(errs, fmt.Sprintf("git status (before) failed: %v", beforeErr))
	}
	if cmdErr != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed != "" {
			errs = append(errs, fmt.Sprintf("ship make:scaffold failed: %v: %s", cmdErr, trimmed))
		} else {
			errs = append(errs, fmt.Sprintf("ship make:scaffold failed: %v", cmdErr))
		}
	}
	if afterErr != nil {
		errs = append(errs, fmt.Sprintf("git status (after) failed: %v", afterErr))
	}

	ok := cmdErr == nil
	hasError := !ok || beforeErr != nil || afterErr != nil
	res := shipScaffoldResult{
		OK:           ok,
		FilesCreated: files,
		Errors:       errs,
	}
	return toolCallResult{
		Content: []toolContent{{Type: "text", Text: marshalShipScaffoldResult(res)}},
		IsError: hasError,
	}, nil
}

func marshalShipDoctorResult(result shipDoctorResult) string {
	b, err := json.Marshal(result)
	if err != nil {
		return `{"ok":false,"issues":[{"type":"config","file":"","detail":"failed to encode ship doctor result","severity":"error"}]}`
	}
	return string(b)
}

func shipCLIMissingDoctorResult(err error) shipDoctorResult {
	return shipDoctorResult{
		OK: false,
		Issues: []shipDoctorIssue{{
			Type:     "config",
			File:     "",
			Detail:   fmt.Sprintf("ship CLI is unavailable: %v", err),
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

func marshalShipScaffoldResult(result shipScaffoldResult) string {
	b, err := json.Marshal(result)
	if err != nil {
		return `{"ok":false,"files_created":[],"errors":["failed to encode ship scaffold result"]}`
	}
	return string(b)
}

func shipCLIMissingVerifyResult(err error) shipVerifyResult {
	return shipVerifyResult{
		OK: false,
		Steps: []shipVerifyStep{{
			Name:   "ship verify --json",
			OK:     false,
			Output: fmt.Sprintf("ship CLI is unavailable: %v", err),
		}},
	}
}

func buildShipScaffoldArgs(in shipScaffoldInput) ([]string, error) {
	resource := toPascalCase(in.Resource)
	if resource == "" {
		return nil, fmt.Errorf("invalid resource name %q", in.Resource)
	}
	args := []string{"make:scaffold", resource}
	for _, field := range in.Fields {
		arg, err := formatShipScaffoldField(field)
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
	}
	return args, nil
}

func formatShipScaffoldField(field shipScaffoldField) (string, error) {
	name := strings.TrimSpace(field.Name)
	typ := strings.TrimSpace(field.Type)
	if name == "" || typ == "" {
		return "", errors.New("each field requires a name and type")
	}
	snake := toSnakeCase(name)
	if snake == "" {
		return "", fmt.Errorf("invalid field name %q", field.Name)
	}
	return fmt.Sprintf("%s:%s", snake, strings.ToLower(typ)), nil
}

func toPascalCase(input string) string {
	var parts []string
	var buffer []rune
	addPart := func() {
		if len(buffer) == 0 {
			return
		}
		parts = append(parts, string(buffer))
		buffer = buffer[:0]
	}
	for _, r := range input {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			buffer = append(buffer, r)
			continue
		}
		addPart()
	}
	addPart()
	if len(parts) == 0 {
		return ""
	}
	for i, part := range parts {
		runes := []rune(strings.ToLower(part))
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, "")
}

func toSnakeCase(input string) string {
	var out []rune
	lastWasSep := false
	for _, r := range input {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if unicode.IsUpper(r) && len(out) > 0 && !lastWasSep && (unicode.IsLower(out[len(out)-1]) || unicode.IsDigit(out[len(out)-1])) {
				out = append(out, '_')
			}
			out = append(out, unicode.ToLower(r))
			lastWasSep = false
			continue
		}
		if len(out) > 0 && !lastWasSep {
			out = append(out, '_')
			lastWasSep = true
		}
	}
	for len(out) > 0 && out[0] == '_' {
		out = out[1:]
	}
	for len(out) > 0 && out[len(out)-1] == '_' {
		out = out[:len(out)-1]
	}
	return string(out)
}

func diffGitStatus(before, after map[string]string) []string {
	if before == nil {
		before = map[string]string{}
	}
	if after == nil {
		after = map[string]string{}
	}
	changed := make(map[string]struct{})
	for path, status := range after {
		if prev, ok := before[path]; !ok || prev != status {
			changed[path] = struct{}{}
		}
	}
	for path := range before {
		if _, ok := after[path]; !ok {
			changed[path] = struct{}{}
		}
	}
	files := make([]string, 0, len(changed))
	for path := range changed {
		files = append(files, filepath.ToSlash(path))
	}
	sort.Strings(files)
	return files
}

func parseGitStatusOutput(out []byte) map[string]string {
	result := make(map[string]string)
	norm := strings.ReplaceAll(strings.ReplaceAll(string(out), "\r\n", "\n"), "\r", "\n")
	for _, line := range strings.Split(norm, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || len(line) < 3 {
			continue
		}
		status := strings.TrimSpace(line[:2])
		path := extractGitPath(strings.TrimSpace(line[2:]))
		if path == "" {
			continue
		}
		result[path] = status
	}
	return result
}

func extractGitPath(raw string) string {
	if idx := strings.Index(raw, "->"); idx != -1 {
		raw = raw[idx+2:]
	}
	return strings.TrimSpace(raw)
}

func resolveShipInvocation(repoRoot string) (string, []string, error) {
	cleanRoot := strings.TrimSpace(repoRoot)
	if cleanRoot != "" {
		localMain := filepath.Join(cleanRoot, "tools", "cli", "ship", "cmd", "ship", "main.go")
		if info, err := os.Stat(localMain); err == nil && !info.IsDir() {
			goPath, goErr := lookPathGo("go")
			if goErr != nil {
				return "", nil, fmt.Errorf("repo-local ship CLI exists at %s but Go toolchain was not found: %w", localMain, goErr)
			}
			return goPath, []string{"run", filepath.Join(cleanRoot, "tools", "cli", "ship", "cmd", "ship")}, nil
		}
	}

	shipPath, err := lookPathShip("ship")
	if err != nil {
		return "", nil, err
	}
	return shipPath, nil, nil
}

var runShipDescribePayload = func(repoRoot string) (shipDescribeResult, error) {
	shipCmd, shipArgs, err := resolveShipInvocation(repoRoot)
	if err != nil {
		return shipDescribeResult{}, err
	}
	out, err := runShipJSON(shipCmd, append(shipArgs, "describe")...)
	if err != nil {
		return shipDescribeResult{}, err
	}
	var payload shipDescribeResult
	if err := json.Unmarshal(out, &payload); err != nil {
		return shipDescribeResult{}, fmt.Errorf("invalid ship describe JSON output: %s", strings.TrimSpace(string(out)))
	}
	return payload, nil
}
