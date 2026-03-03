package ship

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type resourceGenerateOptions struct {
	Name  string
	Path  string
	Auth  string
	Views string
	Wire  bool
}

type resourceGenerateResult struct {
	CreatedFiles       []string
	RouterPath         string
	RouteSnippet       string
	RouteInsertSnippet string
}

func (c CLI) runGenerateResource(args []string) int {
	fs := flag.NewFlagSet("generate resource", flag.ContinueOnError)
	fs.SetOutput(c.Err)
	basePath := fs.String("path", "app/goship", "app path containing router.go (for example app/goship)")
	auth := fs.String("auth", "public", "route group target: public or auth")
	views := fs.String("views", "templ", "view scaffold mode: templ or none")
	wire := fs.Bool("wire", false, "insert generated route snippet into router.go behind ship markers")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(c.Err, "invalid generate resource arguments: %v\n", err)
		return 1
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(c.Err, "usage: ship generate resource <name> [--path app/goship] [--auth public|auth] [--views templ|none] [--wire]")
		return 1
	}

	result, err := generateResourceScaffold(resourceGenerateOptions{
		Name:  fs.Arg(0),
		Path:  *basePath,
		Auth:  *auth,
		Views: *views,
		Wire:  *wire,
	})
	if err != nil {
		fmt.Fprintf(c.Err, "failed to generate resource: %v\n", err)
		return 1
	}

	if *wire {
		if err := wireRouteSnippet(result.RouterPath, *auth, result.RouteInsertSnippet); err != nil {
			fmt.Fprintf(c.Err, "failed to wire generated route: %v\n", err)
			return 1
		}
	}

	fmt.Fprintln(c.Out, "Generated files:")
	for _, f := range result.CreatedFiles {
		fmt.Fprintf(c.Out, "- %s\n", f)
	}
	fmt.Fprintln(c.Out)
	if *wire {
		fmt.Fprintf(c.Out, "Wired route snippet into %s behind ship markers.\n", result.RouterPath)
	} else {
		fmt.Fprintf(c.Out, "Update %s with this snippet:\n\n", result.RouterPath)
		fmt.Fprintln(c.Out, result.RouteSnippet)
	}
	fmt.Fprintln(c.Out)
	fmt.Fprintln(c.Out, "Also add a route name constant in pkg/routing/routenames.")

	return 0
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
	if err := os.MkdirAll(handlerDir, 0o755); err != nil {
		return result, err
	}
	handlerFile := filepath.Join(handlerDir, norm.Snake+".go")
	if err := writeFileIfMissing(handlerFile, renderResourceHandler(norm)); err != nil {
		return result, err
	}
	result.CreatedFiles = append(result.CreatedFiles, handlerFile)

	if opts.Views == "templ" {
		viewDir := filepath.Join(opts.Path, "views", "web", "pages")
		if err := os.MkdirAll(viewDir, 0o755); err != nil {
			return result, err
		}
		viewFile := filepath.Join(viewDir, norm.Snake+".templ")
		if err := writeFileIfMissing(viewFile, renderResourceTempl(norm)); err != nil {
			return result, err
		}
		result.CreatedFiles = append(result.CreatedFiles, viewFile)
	}

	result.RouterPath = filepath.Join(opts.Path, "router.go")
	result.RouteSnippet = renderRouteSnippet(norm, opts.Auth)
	result.RouteInsertSnippet = renderRouteInsertSnippet(norm, opts.Auth)
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

func renderResourceHandler(n normalizedResourceName) string {
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

func renderResourceTempl(n normalizedResourceName) string {
	return fmt.Sprintf(`package pages

templ %sPage() {
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

func writeFileIfMissing(path string, content string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("refusing to overwrite existing file: %s", path)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	return nil
}

func wireRouteSnippet(routerPath, auth, snippet string) error {
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
