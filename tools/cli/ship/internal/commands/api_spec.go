package commands

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"
)

type APIDeps struct {
	Out          io.Writer
	Err          io.Writer
	FindGoModule func(start string) (string, string, error)
}

type openAPISpec struct {
	OpenAPI    string                      `json:"openapi"`
	Info       openAPIInfo                 `json:"info"`
	Servers    []openAPIServer             `json:"servers"`
	Security   []map[string][]string       `json:"security,omitempty"`
	Paths      map[string]openAPIPathItem  `json:"paths"`
	Components openAPIComponents           `json:"components"`
}

type openAPIInfo struct {
	Title   string `json:"title"`
	Version string `json:"version"`
	License openAPILicense `json:"license"`
}

type openAPILicense struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type openAPIServer struct {
	URL string `json:"url"`
}

type openAPIComponents struct {
	Schemas         map[string]openAPISchema        `json:"schemas"`
	SecuritySchemes map[string]openAPISecurityScheme `json:"securitySchemes,omitempty"`
}

type openAPISecurityScheme struct {
	Type string `json:"type"`
	In   string `json:"in,omitempty"`
	Name string `json:"name,omitempty"`
}

type openAPIPathItem map[string]openAPIOperation

type openAPIOperation struct {
	OperationID string                      `json:"operationId"`
	Summary     string                      `json:"summary"`
	Parameters  []openAPIParameter          `json:"parameters,omitempty"`
	RequestBody *openAPIRequestBody         `json:"requestBody,omitempty"`
	Responses   map[string]openAPIResponse  `json:"responses"`
}

type openAPIParameter struct {
	Name     string        `json:"name"`
	In       string        `json:"in"`
	Required bool          `json:"required"`
	Schema   openAPISchema `json:"schema"`
}

type openAPIRequestBody struct {
	Required bool                        `json:"required"`
	Content  map[string]openAPIMediaType `json:"content"`
}

type openAPIResponse struct {
	Description string                      `json:"description"`
	Content     map[string]openAPIMediaType `json:"content,omitempty"`
}

type openAPIMediaType struct {
	Schema openAPISchema `json:"schema"`
}

type openAPISchema struct {
	Ref        string                   `json:"$ref,omitempty"`
	Type       string                   `json:"type,omitempty"`
	Format     string                   `json:"format,omitempty"`
	Properties map[string]openAPISchema `json:"properties,omitempty"`
	Items      *openAPISchema           `json:"items,omitempty"`
	Required   []string                 `json:"required,omitempty"`
}

type contractRoute struct {
	Name   string
	Method string
	Path   string
}

var routeCommentRe = regexp.MustCompile(`^Route:\s*(GET|POST|PUT|PATCH|DELETE)\s+(\S+)$`)
var pathParamRe = regexp.MustCompile(`:([A-Za-z_][A-Za-z0-9_]*)`)

func RunAPI(args []string, d APIDeps) int {
	if len(args) == 0 {
		PrintAPIHelp(d.Out)
		return 1
	}
	if args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		PrintAPIHelp(d.Out)
		return 0
	}

	switch args[0] {
	case "spec":
		return runAPISpec(args[1:], d)
	default:
		fmt.Fprintf(d.Err, "unknown api command: %s\n\n", args[0])
		PrintAPIHelp(d.Err)
		return 1
	}
}

func PrintAPIHelp(w io.Writer) {
	fmt.Fprintln(w, "ship api commands:")
	fmt.Fprintln(w, "  ship api:spec [--out <path>] [--serve]  Generate OpenAPI JSON from route contracts, optionally write file or serve docs UI")
}

func runAPISpec(args []string, d APIDeps) int {
	fs := flag.NewFlagSet("api:spec", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	outPath := fs.String("out", "", "write generated OpenAPI JSON to file")
	serve := fs.Bool("serve", false, "serve Swagger UI with generated OpenAPI JSON")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(d.Err, "invalid api:spec arguments: %v\n", err)
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(d.Err, "unexpected api:spec arguments: %v\n", fs.Args())
		return 1
	}
	if d.FindGoModule == nil {
		fmt.Fprintln(d.Err, "api:spec requires FindGoModule dependency")
		return 1
	}

	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve working directory: %v\n", err)
		return 1
	}
	root, _, err := d.FindGoModule(wd)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve project root (go.mod): %v\n", err)
		return 1
	}

	spec, err := buildOpenAPISpec(filepath.Join(root, "app", "contracts"))
	if err != nil {
		fmt.Fprintf(d.Err, "api:spec failed: %v\n", err)
		return 1
	}
	payload, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		fmt.Fprintf(d.Err, "failed to encode OpenAPI JSON: %v\n", err)
		return 1
	}
	payload = append(payload, '\n')

	if *outPath != "" {
		target := *outPath
		if !filepath.IsAbs(target) {
			target = filepath.Join(root, target)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			fmt.Fprintf(d.Err, "failed to create output directory: %v\n", err)
			return 1
		}
		if err := os.WriteFile(target, payload, 0o644); err != nil {
			fmt.Fprintf(d.Err, "failed to write OpenAPI file: %v\n", err)
			return 1
		}
	}

	if *outPath == "" {
		if _, err := d.Out.Write(payload); err != nil {
			fmt.Fprintf(d.Err, "failed to write OpenAPI output: %v\n", err)
			return 1
		}
	}

	if *serve {
		return serveOpenAPISpec(d.Out, d.Err, payload)
	}

	return 0
}

func buildOpenAPISpec(contractsDir string) (openAPISpec, error) {
	spec := openAPISpec{
		OpenAPI: "3.0.0",
		Info: openAPIInfo{
			Title:   "GoShip App",
			Version: "1.0.0",
			License: openAPILicense{
				Name: "UNLICENSED",
				URL:  "https://example.com/license",
			},
		},
		Servers:  []openAPIServer{{URL: "/"}},
		Security: []map[string][]string{
			{"cookieAuth": {}},
		},
		Paths: map[string]openAPIPathItem{},
		Components: openAPIComponents{
			Schemas: map[string]openAPISchema{},
			SecuritySchemes: map[string]openAPISecurityScheme{
				"cookieAuth": {
					Type: "apiKey",
					In:   "cookie",
					Name: "goship_session",
				},
			},
		},
	}

	routes, structs, err := collectContractRoutes(contractsDir)
	if err != nil {
		return openAPISpec{}, err
	}

	for _, route := range routes {
		spec.Components.Schemas[route.Name] = schemaFromStruct(route.Name, structs, map[string]bool{})

		path, params := normalizeOpenAPIPath(route.Path)
		if _, ok := spec.Paths[path]; !ok {
			spec.Paths[path] = openAPIPathItem{}
		}

		op := openAPIOperation{
			OperationID: route.Name,
			Summary:     summarizeOperation(route.Name),
			Responses: map[string]openAPIResponse{
				"200": {Description: "Success"},
				"400": {Description: "Bad request"},
			},
		}

		if len(params) > 0 {
			op.Parameters = make([]openAPIParameter, 0, len(params))
			for _, param := range params {
				op.Parameters = append(op.Parameters, openAPIParameter{
					Name:     param,
					In:       "path",
					Required: true,
					Schema: openAPISchema{
						Type: "string",
					},
				})
			}
		}

		schemaRef := openAPISchema{Ref: "#/components/schemas/" + route.Name}
		switch route.Method {
		case "POST", "PUT", "PATCH":
			op.RequestBody = &openAPIRequestBody{
				Required: true,
				Content: map[string]openAPIMediaType{
					"application/json": {Schema: schemaRef},
				},
			}
		default:
			op.Responses["200"] = openAPIResponse{
				Description: "Success",
				Content: map[string]openAPIMediaType{
					"application/json": {Schema: schemaRef},
				},
			}
		}

		spec.Paths[path][strings.ToLower(route.Method)] = op
	}

	return spec, nil
}

func collectContractRoutes(contractsDir string) ([]contractRoute, map[string]*ast.StructType, error) {
	entries, err := os.ReadDir(contractsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, map[string]*ast.StructType{}, nil
		}
		return nil, nil, err
	}

	fset := token.NewFileSet()
	routes := make([]contractRoute, 0)
	structs := map[string]*ast.StructType{}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
			continue
		}
		path := filepath.Join(contractsDir, entry.Name())
		file, parseErr := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if parseErr != nil {
			return nil, nil, parseErr
		}

		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				st, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}
				structs[typeSpec.Name.Name] = st

				doc := typeSpec.Doc
				if doc == nil {
					doc = genDecl.Doc
				}
				method, routePath, found := parseRouteComment(doc)
				if !found {
					continue
				}
				routes = append(routes, contractRoute{
					Name:   typeSpec.Name.Name,
					Method: method,
					Path:   routePath,
				})
			}
		}
	}

	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Path == routes[j].Path {
			if routes[i].Method == routes[j].Method {
				return routes[i].Name < routes[j].Name
			}
			return routes[i].Method < routes[j].Method
		}
		return routes[i].Path < routes[j].Path
	})
	return routes, structs, nil
}

func parseRouteComment(group *ast.CommentGroup) (method, path string, ok bool) {
	if group == nil {
		return "", "", false
	}
	for _, comment := range group.List {
		text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
		match := routeCommentRe.FindStringSubmatch(text)
		if match == nil {
			continue
		}
		routePath := match[2]
		if !strings.HasPrefix(routePath, "/") {
			routePath = "/" + routePath
		}
		return match[1], routePath, true
	}
	return "", "", false
}

func normalizeOpenAPIPath(path string) (string, []string) {
	params := make([]string, 0)
	out := pathParamRe.ReplaceAllStringFunc(path, func(segment string) string {
		name := strings.TrimPrefix(segment, ":")
		params = append(params, name)
		return "{" + name + "}"
	})
	return out, params
}

func schemaFromStruct(name string, structs map[string]*ast.StructType, seen map[string]bool) openAPISchema {
	st := structs[name]
	if st == nil {
		return openAPISchema{Type: "object"}
	}
	if seen[name] {
		return openAPISchema{Type: "object"}
	}
	seen[name] = true
	defer delete(seen, name)

	props := map[string]openAPISchema{}
	required := make([]string, 0)

	for _, field := range st.Fields.List {
		fieldSchema := schemaFromExpr(field.Type, structs, seen)
		names := fieldJSONNames(field)
		if len(names) == 0 {
			embedded := embeddedFieldName(field.Type)
			if embedded == "" {
				continue
			}
			props[embedded] = fieldSchema
			continue
		}
		for _, name := range names {
			props[name] = fieldSchema
			if fieldRequired(field) {
				required = append(required, name)
			}
		}
	}

	if len(props) == 0 {
		return openAPISchema{Type: "object"}
	}
	sort.Strings(required)
	required = compactSorted(required)

	schema := openAPISchema{
		Type:       "object",
		Properties: props,
	}
	if len(required) > 0 {
		schema.Required = required
	}
	return schema
}

func schemaFromExpr(expr ast.Expr, structs map[string]*ast.StructType, seen map[string]bool) openAPISchema {
	switch t := expr.(type) {
	case *ast.Ident:
		if schema, ok := primitiveSchemaForIdent(t.Name); ok {
			return schema
		}
		if _, ok := structs[t.Name]; ok {
			return openAPISchema{Ref: "#/components/schemas/" + t.Name}
		}
		return openAPISchema{Type: "object"}
	case *ast.SelectorExpr:
		if id, ok := t.X.(*ast.Ident); ok && id.Name == "time" && t.Sel.Name == "Time" {
			return openAPISchema{Type: "string", Format: "date-time"}
		}
		return openAPISchema{Type: "object"}
	case *ast.StarExpr:
		return schemaFromExpr(t.X, structs, seen)
	case *ast.ArrayType:
		item := schemaFromExpr(t.Elt, structs, seen)
		return openAPISchema{Type: "array", Items: &item}
	case *ast.StructType:
		props := map[string]openAPISchema{}
		required := make([]string, 0)
		for _, field := range t.Fields.List {
			fieldSchema := schemaFromExpr(field.Type, structs, seen)
			names := fieldJSONNames(field)
			if len(names) == 0 {
				embedded := embeddedFieldName(field.Type)
				if embedded == "" {
					continue
				}
				props[embedded] = fieldSchema
				continue
			}
			for _, name := range names {
				props[name] = fieldSchema
				if fieldRequired(field) {
					required = append(required, name)
				}
			}
		}
		schema := openAPISchema{Type: "object"}
		if len(props) > 0 {
			schema.Properties = props
		}
		sort.Strings(required)
		required = compactSorted(required)
		if len(required) > 0 {
			schema.Required = required
		}
		return schema
	case *ast.MapType:
		return openAPISchema{Type: "object"}
	default:
		return openAPISchema{Type: "object"}
	}
}

func primitiveSchemaForIdent(name string) (openAPISchema, bool) {
	switch name {
	case "string":
		return openAPISchema{Type: "string"}, true
	case "bool":
		return openAPISchema{Type: "boolean"}, true
	case "float32", "float64":
		return openAPISchema{Type: "number"}, true
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		return openAPISchema{Type: "integer"}, true
	default:
		return openAPISchema{}, false
	}
}

func fieldJSONNames(field *ast.Field) []string {
	names := make([]string, 0)
	for _, name := range field.Names {
		names = append(names, lowerCamel(name.Name))
	}
	if field.Tag == nil {
		return names
	}
	tagValue := strings.Trim(field.Tag.Value, "`")
	tag := reflect.StructTag(tagValue)
	for _, key := range []string{"json", "form", "query", "param"} {
		value := tag.Get(key)
		if value == "" {
			continue
		}
		parts := strings.Split(value, ",")
		if len(parts) > 0 && parts[0] != "" && parts[0] != "-" {
			if len(field.Names) == 1 {
				return []string{parts[0]}
			}
		}
	}
	return names
}

func fieldRequired(field *ast.Field) bool {
	if field.Tag == nil {
		return false
	}
	tagValue := strings.Trim(field.Tag.Value, "`")
	validate := reflect.StructTag(tagValue).Get("validate")
	for _, part := range strings.Split(validate, ",") {
		if strings.TrimSpace(part) == "required" {
			return true
		}
	}
	return false
}

func embeddedFieldName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return lowerCamel(t.Name)
	case *ast.SelectorExpr:
		return lowerCamel(t.Sel.Name)
	case *ast.StarExpr:
		return embeddedFieldName(t.X)
	default:
		return ""
	}
}

func lowerCamel(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func compactSorted(values []string) []string {
	if len(values) == 0 {
		return values
	}
	out := make([]string, 0, len(values))
	last := ""
	for _, v := range values {
		if v == last {
			continue
		}
		out = append(out, v)
		last = v
	}
	return out
}

func summarizeOperation(name string) string {
	parts := strings.Fields(strings.TrimSpace(strings.ReplaceAll(splitCamel(name), "  ", " ")))
	if len(parts) == 0 {
		return "Operation"
	}
	return strings.Join(parts, " ")
}

func splitCamel(name string) string {
	var b strings.Builder
	for i, r := range name {
		if i > 0 && (r >= 'A' && r <= 'Z') {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func serveOpenAPISpec(out io.Writer, errOut io.Writer, payload []byte) int {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintf(errOut, "failed to start api:spec server: %v\n", err)
		return 1
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/openapi.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(payload)
	})
	mux.HandleFunc("/api/docs", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, `<!doctype html>
<html>
<head>
  <meta charset="utf-8" />
  <title>GoShip API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: '/openapi.json',
      dom_id: '#swagger-ui'
    });
  </script>
</body>
</html>`)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/api/docs", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	})

	server := &http.Server{Handler: mux}
	go func() {
		_ = server.Serve(listener)
	}()

	fmt.Fprintf(out, "serving OpenAPI docs at http://%s/api/docs (Ctrl+C to stop)\n", listener.Addr().String())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	signal.Stop(sigCh)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
	return 0
}
