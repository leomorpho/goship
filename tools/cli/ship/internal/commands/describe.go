package commands

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	rt "github.com/leomorpho/goship/tools/cli/ship/internal/runtime"
)

type DescribeDeps struct {
	Out          io.Writer
	Err          io.Writer
	FindGoModule func(start string) (string, string, error)
}

type describeRoute struct {
	Method  string `json:"method"`
	Path    string `json:"path"`
	Handler string `json:"handler"`
	Auth    bool   `json:"auth"`
	File    string `json:"file"`
}

type describeModule struct {
	ID         string `json:"id"`
	Installed  bool   `json:"installed"`
	Routes     int    `json:"routes"`
	Migrations int    `json:"migrations"`
}

type describeController struct {
	Name     string   `json:"name"`
	File     string   `json:"file"`
	Handlers []string `json:"handlers"`
}

type describeViewModel struct {
	Name   string   `json:"name"`
	File   string   `json:"file"`
	Fields []string `json:"fields"`
}

type describeComponent struct {
	Name          string `json:"name"`
	File          string `json:"file"`
	DataComponent string `json:"data_component"`
}

type describeIsland struct {
	Name string `json:"name"`
	File string `json:"file"`
}

type describeMigration struct {
	File    string `json:"file"`
	Applied *bool  `json:"applied"`
}

type describeSharedInfra struct {
	SharedModules        int      `json:"shared_modules"`
	SharedModuleIDs      []string `json:"shared_module_ids"`
	CustomAppControllers int      `json:"custom_app_controllers"`
	CustomAppJobs        int      `json:"custom_app_jobs"`
	CustomAppCommands    int      `json:"custom_app_commands"`
}

type describeModuleAdoption struct {
	ID         string `json:"id"`
	ModulePath string `json:"module_path"`
	Version    string `json:"version"`
	Source     string `json:"source"`
	Installed  bool   `json:"installed"`
}

type describeResult struct {
	Routes         []describeRoute          `json:"routes"`
	Modules        []describeModule         `json:"modules"`
	ModuleAdoption []describeModuleAdoption `json:"module_adoption"`
	Controllers    []describeController     `json:"controllers"`
	ViewModels     []describeViewModel      `json:"viewmodels"`
	Components     []describeComponent      `json:"components"`
	Islands        []describeIsland         `json:"islands"`
	DBTables       []string                 `json:"db_tables"`
	Migrations     []describeMigration      `json:"migrations"`
	SharedInfra    describeSharedInfra      `json:"shared_infra"`
}

func RunDescribe(args []string, d DescribeDeps) int {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			PrintDescribeHelp(d.Out)
			return 0
		}
	}

	fs := flag.NewFlagSet("describe", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pretty := fs.Bool("pretty", false, "pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(d.Err, "invalid describe arguments: %v\n", err)
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(d.Err, "unexpected describe arguments: %v\n", fs.Args())
		return 1
	}
	if d.FindGoModule == nil {
		fmt.Fprintln(d.Err, "describe requires FindGoModule dependency")
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

	var payload describeResult
	if err := withWorkingDir(root, func() error {
		var buildErr error
		payload, buildErr = buildDescribeResult(root)
		return buildErr
	}); err != nil {
		fmt.Fprintf(d.Err, "describe failed: %v\n", err)
		return 1
	}

	enc := json.NewEncoder(d.Out)
	if *pretty {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(payload); err != nil {
		fmt.Fprintf(d.Err, "failed to encode describe output: %v\n", err)
		return 1
	}
	return 0
}

func PrintDescribeHelp(w io.Writer) {
	fmt.Fprintln(w, "ship describe commands:")
	fmt.Fprintln(w, "  ship describe           Print project inventory as JSON")
	fmt.Fprintln(w, "  ship describe --pretty  Print project inventory as pretty JSON")
}

func buildDescribeResult(root string) (describeResult, error) {
	routes, err := collectDescribeRoutes(root)
	if err != nil {
		return describeResult{}, err
	}
	modules, err := collectDescribeModules(root)
	if err != nil {
		return describeResult{}, err
	}
	controllers, err := collectDescribeControllers(root)
	if err != nil {
		return describeResult{}, err
	}
	viewmodels, err := collectDescribeViewModels(root)
	if err != nil {
		return describeResult{}, err
	}
	components, err := collectDescribeComponents(root)
	if err != nil {
		return describeResult{}, err
	}
	islands, err := collectDescribeIslands(root)
	if err != nil {
		return describeResult{}, err
	}
	dbTables, err := collectDescribeDBTables(root)
	if err != nil {
		return describeResult{}, err
	}
	migrations, err := collectDescribeMigrations(root)
	if err != nil {
		return describeResult{}, err
	}
	moduleAdoption, err := collectDescribeModuleAdoption(root, modules)
	if err != nil {
		return describeResult{}, err
	}
	sharedInfra, err := collectDescribeSharedInfra(root, modules, controllers)
	if err != nil {
		return describeResult{}, err
	}

	return describeResult{
		Routes:         routes,
		Modules:        modules,
		ModuleAdoption: moduleAdoption,
		Controllers:    controllers,
		ViewModels:     viewmodels,
		Components:     components,
		Islands:        islands,
		DBTables:       dbTables,
		Migrations:     migrations,
		SharedInfra:    sharedInfra,
	}, nil
}

func collectDescribeModuleAdoption(root string, modules []describeModule) ([]describeModuleAdoption, error) {
	adoptionByID := make(map[string]describeModuleAdoption, len(modules))
	for _, info := range moduleCatalog {
		id := strings.TrimSpace(info.ID)
		modulePath := strings.TrimSpace(info.ModulePath)
		if id == "" || modulePath == "" {
			continue
		}
		adoptionByID[id] = describeModuleAdoption{
			ID:         id,
			ModulePath: modulePath,
			Version:    "v0.0.0",
			Source:     "first-party-catalog",
			Installed:  false,
		}
	}

	for _, module := range modules {
		if !module.Installed {
			continue
		}

		modulePath := fmt.Sprintf("github.com/leomorpho/goship-modules/%s", module.ID)
		source := "tagged-release"
		version := "v0.0.0"

		moduleGoMod := filepath.Join(root, "modules", module.ID, "go.mod")
		if _, err := os.Stat(moduleGoMod); err == nil {
			if parsedPath := parseDescribeModulePath(moduleGoMod); parsedPath != "" {
				modulePath = parsedPath
			}
			source = "local-replace"
		} else if err != nil && !os.IsNotExist(err) {
			return nil, err
		}

		adoptionByID[module.ID] = describeModuleAdoption{
			ID:         module.ID,
			ModulePath: modulePath,
			Version:    version,
			Source:     source,
			Installed:  true,
		}
	}

	adoption := make([]describeModuleAdoption, 0, len(adoptionByID))
	for _, entry := range adoptionByID {
		adoption = append(adoption, entry)
	}
	sort.Slice(adoption, func(i, j int) bool {
		return adoption[i].ID < adoption[j].ID
	})

	return adoption, nil
}

func parseDescribeModulePath(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

func collectDescribeSharedInfra(root string, modules []describeModule, controllers []describeController) (describeSharedInfra, error) {
	report := describeSharedInfra{
		SharedModuleIDs: make([]string, 0, len(modules)),
	}
	for _, module := range modules {
		if module.Installed {
			report.SharedModules++
			report.SharedModuleIDs = append(report.SharedModuleIDs, module.ID)
		}
	}
	sort.Strings(report.SharedModuleIDs)
	report.CustomAppControllers = len(controllers)

	jobFiles, err := filepath.Glob(filepath.Join(root, "app", "jobs", "*.go"))
	if err != nil {
		return describeSharedInfra{}, err
	}
	for _, path := range jobFiles {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		report.CustomAppJobs++
	}

	commandFiles, err := filepath.Glob(filepath.Join(root, "app", "commands", "*.go"))
	if err != nil {
		return describeSharedInfra{}, err
	}
	for _, path := range commandFiles {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		report.CustomAppCommands++
	}

	return report, nil
}

func collectDescribeRoutes(root string) ([]describeRoute, error) {
	path := filepath.Join(root, "app", "router.go")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}

	routes := make([]describeRoute, 0)
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel == nil {
				return true
			}
			method := sel.Sel.Name
			if method != "GET" && method != "POST" && method != "PUT" && method != "DELETE" && method != "PATCH" {
				return true
			}
			pathExpr := describeExprString(fset, call.Args, 0)
			handler := describeExprString(fset, call.Args, 1)
			receiver := describeExpr(fset, sel.X)
			line := fset.Position(call.Pos()).Line
			routes = append(routes, describeRoute{
				Method:  method,
				Path:    pathExpr,
				Handler: handler,
				Auth:    describeRouteAuth(fn.Name.Name, receiver),
				File:    fmt.Sprintf("%s:%d", filepath.ToSlash(mustRelPath(root, path)), line),
			})
			return true
		})
	}

	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Path == routes[j].Path {
			if routes[i].Method == routes[j].Method {
				return routes[i].Handler < routes[j].Handler
			}
			return routes[i].Method < routes[j].Method
		}
		return routes[i].Path < routes[j].Path
	})
	return routes, nil
}

func collectDescribeModules(root string) ([]describeModule, error) {
	modulesDir := filepath.Join(root, "modules")
	entries, err := os.ReadDir(modulesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	enabledModules := map[string]bool{}
	path := filepath.Join(root, "config", "modules.yaml")
	if manifest, err := rt.LoadModulesManifest(path); err == nil {
		for _, name := range manifest.Modules {
			enabledModules[name] = true
		}
	}

	var modules []describeModule
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		name := entry.Name()
		migrations := 0
		migrationsDir := filepath.Join(modulesDir, name, "db", "migrate", "migrations")
		if entries, err := os.ReadDir(migrationsDir); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
					migrations++
				}
			}
		}

		routes := 0
		routesFiles := []string{
			filepath.Join(modulesDir, name, "routes", "routes.go"),
			filepath.Join(modulesDir, name, "routes.go"),
		}
		for _, rf := range routesFiles {
			if _, err := os.Stat(rf); err == nil {
				routes += countModuleRoutes(rf)
			}
		}

		modules = append(modules, describeModule{
			ID:         name,
			Installed:  enabledModules[name],
			Routes:     routes,
			Migrations: migrations,
		})
	}
	sort.Slice(modules, func(i, j int) bool { return modules[i].ID < modules[j].ID })
	return modules, nil
}

func countModuleRoutes(path string) int {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return 0
	}
	count := 0
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel == nil {
				return true
			}
			method := sel.Sel.Name
			if method == "GET" || method == "POST" || method == "PUT" || method == "DELETE" || method == "PATCH" {
				count++
			}
			return true
		})
	}
	return count
}

func collectDescribeControllers(root string) ([]describeController, error) {
	dir := filepath.Join(root, "app", "web", "controllers")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	type info struct {
		file     string
		handlers []string
	}
	byName := map[string]*info{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		fset := token.NewFileSet()
		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return nil, parseErr
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Name == nil || !describeHandlerLike(fn) {
				continue
			}
			name := describeReceiverName(fn)
			if name == "" {
				continue
			}
			if _, ok := byName[name]; !ok {
				byName[name] = &info{file: filepath.ToSlash(mustRelPath(root, path))}
			}
			byName[name].handlers = append(byName[name].handlers, fn.Name.Name)
		}
	}

	controllers := make([]describeController, 0, len(byName))
	for name, info := range byName {
		sort.Strings(info.handlers)
		controllers = append(controllers, describeController{Name: name, File: info.file, Handlers: info.handlers})
	}
	sort.Slice(controllers, func(i, j int) bool { return controllers[i].Name < controllers[j].Name })
	return controllers, nil
}

func collectDescribeViewModels(root string) ([]describeViewModel, error) {
	dir := filepath.Join(root, "app", "web", "viewmodels")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []describeViewModel
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		fset := token.NewFileSet()
		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return nil, parseErr
		}
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}
				fields := make([]string, 0, len(structType.Fields.List))
				for _, field := range structType.Fields.List {
					for _, name := range field.Names {
						fields = append(fields, name.Name)
					}
				}
				out = append(out, describeViewModel{
					Name:   typeSpec.Name.Name,
					File:   filepath.ToSlash(mustRelPath(root, path)),
					Fields: fields,
				})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func collectDescribeComponents(root string) ([]describeComponent, error) {
	dir := filepath.Join(root, "app", "views", "web", "components")
	var out []describeComponent
	re := regexp.MustCompile(`data-component="([^"]+)"`)
	if !isDirPath(dir) {
		return nil, nil
	}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".templ") {
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		match := re.FindStringSubmatch(string(b))
		component := describeComponent{
			Name:          describeDisplayName(strings.TrimSuffix(filepath.Base(path), ".templ")),
			File:          filepath.ToSlash(mustRelPath(root, path)),
			DataComponent: "",
		}
		if len(match) > 1 {
			component.DataComponent = match[1]
		}
		out = append(out, component)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].File < out[j].File })
	return out, nil
}

func collectDescribeIslands(root string) ([]describeIsland, error) {
	dir := filepath.Join(root, "frontend", "islands")
	if !isDirPath(dir) {
		return nil, nil
	}
	var out []describeIsland
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Base(path) == ".gitkeep" {
			return nil
		}
		out = append(out, describeIsland{
			Name: describeDisplayName(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))),
			File: filepath.ToSlash(mustRelPath(root, path)),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].File < out[j].File })
	return out, nil
}

func collectDescribeDBTables(root string) ([]string, error) {
	re := regexp.MustCompile(`(?i)\b(?:CREATE TABLE(?: IF NOT EXISTS)?|FROM|JOIN|UPDATE|INSERT INTO|DELETE FROM)\s+([a-zA-Z_][a-zA-Z0-9_]*)`)
	tables := map[string]struct{}{}
	err := filepath.WalkDir(filepath.Join(root, "db", "queries"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".sql") {
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, match := range re.FindAllStringSubmatch(string(b), -1) {
			if len(match) > 1 {
				tables[strings.ToLower(match[1])] = struct{}{}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(tables))
	for table := range tables {
		out = append(out, table)
	}
	sort.Strings(out)
	return out, nil
}

func collectDescribeMigrations(root string) ([]describeMigration, error) {
	paths := []string{
		filepath.Join(root, "db", "migrate", "migrations"),
	}
	modulesDir := filepath.Join(root, "modules")
	if entries, err := os.ReadDir(modulesDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				paths = append(paths, filepath.Join(modulesDir, entry.Name(), "db", "migrate", "migrations"))
			}
		}
	}

	var out []describeMigration
	for _, dir := range paths {
		if !isDirPath(dir) {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
				continue
			}
			out = append(out, describeMigration{
				File:    filepath.ToSlash(mustRelPath(root, filepath.Join(dir, entry.Name()))),
				Applied: nil,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].File < out[j].File })
	return out, nil
}

func describeExprString(fset *token.FileSet, args []ast.Expr, idx int) string {
	if idx >= len(args) {
		return ""
	}
	return describeExpr(fset, args[idx])
}

func describeExpr(fset *token.FileSet, expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	var b bytes.Buffer
	_ = printer.Fprint(&b, fset, expr)
	return strings.TrimSpace(b.String())
}

func describeRouteAuth(fnName string, receiver string) bool {
	if fnName == "registerRealtimeRoutes" {
		return true
	}
	switch receiver {
	case "allGroup", "onboardingGroup", "onboardedGroup":
		return true
	default:
		return false
	}
}

func describeHandlerLike(fn *ast.FuncDecl) bool {
	return fn.Recv != nil && fn.Type != nil && fn.Type.Params != nil && fn.Type.Results != nil &&
		len(fn.Type.Results.List) == 1 && describeIsErrorType(fn.Type.Results.List[0].Type) &&
		describeHasEchoContext(fn.Type.Params.List)
}

func describeHasEchoContext(fields []*ast.Field) bool {
	for _, field := range fields {
		sel, ok := field.Type.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil || sel.Sel.Name != "Context" {
			continue
		}
		if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "echo" {
			return true
		}
	}
	return false
}

func describeIsErrorType(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "error"
}

func describeReceiverName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}
	switch t := fn.Recv.List[0].Type.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}

func describeDisplayName(raw string) string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, "")
}

func mustRelPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}

func isDirPath(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
