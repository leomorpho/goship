package generators

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type ControllerMakeOptions struct {
	Name    string
	Path    string
	Actions []string
	Auth    string
	Wire    bool
	Domain  string
}

type ControllerNames struct {
	BaseSnake string
	BaseKebab string
	BaseTitle string
	VarName   string
	TypeName  string
	FileName  string
}

type NormalizedDomainTarget struct {
	Name   string
	Snake  string
	Pascal string
}

type ControllerDeps struct {
	Out                    io.Writer
	Err                    io.Writer
	HasFile                func(path string) bool
	EnsureRouteNamesImport func(path string, dryRun bool) error
	WireRouteSnippet       func(path, auth, snippet string, dryRun bool) error
}

func RunMakeController(args []string, d ControllerDeps) int {
	opts, err := ParseMakeControllerArgs(args)
	if err != nil {
		fmt.Fprintf(d.Err, "invalid make:controller arguments: %v\n", err)
		return 1
	}
	names, err := NormalizeControllerName(opts.Name)
	if err != nil {
		fmt.Fprintf(d.Err, "invalid controller name: %v\n", err)
		return 1
	}

	controllerPath := filepath.Join(opts.Path, "web", "controllers", names.FileName)
	if d.HasFile(controllerPath) {
		fmt.Fprintf(d.Err, "refusing to overwrite existing controller file: %s\n", controllerPath)
		return 1
	}

	domain, err := NormalizeDomainTarget(opts.Domain)
	if err != nil {
		fmt.Fprintf(d.Err, "invalid --domain value: %v\n", err)
		return 1
	}

	content, err := RenderControllerFile(names, opts.Actions, domain)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to render controller: %v\n", err)
		return 1
	}
	if err := os.MkdirAll(filepath.Dir(controllerPath), 0o755); err != nil {
		fmt.Fprintf(d.Err, "failed to create controller directory: %v\n", err)
		return 1
	}
	if err := os.WriteFile(controllerPath, []byte(content), 0o644); err != nil {
		fmt.Fprintf(d.Err, "failed to write controller file: %v\n", err)
		return 1
	}

	fmt.Fprintf(d.Out, "Generated controller: %s\n", controllerPath)
	routeSnippet := RenderControllerRouteSnippet(names, opts.Actions, opts.Auth, domain.Name != "")
	if opts.Wire {
		routerPath := filepath.Join(opts.Path, "router.go")
		if err := d.EnsureRouteNamesImport(routerPath, false); err != nil {
			fmt.Fprintf(d.Err, "failed to ensure routeNames import: %v\n", err)
			return 1
		}
		if err := d.WireRouteSnippet(routerPath, opts.Auth, routeSnippet, false); err != nil {
			fmt.Fprintf(d.Err, "failed to wire controller routes: %v\n", err)
			return 1
		}
		fmt.Fprintf(d.Out, "Wired routes into %s\n", routerPath)
	} else {
		fmt.Fprintln(d.Out, "Route snippet:")
		fmt.Fprintln(d.Out, routeSnippet)
	}
	return 0
}

func NormalizeDomainTarget(raw string) (NormalizedDomainTarget, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return NormalizedDomainTarget{}, nil
	}
	parts := splitWords(name)
	if len(parts) == 0 {
		return NormalizedDomainTarget{}, errors.New("domain must contain letters or numbers")
	}
	return NormalizedDomainTarget{
		Name:   name,
		Snake:  strings.Join(parts, "_"),
		Pascal: toPascalFromParts(parts),
	}, nil
}

func ParseMakeControllerArgs(args []string) (ControllerMakeOptions, error) {
	opts := ControllerMakeOptions{Path: "apps/site", Actions: []string{"index"}, Auth: "public"}
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

func NormalizeControllerName(raw string) (ControllerNames, error) {
	name := strings.TrimSpace(raw)
	name = strings.TrimSuffix(name, "Controller")
	parts := splitWords(name)
	if len(parts) == 0 {
		return ControllerNames{}, errors.New("name cannot be empty")
	}
	baseSnake := strings.Join(parts, "_")
	baseKebab := strings.Join(parts, "-")
	baseTitle := toPascalFromParts(parts)
	varName := toLowerCamel(baseTitle)
	return ControllerNames{BaseSnake: baseSnake, BaseKebab: baseKebab, BaseTitle: baseTitle, VarName: varName, TypeName: baseTitle + "Controller", FileName: baseSnake + ".go"}, nil
}

func RenderControllerFile(names ControllerNames, actions []string, domain NormalizedDomainTarget) (string, error) {
	var b strings.Builder
	b.WriteString("package controllers\n\n")
	b.WriteString("import (\n\t\"net/http\"\n\n\t\"github.com/labstack/echo/v4\"\n)\n\n")
	b.WriteString("type " + names.VarName + " struct {\n")
	if domain.Name != "" {
		b.WriteString("\tdomainService any\n")
	}
	b.WriteString("}\n\n")
	b.WriteString("func New" + names.BaseTitle + "Controller(")
	if domain.Name != "" {
		b.WriteString("domainService any")
	}
	b.WriteString(") *")
	if domain.Name != "" {
		b.WriteString(names.VarName + " {\n\treturn &" + names.VarName + "{domainService: domainService}\n}\n\n")
	} else {
		b.WriteString(names.VarName + " {\n\treturn &" + names.VarName + "{}\n}\n\n")
	}
	for _, action := range actions {
		methodName := actionMethodName(action)
		b.WriteString("func (c *" + names.VarName + ") " + methodName + "(ctx echo.Context) error {\n")
		if domain.Name != "" {
			b.WriteString("\t// TODO: delegate to domain service in apps/site/app/" + domain.Snake + "\n")
		}
		b.WriteString("\treturn ctx.String(http.StatusNotImplemented, \"TODO: " + names.BaseTitle + "." + methodName + "\")\n}\n\n")
	}
	src, err := format.Source([]byte(b.String()))
	if err != nil {
		return "", err
	}
	return string(src), nil
}

func RenderControllerRouteSnippet(names ControllerNames, actions []string, auth string, withDomain bool) string {
	var b bytes.Buffer
	b.WriteString("\t// ship:generated:" + names.BaseSnake + "\n")
	b.WriteString("\t" + names.VarName + " := controllers.New" + names.BaseTitle + "Controller(")
	if withDomain {
		b.WriteString("nil")
	}
	b.WriteString(")\n")
	for _, action := range actions {
		line := actionRouteLine(names, action, auth)
		if line != "" {
			b.WriteString("\t" + line + "\n")
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

func actionRouteLine(names ControllerNames, action, auth string) string {
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
