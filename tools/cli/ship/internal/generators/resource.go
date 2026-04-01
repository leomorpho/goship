package generators

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type ResourceGenerateOptions struct {
	Name      string
	Path      string
	Auth      string
	Views     string
	Wire      bool
	DryRun    bool
	Domain    string
	TestFirst bool
}

type ResourceGenerateResult struct {
	CreatedFiles       []string
	RouterPath         string
	RouteSnippet       string
	RouteInsertSnippet string
	RouteNamePath      string
	RouteNameConst     string
	RouteNameValue     string
}

func RunGenerateResource(args []string, out io.Writer, errOut io.Writer) int {
	parsed, err := ParseGenerateResourceArgs(args)
	if err != nil {
		fmt.Fprintf(errOut, "invalid generate resource arguments: %v\n", err)
		return 1
	}
	if strings.TrimSpace(parsed.Name) == "" {
		fmt.Fprintln(errOut, "usage: ship make:resource <name> [--path app] [--auth public|auth] [--views templ|none] [--domain <name>] [--wire] [--dry-run]")
		return 1
	}

	result, err := GenerateResourceScaffold(parsed)
	if err != nil {
		fmt.Fprintf(errOut, "failed to generate resource: %v\n", err)
		return 1
	}

	if parsed.Wire {
		if err := EnsureRouteNamesImport(result.RouterPath, parsed.DryRun); err != nil {
			fmt.Fprintf(errOut, "failed to ensure routeNames import: %v\n", err)
			return 1
		}
		if err := WireRouteSnippet(result.RouterPath, parsed.Auth, result.RouteInsertSnippet, parsed.DryRun); err != nil {
			fmt.Fprintf(errOut, "failed to wire generated route: %v\n", err)
			return 1
		}
	}
	if err := WireRouteNameConstant(result.RouteNamePath, result.RouteNameConst, result.RouteNameValue, parsed.DryRun); err != nil {
		fmt.Fprintf(errOut, "failed to wire route name constant: %v\n", err)
		return 1
	}

	updated := []string{result.RouteNamePath}
	previews := make([]generatorPreview, 0, 2)
	if parsed.Wire {
		if !parsed.DryRun {
			updated = append(updated, result.RouterPath)
		}
		previews = append(previews, generatorPreview{
			Title: "Router snippet for " + result.RouterPath,
			Body:  result.RouteSnippet,
		})
	} else {
		previews = append(previews, generatorPreview{
			Title: "Router snippet for " + result.RouterPath,
			Body:  result.RouteSnippet,
		})
	}
	previews = append(previews, generatorPreview{
		Title: "Route name constant for " + result.RouteNamePath,
		Body:  fmt.Sprintf("%s = %q", result.RouteNameConst, result.RouteNameValue),
	})
	if parsed.DryRun {
		updated = nil
	}
	writeGeneratorReport(out, "resource", parsed.DryRun, result.CreatedFiles, updated, previews, nil)

	return 0
}

func ParseGenerateResourceArgs(args []string) (ResourceGenerateOptions, error) {
	opts := ResourceGenerateOptions{
		Path:  "app",
		Auth:  "public",
		Views: "templ",
	}

	var positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			positionals = append(positionals, arg)
			continue
		}

		switch {
		case arg == "--wire":
			opts.Wire = true
		case arg == "--test-first":
			opts.TestFirst = true
		case arg == "--dry-run":
			opts.DryRun = true
		case strings.HasPrefix(arg, "--path="):
			opts.Path = strings.TrimPrefix(arg, "--path=")
		case strings.HasPrefix(arg, "--auth="):
			opts.Auth = strings.TrimPrefix(arg, "--auth=")
		case strings.HasPrefix(arg, "--views="):
			opts.Views = strings.TrimPrefix(arg, "--views=")
		case strings.HasPrefix(arg, "--domain="):
			opts.Domain = strings.TrimPrefix(arg, "--domain=")
		case arg == "--path" || arg == "--auth" || arg == "--views" || arg == "--domain":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing value for %s", arg)
			}
			i++
			switch arg {
			case "--path":
				opts.Path = args[i]
			case "--auth":
				opts.Auth = args[i]
			case "--views":
				opts.Views = args[i]
			case "--domain":
				opts.Domain = args[i]
			}
		default:
			return opts, fmt.Errorf("unknown option: %s", arg)
		}
	}

	if len(positionals) > 1 {
		return opts, fmt.Errorf("unexpected positional arguments: %v", positionals[1:])
	}
	if len(positionals) == 1 {
		opts.Name = positionals[0]
	}
	normalizedPath, err := normalizeOwnedGeneratorPath(opts.Path, "app")
	if err != nil {
		return opts, err
	}
	opts.Path = normalizedPath
	return opts, nil
}

func GenerateResourceScaffold(opts ResourceGenerateOptions) (ResourceGenerateResult, error) {
	var result ResourceGenerateResult

	if strings.TrimSpace(opts.Path) == "" {
		return result, errors.New("path cannot be empty")
	}
	if opts.Auth != "public" && opts.Auth != "auth" {
		return result, fmt.Errorf("invalid --auth value %q: expected public or auth", opts.Auth)
	}
	if opts.Views != "templ" && opts.Views != "none" {
		return result, fmt.Errorf("invalid --views value %q: expected templ or none", opts.Views)
	}

	norm, err := NormalizeResourceName(opts.Name)
	if err != nil {
		return result, err
	}
	domain, err := NormalizeDomainTarget(opts.Domain)
	if err != nil {
		return result, fmt.Errorf("invalid --domain value: %w", err)
	}

	handlerDir := filepath.Join(opts.Path, "web", "controllers")
	handlerFile := filepath.Join(handlerDir, norm.Snake+".go")
	if err := writeFile(handlerFile, renderResourceHandler(norm, opts.Views, domain, opts.TestFirst), opts.DryRun); err != nil {
		return result, err
	}
	result.CreatedFiles = append(result.CreatedFiles, handlerFile)

	if opts.TestFirst {
		testFile := filepath.Join(handlerDir, norm.Snake+"_test.go")
		if err := writeFile(testFile, renderResourceTestFile(norm), opts.DryRun); err != nil {
			return result, err
		}
		result.CreatedFiles = append(result.CreatedFiles, testFile)
	}

	if opts.Views == "templ" {
		viewDir := filepath.Join(opts.Path, "views", "web", "pages")
		viewFile := filepath.Join(viewDir, norm.Snake+".templ")
		if err := writeFile(viewFile, renderResourceTempl(norm), opts.DryRun); err != nil {
			return result, err
		}
		result.CreatedFiles = append(result.CreatedFiles, viewFile)
	}

	result.RouterPath = filepath.Join(opts.Path, "router.go")
	result.RouteSnippet = renderRouteSnippet(norm, opts.Auth, domain.Name != "")
	result.RouteInsertSnippet = renderRouteInsertSnippet(norm, opts.Auth, domain.Name != "")
	result.RouteNamePath = filepath.Join(opts.Path, "web", "routenames", "routenames.go")
	result.RouteNameConst = "RouteName" + norm.Pascal
	result.RouteNameValue = norm.Snake
	return result, nil
}

type NormalizedResourceName struct {
	Snake      string
	Kebab      string
	Pascal     string
	LowerCamel string
}

func NormalizeResourceName(raw string) (NormalizedResourceName, error) {
	var out NormalizedResourceName
	tokens := tokenizeResourceName(raw)
	if len(tokens) == 0 {
		return out, errors.New("resource name must contain at least one letter or number")
	}

	out.Snake = strings.Join(tokens, "_")
	out.Kebab = strings.Join(tokens, "-")

	var pascalParts []string
	for _, token := range tokens {
		pascalParts = append(pascalParts, strings.ToUpper(token[:1])+token[1:])
	}
	out.Pascal = strings.Join(pascalParts, "")
	out.LowerCamel = strings.ToLower(out.Pascal[:1]) + out.Pascal[1:]
	return out, nil
}

func tokenizeResourceName(raw string) []string {
	var tokens []string
	var current []rune
	runes := []rune(strings.TrimSpace(raw))

	flush := func() {
		if len(current) == 0 {
			return
		}
		tokens = append(tokens, strings.ToLower(string(current)))
		current = current[:0]
	}

	for i, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush()
			continue
		}

		if unicode.IsUpper(r) && len(current) > 0 {
			prev := runes[i-1]
			var next rune
			if i+1 < len(runes) {
				next = runes[i+1]
			}
			if unicode.IsLower(prev) || (unicode.IsUpper(prev) && next != 0 && unicode.IsLower(next)) || unicode.IsDigit(prev) {
				flush()
			}
		}

		current = append(current, unicode.ToLower(r))
	}
	flush()
	return tokens
}

func renderResourceHandler(n NormalizedResourceName, views string, domain NormalizedDomainTarget, testFirst bool) string {
	if views == "templ" {
		return renderResourceTemplHandler(n, domain, testFirst)
	}
	return renderResourceBasicHandler(n, domain, testFirst)
}

func renderResourceBasicHandler(n NormalizedResourceName, domain NormalizedDomainTarget, testFirst bool) string {
	domainField := ""
	constructorArg := ""
	constructorAssign := ""
	domainComment := ""
	if domain.Name != "" {
		domainField = "\tdomainService any\n"
		constructorArg = ", domainService any"
		constructorAssign = ", domainService: domainService"
		domainComment = "\t// SCAFFOLD: delegate to domain service in app/" + domain.Snake + ".\n"
	}

	body := ""
	if testFirst {
		body = "\tpanic(\"not implemented\")"
	} else {
		body = fmt.Sprintf("%s\treturn ctx.String(http.StatusOK, \"%s resource\")", domainComment, n.Kebab)
	}

	return fmt.Sprintf(`package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/v2/framework/http/ui"
)

type %s struct {
	ctr ui.Controller
%s}

func New%sRoute(ctr ui.Controller%s) *%s {
	return &%s{ctr: ctr%s}
}

func (r *%s) Get(ctx echo.Context) error {
%s
}
`, n.LowerCamel, domainField, n.Pascal, constructorArg, n.LowerCamel, n.LowerCamel, constructorAssign, n.LowerCamel, body)
}

func renderResourceTemplHandler(n NormalizedResourceName, domain NormalizedDomainTarget, testFirst bool) string {
	domainField := ""
	constructorArg := ""
	constructorAssign := ""
	domainComment := ""
	if domain.Name != "" {
		domainField = "\tdomainService any\n"
		constructorArg = ", domainService any"
		constructorAssign = ", domainService: domainService"
		domainComment = "\t// SCAFFOLD: delegate to domain service in app/" + domain.Snake + ".\n"
	}

	imports := ""
	if testFirst {
		imports = `import (
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/v2/framework/http/ui"
)`
	} else {
		imports = `import (
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/v2/framework/http/templates"
	layouts "github.com/leomorpho/goship/v2/framework/http/layouts/gen"
	pages "github.com/leomorpho/goship/v2/framework/http/pages/gen"
	"github.com/leomorpho/goship/v2/framework/http/ui"
)`
	}

	body := ""
	if testFirst {
		body = "\tpanic(\"not implemented\")"
	} else {
		body = fmt.Sprintf(`%s	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Name = templates.Page("%s")
	page.Title = "%s"
	page.Component = pages.%sPage(&page)
	page.HTMX.Request.Boosted = true

	return r.ctr.RenderPage(ctx, page)`, domainComment, n.Kebab, n.Pascal, n.Pascal)
	}

	return fmt.Sprintf(`package controllers

%s

type %s struct {
	ctr ui.Controller
%s}

func New%sRoute(ctr ui.Controller%s) *%s {
	return &%s{ctr: ctr%s}
}

func (r *%s) Get(ctx echo.Context) error {
%s
}
`, imports, n.LowerCamel, domainField, n.Pascal, constructorArg, n.LowerCamel, n.LowerCamel, constructorAssign, n.LowerCamel, body)
}

func renderResourceTestFile(n NormalizedResourceName) string {
	return fmt.Sprintf(`package controllers

import (
	"testing"
)

func Test%sRoute_Get(t *testing.T) {
	// SCAFFOLD: implement %s show — should return 200 with %s details
	t.Skip("scaffold: implement me")
}
`, n.Pascal, n.Pascal, n.Snake)
}

func renderResourceTempl(n NormalizedResourceName) string {
	return fmt.Sprintf(`package pages

import "github.com/leomorpho/goship/v2/framework/http/ui"

templ %sPage(page *ui.Page) {
	<section>
		<h1>%s</h1>
		<p>Scaffold page for %s. Replace with your real UI.</p>
	</section>
}
`, n.Pascal, n.Pascal, n.Kebab)
}

func renderRouteSnippet(n NormalizedResourceName, auth string, withDomain bool) string {
	targetFn := "registerPublicRoutes"
	if auth == "auth" {
		targetFn = "registerAuthRoutes"
	}

	return fmt.Sprintf(`// In %s:
%s`, targetFn, strings.TrimSpace(renderRouteInsertSnippet(n, auth, withDomain)))
}

func renderRouteInsertSnippet(n NormalizedResourceName, auth string, withDomain bool) string {
	targetGroup := "g"
	if auth == "auth" {
		targetGroup = "onboardedGroup"
	}
	constructorArg := ""
	if withDomain {
		constructorArg = ", nil"
	}

	return fmt.Sprintf(`	// ship:generated:%s
	%s := controllers.New%sRoute(ctr%s)
	%s.GET("/%s", %s.Get).Name = routeNames.RouteName%s
`, n.Snake, n.LowerCamel, n.Pascal, constructorArg, targetGroup, n.Kebab, n.LowerCamel, n.Pascal)
}

func writeFile(path string, content string, dryRun bool) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("refusing to overwrite existing file: %s", path)
	}
	if dryRun {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	return nil
}

func WireRouteSnippet(routerPath, auth, snippet string, dryRun bool) error {
	startMarker, endMarker, err := routeMarkerPair(auth)
	if err != nil {
		return err
	}

	b, err := os.ReadFile(routerPath)
	if err != nil {
		return err
	}
	content := string(b)

	startIdx := strings.Index(content, startMarker)
	endIdx := strings.Index(content, endMarker)
	if startIdx == -1 || endIdx == -1 {
		return fmt.Errorf("required route markers not found for %q in %s", auth, routerPath)
	}
	if endIdx <= startIdx {
		return fmt.Errorf("invalid marker ordering for %q in %s", auth, routerPath)
	}

	insertPos := endIdx
	block := content[startIdx:endIdx]
	trimmedSnippet := strings.TrimSpace(snippet)
	if strings.Contains(block, trimmedSnippet) {
		return nil
	}

	var insert bytes.Buffer
	if !strings.HasSuffix(block, "\n") {
		insert.WriteString("\n")
	}
	insert.WriteString(snippet)
	if !strings.HasSuffix(snippet, "\n") {
		insert.WriteString("\n")
	}

	updated := content[:insertPos] + insert.String() + content[insertPos:]
	if dryRun {
		return nil
	}
	return os.WriteFile(routerPath, []byte(updated), 0o644)
}

func routeMarkerPair(auth string) (string, string, error) {
	switch auth {
	case "public":
		return "// ship:routes:public:start", "// ship:routes:public:end", nil
	case "auth":
		return "// ship:routes:auth:start", "// ship:routes:auth:end", nil
	default:
		return "", "", fmt.Errorf("unknown auth group %q", auth)
	}
}

func WireRouteNameConstant(routeNamesPath, constName, constValue string, dryRun bool) error {
	b, err := os.ReadFile(routeNamesPath)
	if err != nil {
		return err
	}
	content := string(b)
	if strings.Contains(content, constName+" ") || strings.Contains(content, constName+"\t") {
		return nil
	}
	constStart := strings.Index(content, "const (")
	if constStart == -1 {
		return fmt.Errorf("const block not found in %s", routeNamesPath)
	}
	constEnd := strings.LastIndex(content, "\n)")
	if constEnd == -1 {
		return fmt.Errorf("const block closing not found in %s", routeNamesPath)
	}
	line := fmt.Sprintf("\t%s = %q\n", constName, constValue)
	updated := content[:constEnd+1] + line + content[constEnd+1:]
	if dryRun {
		return nil
	}
	return os.WriteFile(routeNamesPath, []byte(updated), 0o644)
}

func EnsureRouteNamesImport(routerPath string, dryRun bool) error {
	b, err := os.ReadFile(routerPath)
	if err != nil {
		return err
	}
	content := string(b)
	if strings.Contains(content, `routeNames "github.com/leomorpho/goship/v2/framework/http/routenames"`) {
		return nil
	}

	importStart := strings.Index(content, "import (\n")
	if importStart == -1 {
		return fmt.Errorf("import block not found in %s", routerPath)
	}
	importEnd := strings.Index(content[importStart:], "\n)")
	if importEnd == -1 {
		return fmt.Errorf("import block closing not found in %s", routerPath)
	}
	importEnd += importStart

	line := "\trouteNames \"github.com/leomorpho/goship/v2/framework/http/routenames\"\n"
	updated := content[:importEnd] + line + content[importEnd:]
	if dryRun {
		return nil
	}
	return os.WriteFile(routerPath, []byte(updated), 0o644)
}
