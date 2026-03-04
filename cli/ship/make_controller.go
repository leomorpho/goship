package ship

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
)

type controllerMakeOptions struct {
	Name    string
	Path    string
	Actions []string
	Auth    string
	Wire    bool
	Domain  string
}

type controllerNames struct {
	BaseSnake string
	BaseKebab string
	BaseTitle string
	VarName   string
	TypeName  string
	FileName  string
}

func (c CLI) runMakeController(args []string) int {
	opts, err := parseMakeControllerArgs(args)
	if err != nil {
		fmt.Fprintf(c.Err, "invalid make:controller arguments: %v\n", err)
		return 1
	}
	names, err := normalizeControllerName(opts.Name)
	if err != nil {
		fmt.Fprintf(c.Err, "invalid controller name: %v\n", err)
		return 1
	}

	controllerPath := filepath.Join(opts.Path, "web", "controllers", names.FileName)
	if hasFile(controllerPath) {
		fmt.Fprintf(c.Err, "refusing to overwrite existing controller file: %s\n", controllerPath)
		return 1
	}

	domain, err := normalizeDomainTarget(opts.Domain)
	if err != nil {
		fmt.Fprintf(c.Err, "invalid --domain value: %v\n", err)
		return 1
	}

	content, err := renderControllerFile(names, opts.Actions, domain)
	if err != nil {
		fmt.Fprintf(c.Err, "failed to render controller: %v\n", err)
		return 1
	}
	if err := os.MkdirAll(filepath.Dir(controllerPath), 0o755); err != nil {
		fmt.Fprintf(c.Err, "failed to create controller directory: %v\n", err)
		return 1
	}
	if err := os.WriteFile(controllerPath, []byte(content), 0o644); err != nil {
		fmt.Fprintf(c.Err, "failed to write controller file: %v\n", err)
		return 1
	}

	fmt.Fprintf(c.Out, "Generated controller: %s\n", controllerPath)

	routeSnippet := renderControllerRouteSnippet(names, opts.Actions, opts.Auth, domain.Name != "")
	if opts.Wire {
		routerPath := filepath.Join(opts.Path, "router.go")
		if err := ensureRouteNamesImport(routerPath, false); err != nil {
			fmt.Fprintf(c.Err, "failed to ensure routeNames import: %v\n", err)
			return 1
		}
		if err := wireRouteSnippet(routerPath, opts.Auth, routeSnippet, false); err != nil {
			fmt.Fprintf(c.Err, "failed to wire controller routes: %v\n", err)
			return 1
		}
		fmt.Fprintf(c.Out, "Wired routes into %s\n", routerPath)
	} else {
		fmt.Fprintln(c.Out, "Route snippet:")
		fmt.Fprintln(c.Out, routeSnippet)
	}

	return 0
}

func parseMakeControllerArgs(args []string) (controllerMakeOptions, error) {
	opts := controllerMakeOptions{
		Path:    "apps/goship",
		Actions: []string{"index"},
		Auth:    "public",
	}
	if len(args) == 0 {
		return opts, errors.New("usage: ship make:controller <Name|NameController> [--actions index,show,create,update,destroy] [--auth public|auth] [--domain <name>] [--wire]")
	}
	opts.Name = strings.TrimSpace(args[0])
	if opts.Name == "" {
		return opts, errors.New("usage: ship make:controller <Name|NameController> [--actions index,show,create,update,destroy] [--auth public|auth] [--domain <name>] [--wire]")
	}

	for i := 1; i < len(args); i++ {
		switch {
		case args[i] == "--wire":
			opts.Wire = true
		case strings.HasPrefix(args[i], "--path="):
			opts.Path = strings.TrimSpace(strings.TrimPrefix(args[i], "--path="))
		case args[i] == "--path":
			if i+1 >= len(args) {
				return opts, errors.New("missing value for --path")
			}
			i++
			opts.Path = strings.TrimSpace(args[i])
		case strings.HasPrefix(args[i], "--auth="):
			opts.Auth = strings.TrimSpace(strings.TrimPrefix(args[i], "--auth="))
		case args[i] == "--auth":
			if i+1 >= len(args) {
				return opts, errors.New("missing value for --auth")
			}
			i++
			opts.Auth = strings.TrimSpace(args[i])
		case strings.HasPrefix(args[i], "--actions="):
			val := strings.TrimSpace(strings.TrimPrefix(args[i], "--actions="))
			actions, err := parseControllerActions(val)
			if err != nil {
				return opts, err
			}
			opts.Actions = actions
		case strings.HasPrefix(args[i], "--domain="):
			opts.Domain = strings.TrimSpace(strings.TrimPrefix(args[i], "--domain="))
		case args[i] == "--domain":
			if i+1 >= len(args) {
				return opts, errors.New("missing value for --domain")
			}
			i++
			opts.Domain = strings.TrimSpace(args[i])
		case args[i] == "--actions":
			if i+1 >= len(args) {
				return opts, errors.New("missing value for --actions")
			}
			i++
			actions, err := parseControllerActions(strings.TrimSpace(args[i]))
			if err != nil {
				return opts, err
			}
			opts.Actions = actions
		default:
			return opts, fmt.Errorf("unknown option: %s", args[i])
		}
	}

	if opts.Auth != "public" && opts.Auth != "auth" {
		return opts, fmt.Errorf("invalid --auth value %q (expected public|auth)", opts.Auth)
	}
	if strings.TrimSpace(opts.Path) == "" {
		return opts, errors.New("path cannot be empty")
	}
	return opts, nil
}

func parseControllerActions(raw string) ([]string, error) {
	if raw == "" {
		return nil, errors.New("actions cannot be empty")
	}
	allowed := []string{"index", "show", "create", "update", "destroy"}
	var out []string
	for _, token := range strings.Split(raw, ",") {
		action := strings.ToLower(strings.TrimSpace(token))
		if action == "" {
			continue
		}
		if !slices.Contains(allowed, action) {
			return nil, fmt.Errorf("unsupported action %q", action)
		}
		if !slices.Contains(out, action) {
			out = append(out, action)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("actions cannot be empty")
	}
	return out, nil
}

func normalizeControllerName(raw string) (controllerNames, error) {
	name := strings.TrimSpace(raw)
	name = strings.TrimSuffix(name, "Controller")
	parts := splitWords(name)
	if len(parts) == 0 {
		return controllerNames{}, errors.New("name cannot be empty")
	}
	baseSnake := strings.Join(parts, "_")
	baseKebab := strings.Join(parts, "-")
	baseTitle := toPascalFromParts(parts)
	varName := toLowerCamel(baseTitle)
	return controllerNames{
		BaseSnake: baseSnake,
		BaseKebab: baseKebab,
		BaseTitle: baseTitle,
		VarName:   varName,
		TypeName:  baseTitle + "Controller",
		FileName:  baseSnake + ".go",
	}, nil
}

type normalizedDomainTarget struct {
	Name   string
	Snake  string
	Pascal string
}

func normalizeDomainTarget(raw string) (normalizedDomainTarget, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return normalizedDomainTarget{}, nil
	}
	parts := splitWords(name)
	if len(parts) == 0 {
		return normalizedDomainTarget{}, errors.New("domain must contain letters or numbers")
	}
	return normalizedDomainTarget{
		Name:   name,
		Snake:  strings.Join(parts, "_"),
		Pascal: toPascalFromParts(parts),
	}, nil
}

func renderControllerFile(names controllerNames, actions []string, domain normalizedDomainTarget) (string, error) {
	var b strings.Builder
	b.WriteString("package controllers\n\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"net/http\"\n\n")
	b.WriteString("\t\"github.com/labstack/echo/v4\"\n")
	b.WriteString(")\n\n")
	b.WriteString("type ")
	b.WriteString(names.VarName)
	b.WriteString(" struct {\n")
	if domain.Name != "" {
		b.WriteString("\tdomainService any\n")
	}
	b.WriteString("}\n\n")
	b.WriteString("func New")
	b.WriteString(names.BaseTitle)
	b.WriteString("Controller(")
	if domain.Name != "" {
		b.WriteString("domainService any")
	}
	b.WriteString(") *")
	if domain.Name != "" {
		b.WriteString(names.VarName)
		b.WriteString(" {\n")
		b.WriteString("\treturn &")
		b.WriteString(names.VarName)
		b.WriteString("{domainService: domainService}\n")
		b.WriteString("}\n\n")
	} else {
		b.WriteString(names.VarName)
		b.WriteString(" {\n")
		b.WriteString("\treturn &")
		b.WriteString(names.VarName)
		b.WriteString("{}\n")
		b.WriteString("}\n\n")
	}
	for _, action := range actions {
		methodName := actionMethodName(action)
		b.WriteString("func (c *")
		b.WriteString(names.VarName)
		b.WriteString(") ")
		b.WriteString(methodName)
		b.WriteString("(ctx echo.Context) error {\n")
		if domain.Name != "" {
			b.WriteString("\t// TODO: delegate to domain service in apps/goship/app/")
			b.WriteString(domain.Snake)
			b.WriteString("\n")
		}
		b.WriteString("\treturn ctx.String(http.StatusNotImplemented, \"TODO: ")
		b.WriteString(names.BaseTitle)
		b.WriteString(".")
		b.WriteString(methodName)
		b.WriteString("\")\n")
		b.WriteString("}\n\n")
	}

	src, err := format.Source([]byte(b.String()))
	if err != nil {
		return "", err
	}
	return string(src), nil
}

func renderControllerRouteSnippet(names controllerNames, actions []string, auth string, withDomain bool) string {
	var b bytes.Buffer
	b.WriteString("\t// ship:generated:")
	b.WriteString(names.BaseSnake)
	b.WriteString("\n")
	b.WriteString("\t")
	b.WriteString(names.VarName)
	b.WriteString(" := controllers.New")
	b.WriteString(names.BaseTitle)
	b.WriteString("Controller(")
	if withDomain {
		b.WriteString("nil")
	}
	b.WriteString(")\n")
	for _, action := range actions {
		line := actionRouteLine(names, action, auth)
		if line != "" {
			b.WriteString("\t")
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	return b.String()
}

func actionMethodName(action string) string {
	switch action {
	case "index":
		return "Index"
	case "show":
		return "Show"
	case "create":
		return "Create"
	case "update":
		return "Update"
	default:
		return "Destroy"
	}
}

func actionRouteLine(names controllerNames, action, auth string) string {
	route := "/" + names.BaseKebab
	group := "g"
	if auth == "auth" {
		group = "onboardedGroup"
	}
	switch action {
	case "index":
		return fmt.Sprintf(`%s.GET("%s", %s.Index)`, group, route, names.VarName)
	case "show":
		return fmt.Sprintf(`%s.GET("%s/:id", %s.Show)`, group, route, names.VarName)
	case "create":
		return fmt.Sprintf(`%s.POST("%s", %s.Create)`, group, route, names.VarName)
	case "update":
		return fmt.Sprintf(`%s.PUT("%s/:id", %s.Update)`, group, route, names.VarName)
	case "destroy":
		return fmt.Sprintf(`%s.DELETE("%s/:id", %s.Destroy)`, group, route, names.VarName)
	default:
		return ""
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
