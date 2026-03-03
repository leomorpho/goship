package ship

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type resourceGenerateOptions struct {
	Name   string
	Path   string
	Auth   string
	Views  string
	Wire   bool
	DryRun bool
}

type resourceGenerateResult struct {
	CreatedFiles       []string
	RouterPath         string
	RouteSnippet       string
	RouteInsertSnippet string
	RouteNamePath      string
	RouteNameConst     string
	RouteNameValue     string
}

func (c CLI) runGenerateResource(args []string) int {
	parsed, err := parseGenerateResourceArgs(args)
	if err != nil {
		fmt.Fprintf(c.Err, "invalid generate resource arguments: %v\n", err)
		return 1
	}
	if strings.TrimSpace(parsed.Name) == "" {
		fmt.Fprintln(c.Err, "usage: ship generate resource <name> [--path app/goship] [--auth public|auth] [--views templ|none] [--wire] [--dry-run]")
		return 1
	}

	result, err := generateResourceScaffold(parsed)
	if err != nil {
		fmt.Fprintf(c.Err, "failed to generate resource: %v\n", err)
		return 1
	}

	if parsed.Wire {
		if err := ensureRouteNamesImport(result.RouterPath, parsed.DryRun); err != nil {
			fmt.Fprintf(c.Err, "failed to ensure routeNames import: %v\n", err)
			return 1
		}
		if err := wireRouteSnippet(result.RouterPath, parsed.Auth, result.RouteInsertSnippet, parsed.DryRun); err != nil {
			fmt.Fprintf(c.Err, "failed to wire generated route: %v\n", err)
			return 1
		}
	}
	if err := wireRouteNameConstant(result.RouteNamePath, result.RouteNameConst, result.RouteNameValue, parsed.DryRun); err != nil {
		fmt.Fprintf(c.Err, "failed to wire route name constant: %v\n", err)
		return 1
	}

	fmt.Fprintln(c.Out, "Generated files:")
	for _, f := range result.CreatedFiles {
		fmt.Fprintf(c.Out, "- %s\n", f)
	}
	fmt.Fprintln(c.Out)
	if parsed.DryRun {
		fmt.Fprintln(c.Out, "Dry-run mode: no files were written.")
		fmt.Fprintf(c.Out, "Would update route names in %s:\n", result.RouteNamePath)
		fmt.Fprintf(c.Out, "- %s = %q\n\n", result.RouteNameConst, result.RouteNameValue)
	}
	if parsed.Wire {
		fmt.Fprintf(c.Out, "Wired route snippet into %s behind ship markers.\n", result.RouterPath)
	} else {
		fmt.Fprintf(c.Out, "Update %s with this snippet:\n\n", result.RouterPath)
		fmt.Fprintln(c.Out, result.RouteSnippet)
	}
	fmt.Fprintln(c.Out)
	if !parsed.DryRun {
		fmt.Fprintf(c.Out, "Route name constant ensured in %s.\n", result.RouteNamePath)
	}

	return 0
}

func parseGenerateResourceArgs(args []string) (resourceGenerateOptions, error) {
	opts := resourceGenerateOptions{
		Path:  "app/goship",
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
		case arg == "--dry-run":
			opts.DryRun = true
		case strings.HasPrefix(arg, "--path="):
			opts.Path = strings.TrimPrefix(arg, "--path=")
		case strings.HasPrefix(arg, "--auth="):
			opts.Auth = strings.TrimPrefix(arg, "--auth=")
		case strings.HasPrefix(arg, "--views="):
			opts.Views = strings.TrimPrefix(arg, "--views=")
		case arg == "--path" || arg == "--auth" || arg == "--views":
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
	return opts, nil
}

func generateResourceScaffold(opts resourceGenerateOptions) (resourceGenerateResult, error) {
	var result resourceGenerateResult

	if strings.TrimSpace(opts.Path) == "" {
		return result, errors.New("path cannot be empty")
	}
	if opts.Auth != "public" && opts.Auth != "auth" {
		return result, fmt.Errorf("invalid --auth value %q: expected public or auth", opts.Auth)
	}
	if opts.Views != "templ" && opts.Views != "none" {
		return result, fmt.Errorf("invalid --views value %q: expected templ or none", opts.Views)
	}

	norm, err := normalizeResourceName(opts.Name)
	if err != nil {
		return result, err
	}

	handlerDir := filepath.Join(opts.Path, "web", "routes")
	handlerFile := filepath.Join(handlerDir, norm.Snake+".go")
	if err := writeFile(handlerFile, renderResourceHandler(norm, opts.Views), opts.DryRun); err != nil {
		return result, err
	}
	result.CreatedFiles = append(result.CreatedFiles, handlerFile)

	if opts.Views == "templ" {
		viewDir := filepath.Join(opts.Path, "views", "web", "pages")
		viewFile := filepath.Join(viewDir, norm.Snake+".templ")
		if err := writeFile(viewFile, renderResourceTempl(norm), opts.DryRun); err != nil {
			return result, err
		}
		result.CreatedFiles = append(result.CreatedFiles, viewFile)
	}

	result.RouterPath = filepath.Join(opts.Path, "router.go")
	result.RouteSnippet = renderRouteSnippet(norm, opts.Auth)
	result.RouteInsertSnippet = renderRouteInsertSnippet(norm, opts.Auth)
	result.RouteNamePath = filepath.Join("pkg", "routing", "routenames", "routenames.go")
	result.RouteNameConst = "RouteName" + norm.Pascal
	result.RouteNameValue = norm.Snake
	return result, nil
}

type normalizedResourceName struct {
	Snake      string
	Kebab      string
	Pascal     string
	LowerCamel string
}

func normalizeResourceName(raw string) (normalizedResourceName, error) {
	var out normalizedResourceName
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

func renderResourceHandler(n normalizedResourceName, views string) string {
	if views == "templ" {
		return renderResourceTemplHandler(n)
	}
	return renderResourceBasicHandler(n)
}

func renderResourceBasicHandler(n normalizedResourceName) string {
	return fmt.Sprintf(`package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/pkg/controller"
)

type %s struct {
	ctr controller.Controller
}

func New%sRoute(ctr controller.Controller) *%s {
	return &%s{ctr: ctr}
}

func (r *%s) Get(ctx echo.Context) error {
	// TODO: Replace with templ/page rendering or real handler logic.
	return ctx.String(http.StatusOK, "%s resource")
}
`, n.LowerCamel, n.Pascal, n.LowerCamel, n.LowerCamel, n.LowerCamel, n.Kebab)
}

func renderResourceTemplHandler(n normalizedResourceName) string {
	return fmt.Sprintf(`package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/goship/views"
	"github.com/leomorpho/goship/app/goship/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/goship/views/web/pages/gen"
	"github.com/leomorpho/goship/pkg/controller"
)

type %s struct {
	ctr controller.Controller
}

func New%sRoute(ctr controller.Controller) *%s {
	return &%s{ctr: ctr}
}

func (r *%s) Get(ctx echo.Context) error {
	page := controller.NewPage(ctx)
	page.Layout = layouts.Main
	page.Name = templates.Page("%s")
	page.Title = "%s"
	page.Component = pages.%sPage(&page)
	page.HTMX.Request.Boosted = true

	return r.ctr.RenderPage(ctx, page)
}
`, n.LowerCamel, n.Pascal, n.LowerCamel, n.LowerCamel, n.LowerCamel, n.Kebab, n.Pascal, n.Pascal)
}

func renderResourceTempl(n normalizedResourceName) string {
	return fmt.Sprintf(`package pages

import "github.com/leomorpho/goship/pkg/controller"

templ %sPage(page *controller.Page) {
	<section>
		<h1>%s</h1>
		<p>TODO: implement %s page.</p>
	</section>
}
`, n.Pascal, n.Pascal, n.Kebab)
}

func renderRouteSnippet(n normalizedResourceName, auth string) string {
	targetFn := "registerPublicRoutes"
	if auth == "auth" {
		targetFn = "registerAuthRoutes"
	}

	return fmt.Sprintf(`// In %s:
%s`, targetFn, strings.TrimSpace(renderRouteInsertSnippet(n, auth)))
}

func renderRouteInsertSnippet(n normalizedResourceName, auth string) string {
	targetGroup := "g"
	if auth == "auth" {
		targetGroup = "onboardedGroup"
	}

	return fmt.Sprintf(`	// ship:generated:%s
	%s := routes.New%sRoute(ctr)
	%s.GET("/%s", %s.Get).Name = routeNames.RouteName%s
`, n.Snake, n.LowerCamel, n.Pascal, targetGroup, n.Kebab, n.LowerCamel, n.Pascal)
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

func wireRouteSnippet(routerPath, auth, snippet string, dryRun bool) error {
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

func wireRouteNameConstant(routeNamesPath, constName, constValue string, dryRun bool) error {
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
	constEnd := strings.Index(content[constStart:], "\n)")
	if constEnd == -1 {
		return fmt.Errorf("const block closing not found in %s", routeNamesPath)
	}
	constEnd += constStart
	line := fmt.Sprintf("\t%s = %q\n", constName, constValue)
	updated := content[:constEnd] + line + content[constEnd:]
	if dryRun {
		return nil
	}
	return os.WriteFile(routeNamesPath, []byte(updated), 0o644)
}

func ensureRouteNamesImport(routerPath string, dryRun bool) error {
	b, err := os.ReadFile(routerPath)
	if err != nil {
		return err
	}
	content := string(b)
	if strings.Contains(content, `routeNames "github.com/leomorpho/goship/pkg/routing/routenames"`) {
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

	line := "\trouteNames \"github.com/leomorpho/goship/pkg/routing/routenames\"\n"
	updated := content[:importEnd] + line + content[importEnd:]
	if dryRun {
		return nil
	}
	return os.WriteFile(routerPath, []byte(updated), 0o644)
}
