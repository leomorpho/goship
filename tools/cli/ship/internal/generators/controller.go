package generators

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type ControllerMakeOptions struct {
	Name      string
	Path      string
	Actions   []string
	Auth      string
	Wire      bool
	Domain    string
	TestFirst bool
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
	capabilities := CapabilityModelForRoot(".")
	if capabilities.Workspace == GeneratorWorkspaceStarterScaffold {
		return runMakeStarterController(opts, names, d)
	}
	if !capabilities.SupportsControllerGeneration {
		fmt.Fprintln(d.Err, "make:controller is not supported on the starter scaffold yet; no files were changed")
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

	content, err := RenderControllerFile(names, opts.Actions, domain, opts.TestFirst)
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

	created := []string{controllerPath}

	if opts.TestFirst {
		testPath := filepath.Join(opts.Path, "web", "controllers", strings.TrimSuffix(names.FileName, ".go")+"_test.go")
		if !d.HasFile(testPath) {
			testContent, err := RenderControllerTestFile(names, opts.Actions)
			if err != nil {
				fmt.Fprintf(d.Err, "failed to render controller test: %v\n", err)
				return 1
			}
			if err := os.WriteFile(testPath, []byte(testContent), 0o644); err != nil {
				fmt.Fprintf(d.Err, "failed to write controller test file: %v\n", err)
				return 1
			}
			created = append(created, testPath)
		}
	}
	routeSnippet := RenderControllerRouteSnippet(names, opts.Actions, opts.Auth, domain.Name != "")
	updated := make([]string, 0, 1)
	previews := make([]generatorPreview, 0, 1)
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
		updated = append(updated, routerPath)
	} else {
		previews = append(previews, generatorPreview{
			Title: "Router snippet for " + filepath.Join(opts.Path, "router.go"),
			Body:  routeSnippet,
		})
	}
	writeGeneratorReport(d.Out, "controller", false, created, updated, previews, nil)
	return 0
}

func runMakeStarterController(opts ControllerMakeOptions, names ControllerNames, d ControllerDeps) int {
	routePath := "/" + names.BaseKebab
	if opts.Auth == "auth" {
		routePath = "/auth/" + names.BaseKebab
	}
	spec := StarterGeneratedRouteSpec{
		Snake:       names.BaseSnake,
		Kebab:       names.BaseKebab,
		Pascal:      names.BaseTitle,
		RoutePath:   routePath,
		Actions:     append([]string(nil), opts.Actions...),
		Description: fmt.Sprintf("Starter controller scaffold for %s with actions: %s.", names.BaseKebab, strings.Join(opts.Actions, ", ")),
	}
	pageFile := filepath.Join(opts.Path, "views", "web", "pages", "gen", names.BaseSnake+".go")
	if err := writeFile(pageFile, renderStarterGeneratedPageSpec(spec), false); err != nil {
		fmt.Fprintf(d.Err, "failed to write starter controller page: %v\n", err)
		return 1
	}
	routerPath := filepath.Join(opts.Path, "router.go")
	routeNamePath := filepath.Join(opts.Path, "web", "routenames", "routenames.go")
	routeNameConst := "RouteName" + names.BaseTitle
	routeNameValue := names.BaseSnake
	templatesPath := filepath.Join(opts.Path, "views", "templates.go")
	mainPath := filepath.Join("cmd", "web", "main.go")
	if opts.Wire {
		if err := EnsureRouteNamesImport(routerPath, false); err != nil {
			fmt.Fprintf(d.Err, "failed to ensure routeNames import: %v\n", err)
			return 1
		}
		if err := WireRouteSnippet(routerPath, opts.Auth, renderStarterRouteInsertSnippetForSpec(spec), false); err != nil {
			fmt.Fprintf(d.Err, "failed to wire starter controller routes: %v\n", err)
			return 1
		}
	}
	if err := WireRouteNameConstant(routeNamePath, routeNameConst, routeNameValue, false); err != nil {
		fmt.Fprintf(d.Err, "failed to wire route name constant: %v\n", err)
		return 1
	}
	if opts.Wire {
		if err := WireStarterPageConstant(templatesPath, "Page"+NormalizePageName(routeNameConst), routeNameValue, false); err != nil {
			fmt.Fprintf(d.Err, "failed to wire starter page constant: %v\n", err)
			return 1
		}
		if err := WireStarterComponentForPage(mainPath, routeNameValue, "Page"+NormalizePageName(routeNameConst), NormalizePageName(routeNameConst), titleFromRouteName(routeNameConst), false); err != nil {
			fmt.Fprintf(d.Err, "failed to wire starter component switch: %v\n", err)
			return 1
		}
	}
	updated := []string{routeNamePath}
	if opts.Wire {
		updated = append(updated, routerPath, templatesPath, mainPath)
	}
	writeGeneratorReport(d.Out, "controller", false, []string{pageFile}, updated, []generatorPreview{
		{
			Title: "Router snippet for " + routerPath,
			Body:  renderStarterRoutePreview(spec, opts.Auth),
		},
		{
			Title: "Route name constant for " + routeNamePath,
			Body:  fmt.Sprintf("%s = %q", routeNameConst, routeNameValue),
		},
	}, []string{
		"Starter controller scaffolds use the starter CRUD/runtime route backend rather than framework Echo controllers.",
	})
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
	opts := ControllerMakeOptions{Path: "app", Actions: []string{"index"}, Auth: "public"}
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
		case args[i] == "--test-first":
			opts.TestFirst = true
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
	normalizedPath, err := normalizeOwnedGeneratorPath(opts.Path, "app")
	if err != nil {
		return opts, err
	}
	opts.Path = normalizedPath
	return opts, nil
}

func parseControllerActions(raw string) ([]string, error) {
	return ParseGeneratorCRUDActionNames(raw)
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

func RenderControllerFile(names ControllerNames, actions []string, domain NormalizedDomainTarget, testFirst bool) (string, error) {
	var b strings.Builder
	b.WriteString("package controllers\n\n")
	b.WriteString("import (\n")
	if !testFirst {
		b.WriteString("\t\"net/http\"\n")
	}
	b.WriteString("\t\"github.com/labstack/echo/v4\"\n)\n\n")
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
		if testFirst {
			b.WriteString("\tpanic(\"not implemented\")\n}\n\n")
		} else {
			if domain.Name != "" {
				b.WriteString("\t// SCAFFOLD: delegate to domain service in app/" + domain.Snake + ".\n")
			}
			b.WriteString("\treturn ctx.String(http.StatusNotImplemented, \"" + names.BaseTitle + "." + methodName + " scaffold\")\n}\n\n")
		}
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

func RenderControllerTestFile(names ControllerNames, actions []string) (string, error) {
	var b strings.Builder
	b.WriteString("package controllers\n\n")
	b.WriteString("import (\n\t\"testing\"\n)\n\n")
	for _, action := range actions {
		methodName := actionMethodName(action)
		b.WriteString("func Test" + names.BaseTitle + "Controller_" + methodName + "(t *testing.T) {\n")
		comment := ""
		switch action {
		case "index":
			comment = "// SCAFFOLD: implement " + names.BaseTitle + " index — should return 200 with list of " + names.BaseSnake
		case "show":
			comment = "// SCAFFOLD: implement " + names.BaseTitle + " show — should return 200 with " + names.BaseSnake + " details"
		case "create":
			comment = "// SCAFFOLD: implement " + names.BaseTitle + " create — should return 200 with create form"
		case "update":
			comment = "// SCAFFOLD: implement " + names.BaseTitle + " update — PUT with valid data returns 302 redirect"
		case "destroy":
			comment = "// SCAFFOLD: implement " + names.BaseTitle + " destroy — DELETE returns 302 redirect"
		}
		if comment != "" {
			b.WriteString("\t" + comment + "\n")
		}
		b.WriteString("\tt.Skip(\"scaffold: implement me\")\n}\n\n")
	}
	src, err := format.Source([]byte(b.String()))
	if err != nil {
		return "", err
	}
	return string(src), nil
}
