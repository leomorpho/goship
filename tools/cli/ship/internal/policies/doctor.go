package policies

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	rt "github.com/leomorpho/goship/tools/cli/ship/internal/runtime"
)

type DoctorIssue struct {
	Code     string
	Message  string
	Fix      string
	File     string
	Severity string
}

type DoctorDeps struct {
	Out          io.Writer
	Err          io.Writer
	FindGoModule func(start string) (string, string, error)
}

func RunDoctor(args []string, d DoctorDeps) int {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			printDoctorHelp(d.Out)
			return 0
		}
	}

	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOutput := fs.Bool("json", false, "output doctor issues as JSON")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(d.Err, "invalid doctor arguments: %v\n", err)
		return 1
	}
	if fs.NArg() > 0 {
		if *jsonOutput {
			return writeDoctorJSON(d.Out, false, []DoctorIssue{{
				Code:     "config",
				Message:  fmt.Sprintf("unexpected doctor arguments: %v", fs.Args()),
				Severity: "error",
			}})
		}
		fmt.Fprintf(d.Err, "unexpected doctor arguments: %v\n", fs.Args())
		return 1
	}

	wd, err := os.Getwd()
	if err != nil {
		if *jsonOutput {
			return writeDoctorJSON(d.Out, false, []DoctorIssue{{
				Code:     "config",
				Message:  fmt.Sprintf("failed to resolve working directory: %v", err),
				Severity: "error",
			}})
		}
		fmt.Fprintf(d.Err, "failed to resolve working directory: %v\n", err)
		return 1
	}
	root, _, err := d.FindGoModule(wd)
	if err != nil {
		if *jsonOutput {
			return writeDoctorJSON(d.Out, false, []DoctorIssue{{
				Code:     "config",
				Message:  fmt.Sprintf("failed to resolve project root (go.mod): %v", err),
				Severity: "error",
			}})
		}
		fmt.Fprintf(d.Err, "failed to resolve project root (go.mod): %v\n", err)
		return 1
	}

	issues := RunDoctorChecks(root)
	if *jsonOutput {
		return writeDoctorJSON(d.Out, !hasDoctorErrors(issues), issues)
	}

	if !hasDoctorErrors(issues) && len(issues) == 0 {
		fmt.Fprintf(d.Out, "ship doctor: OK (%s)\n", root)
		return 0
	}
	if !hasDoctorErrors(issues) {
		fmt.Fprintf(d.Out, "ship doctor: OK with %d warning(s) (%s)\n", len(issues), root)
		printDoctorIssues(d.Out, issues)
		return 0
	}

	fmt.Fprintf(d.Err, "ship doctor: found %d issue(s)\n", len(issues))
	printDoctorIssues(d.Err, issues)
	return 1
}

type doctorJSONIssue struct {
	Type     string `json:"type"`
	File     string `json:"file"`
	Detail   string `json:"detail"`
	Severity string `json:"severity"`
}

type doctorJSONResult struct {
	OK     bool              `json:"ok"`
	Issues []doctorJSONIssue `json:"issues"`
}

func writeDoctorJSON(w io.Writer, ok bool, issues []DoctorIssue) int {
	payload := doctorJSONResult{
		OK:     ok,
		Issues: make([]doctorJSONIssue, 0, len(issues)),
	}
	for _, issue := range issues {
		payload.Issues = append(payload.Issues, doctorJSONIssue{
			Type:     issue.Code,
			File:     filepath.ToSlash(issue.File),
			Detail:   issue.Message,
			Severity: doctorIssueSeverity(issue),
		})
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(payload); err != nil {
		fmt.Fprintf(w, "{\"ok\":false,\"issues\":[{\"type\":\"config\",\"file\":\"\",\"detail\":%q,\"severity\":\"error\"}]}\n", fmt.Sprintf("failed to encode doctor JSON: %v", err))
		return 1
	}
	if ok {
		return 0
	}
	return 1
}

func hasDoctorErrors(issues []DoctorIssue) bool {
	for _, issue := range issues {
		if doctorIssueSeverity(issue) != "warning" {
			return true
		}
	}
	return false
}

func doctorIssueSeverity(issue DoctorIssue) string {
	if issue.Severity == "" {
		return "error"
	}
	return issue.Severity
}

func printDoctorIssues(w io.Writer, issues []DoctorIssue) {
	for _, issue := range issues {
		fmt.Fprintf(w, "- [%s] %s\n", issue.Code, issue.Message)
		if issue.Fix != "" {
			fmt.Fprintf(w, "  fix: %s\n", issue.Fix)
		}
	}
}

func RunDoctorChecks(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)

	requiredDirs := []string{
		filepath.Join("app"),
		filepath.Join("app", "foundation"),
		filepath.Join("app", "web", "controllers"),
		filepath.Join("app", "web", "middleware"),
		filepath.Join("app", "web", "ui"),
		filepath.Join("app", "web", "viewmodels"),
		filepath.Join("app", "jobs"),
		filepath.Join("app", "views"),
		filepath.Join("db", "queries"),
		filepath.Join("db", "migrate", "migrations"),
	}
	for _, rel := range requiredDirs {
		if !isDir(filepath.Join(root, rel)) {
			issues = append(issues, DoctorIssue{
				Code:    "DX001",
				Message: fmt.Sprintf("missing required directory: %s", rel),
				Fix:     fmt.Sprintf("create %s or regenerate the app scaffold with `ship new`", rel),
			})
		}
	}

	requiredFiles := []string{
		filepath.Join("app", "router.go"),
		filepath.Join("app", "foundation", "container.go"),
		filepath.Join("app", "web", "routenames", "routenames.go"),
		filepath.Join("db", "bobgen.yaml"),
		filepath.Join("config", "modules.yaml"),
		filepath.Join("docs", "00-index.md"),
		filepath.Join("docs", "architecture", "01-architecture.md"),
		filepath.Join("docs", "architecture", "08-cognitive-model.md"),
	}
	for _, rel := range requiredFiles {
		if !hasFile(filepath.Join(root, rel)) {
			issues = append(issues, DoctorIssue{
				Code:    "DX002",
				Message: fmt.Sprintf("missing required file: %s", rel),
				Fix:     "restore missing documentation or scaffold files",
			})
		}
	}

	forbidden := []string{
		filepath.Join("app", "site"),
		filepath.Join("app", "bootstrap"),
		filepath.Join("app", "domains"),
		filepath.Join("app", "tasks"),
		filepath.Join("app", "types"),
		filepath.Join("app", "webui"),
		filepath.Join("app", "middleware"),
	}
	for _, rel := range forbidden {
		if pathExists(filepath.Join(root, rel)) {
			issues = append(issues, DoctorIssue{
				Code:    "DX003",
				Message: fmt.Sprintf("forbidden legacy path present: %s", rel),
				Fix:     "remove or migrate legacy paths to canonical app layout",
			})
		}
	}

	rootBinaries := []string{"web", "worker", "seed", "ship", "ship-mcp"}
	for _, name := range rootBinaries {
		if hasFile(filepath.Join(root, name)) {
			issues = append(issues, DoctorIssue{
				Code:    "DX008",
				Message: fmt.Sprintf("root build artifact present: %s", name),
				Fix:     fmt.Sprintf("remove %s and keep it ignored in .gitignore", name),
			})
		}
	}

	gitignore := filepath.Join(root, ".gitignore")
	if hasFile(gitignore) {
		content, err := os.ReadFile(gitignore)
		if err != nil {
			issues = append(issues, DoctorIssue{
				Code:    "DX009",
				Message: "failed to read .gitignore",
				Fix:     err.Error(),
			})
		} else {
			ignoreText := string(content)
			required := []string{"/web", "/worker", "/seed", "/ship", "/ship-mcp"}
			for _, entry := range required {
				if !strings.Contains(ignoreText, entry) {
					issues = append(issues, DoctorIssue{
						Code:    "DX009",
						Message: fmt.Sprintf(".gitignore missing required artifact entry: %s", entry),
						Fix:     "add required root binary ignore entries to .gitignore",
					})
				}
			}
		}
	}

	issues = append(issues, checkMarkerIntegrity(root)...)
	issues = append(issues, checkPackageNaming(root, filepath.Join("app", "web", "ui"), "ui")...)
	issues = append(issues, checkPackageNaming(root, filepath.Join("app", "web", "viewmodels"), "viewmodels")...)
	issues = append(issues, checkTopLevelDirs(root)...)
	issues = append(issues, checkFileSizes(root)...)
	issues = append(issues, checkCLIDocsCoverage(root)...)
	issues = append(issues, checkGoWorkModules(root)...)
	issues = append(issues, checkDockerIgnoreCoverage(root)...)
	issues = append(issues, checkDockerLocalReplaceOrder(root)...)
	issues = append(issues, checkAgentPolicyArtifacts(root)...)
	issues = append(issues, checkModulesManifestFormat(root)...)
	issues = append(issues, checkEnabledModuleDBArtifacts(root)...)
	issues = append(issues, checkForbiddenCrossBoundaryImports(root)...)
	issues = append(issues, checkCanonicalFilePlacement(root)...)

	return issues
}

func checkCanonicalFilePlacement(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	issues = append(issues, checkHandlerPlacement(root)...)
	issues = append(issues, checkRoutePlacement(root)...)
	issues = append(issues, checkRawSQLPlacement(root)...)
	issues = append(issues, checkMigrationPlacement(root)...)
	issues = append(issues, checkConfigStructPlacement(root)...)
	issues = append(issues, checkRendersComments(root)...)
	issues = append(issues, checkDataComponentAttributes(root)...)
	return issues
}

func checkHandlerPlacement(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	handlerName := regexp.MustCompile(`^(Get|Post|Put|Delete|Patch|Handle|Create|Update|Submit|Save|Register|Mark|Index|Show|Destroy|List|Edit)`) 
	controllersDir := filepath.ToSlash(filepath.Join("app", "web", "controllers")) + "/"
	webDir := filepath.Join(root, "app", "web")
	if !isDir(webDir) {
		return issues
	}

	_ = filepath.WalkDir(webDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel := filepath.ToSlash(mustRel(root, path))
		if strings.HasPrefix(rel, controllersDir) {
			return nil
		}
		file, parseErr := parseDoctorGoFile(path)
		if parseErr != nil {
			issues = append(issues, DoctorIssue{
				Code:    "DX021",
				File:    rel,
				Message: fmt.Sprintf("failed to parse Go file for handler placement: %s", rel),
				Fix:     parseErr.Error(),
			})
			return nil
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name == nil || !handlerName.MatchString(fn.Name.Name) {
				continue
			}
			if !funcHasReceiver(fn) {
				continue
			}
			if !funcHasEchoContextParam(fn) || !funcReturnsOnlyError(fn) {
				continue
			}
			issues = append(issues, DoctorIssue{
				Code:    "DX021",
				File:    rel,
				Message: fmt.Sprintf("controller-style HTTP handler must live under app/web/controllers: %s", rel),
				Fix:     "move the handler into app/web/controllers or convert it into a non-handler helper",
			})
			return nil
		}
		return nil
	})

	return issues
}

func checkRoutePlacement(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	allowed := map[string]struct{}{
		"app/router.go":     {},
		"app/web/wiring.go": {},
	}
	methods := map[string]struct{}{
		"GET":    {},
		"POST":   {},
		"PUT":    {},
		"DELETE": {},
		"PATCH":  {},
	}

	appDir := filepath.Join(root, "app")
	if !isDir(appDir) {
		return issues
	}

	_ = filepath.WalkDir(appDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel := filepath.ToSlash(mustRel(root, path))
		if _, ok := allowed[rel]; ok {
			return nil
		}
		file, parseErr := parseDoctorGoFile(path)
		if parseErr != nil {
			issues = append(issues, DoctorIssue{
				Code:    "DX021",
				File:    rel,
				Message: fmt.Sprintf("failed to parse Go file for route placement: %s", rel),
				Fix:     parseErr.Error(),
			})
			return nil
		}
		if fileHasRouteRegistration(file, methods) {
			issues = append(issues, DoctorIssue{
				Code:    "DX021",
				File:    rel,
				Message: fmt.Sprintf("route registration must live in app/router.go or app/web/wiring.go: %s", rel),
				Fix:     "move route registration into app/router.go or app/web/wiring.go",
			})
		}
		return nil
	})

	return issues
}

func checkRawSQLPlacement(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	callArgIndex := map[string]int{
		"Exec":            0,
		"Query":           0,
		"QueryRow":        0,
		"ExecContext":     1,
		"QueryContext":    1,
		"QueryRowContext": 1,
	}
	roots := []string{
		filepath.Join(root, "app"),
		filepath.Join(root, "framework"),
		filepath.Join(root, "modules"),
	}

	for _, scanRoot := range roots {
		if !isDir(scanRoot) {
			continue
		}
		_ = filepath.WalkDir(scanRoot, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel := filepath.ToSlash(mustRel(root, path))
			if doctorAllowsInlineSQL(rel) {
				return nil
			}
			file, parseErr := parseDoctorGoFile(path)
			if parseErr != nil {
				issues = append(issues, DoctorIssue{
					Code:    "DX021",
					File:    rel,
					Message: fmt.Sprintf("failed to parse Go file for SQL placement: %s", rel),
					Fix:     parseErr.Error(),
				})
				return nil
			}
			if fileHasInlineSQLCall(file, callArgIndex) {
				issues = append(issues, DoctorIssue{
					Code:    "DX021",
					File:    rel,
					Message: fmt.Sprintf("inline SQL must live in db/queries assets or dedicated store/query layers: %s", rel),
					Fix:     "move SQL into db/queries and execute it from a store/query abstraction",
				})
			}
			return nil
		})
	}

	return issues
}

func checkMigrationPlacement(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	migrationName := regexp.MustCompile(`^\d{10,}.*\.sql$`)
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			rel := filepath.ToSlash(mustRel(root, path))
			if rel == ".git" || rel == "node_modules" || rel == ".cache" || strings.Contains(rel, "/.cache/") {
				return filepath.SkipDir
			}
			return nil
		}
		base := filepath.Base(path)
		if base != "atlas.sum" && !migrationName.MatchString(base) {
			return nil
		}
		rel := filepath.ToSlash(mustRel(root, path))
		if doctorIsAllowedMigrationPath(rel) {
			return nil
		}
		issues = append(issues, DoctorIssue{
			Code:    "DX021",
			File:    rel,
			Message: fmt.Sprintf("migration files must live under db/migrate/migrations: %s", rel),
			Fix:     "move the migration into db/migrate/migrations or modules/*/db/migrate/migrations",
		})
		return nil
	})
	return issues
}

func checkConfigStructPlacement(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	configDir := filepath.Join(root, "config")
	if !isDir(configDir) {
		return issues
	}
	_ = filepath.WalkDir(configDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel := filepath.ToSlash(mustRel(root, path))
		if rel == "config/config.go" {
			return nil
		}
		file, parseErr := parseDoctorGoFile(path)
		if parseErr != nil {
			issues = append(issues, DoctorIssue{
				Code:    "DX021",
				File:    rel,
				Message: fmt.Sprintf("failed to parse Go file for config placement: %s", rel),
				Fix:     parseErr.Error(),
			})
			return nil
		}
		if fileHasConfigStruct(file) {
			issues = append(issues, DoctorIssue{
				Code:    "DX021",
				File:    rel,
				Message: fmt.Sprintf("config structs must live in config/config.go: %s", rel),
				Fix:     "move app config struct definitions into config/config.go",
			})
		}
		return nil
	})
	return issues
}

func checkRendersComments(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	searchDirs := []string{filepath.Join(root, "app", "views")}
	modulesDir := filepath.Join(root, "modules")
	_ = filepath.WalkDir(modulesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if path == modulesDir {
			return nil
		}
		viewsDir := filepath.Join(path, "views")
		if isDir(viewsDir) {
			searchDirs = append(searchDirs, viewsDir)
		}
		return filepath.SkipDir
	})

	for _, dir := range searchDirs {
		if !isDir(dir) {
			continue
		}
		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || filepath.Ext(path) != ".templ" {
				return nil
			}
			for _, fn := range templFunctionsMissingRenders(path) {
				issues = append(issues, DoctorIssue{
					Code:    "DX023",
					File:    filepath.ToSlash(mustRel(root, path)),
					Message: fmt.Sprintf("exported templ function %s lacks a // Renders: comment", fn),
					Fix:     "add an English `// Renders:` comment describing the UI output just above the templ declaration",
				})
			}
			return nil
		})
	}
	return issues
}

func checkDataComponentAttributes(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	searchDirs := []string{
		filepath.Join(root, "app", "views", "web", "components"),
		filepath.Join(root, "app", "views", "web", "helpers"),
	}
	modulesDir := filepath.Join(root, "modules")
	_ = filepath.WalkDir(modulesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() || path == modulesDir {
			return nil
		}
		compDir := filepath.Join(path, "views", "web", "components")
		if isDir(compDir) {
			searchDirs = append(searchDirs, compDir)
		}
		return filepath.SkipDir
	})

	for _, dir := range searchDirs {
		if !isDir(dir) {
			continue
		}
		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || filepath.Ext(path) != ".templ" {
				return nil
			}
			lines, readErr := readLines(path)
			if readErr != nil {
				return nil
			}
			re := regexp.MustCompile(`^\s*templ\s+([A-Z][A-Za-z0-9_]*)\s*\(`)
			for i, line := range lines {
				match := re.FindStringSubmatch(line)
				if match == nil {
					continue
				}
				fn := match[1]
				rootLine, ok := findRootElementLine(lines, i+1)
				if !ok {
					issues = append(issues, DoctorIssue{
						Code:    "DX024",
						File:    filepath.ToSlash(mustRel(root, path)),
						Message: fmt.Sprintf("templ %s has no recognizable root element for data-component", fn),
						Fix:     "add a root HTML element with a data-component attribute matching the component name",
					})
					continue
				}
				hasAttr, value := extractDataComponent(rootLine)
				expected := toKebabCase(fn)
				if !hasAttr {
					issues = append(issues, DoctorIssue{
						Code:    "DX024",
						File:    filepath.ToSlash(mustRel(root, path)),
						Message: fmt.Sprintf("templ %s lacks data-component attribute on its root element", fn),
						Fix:     fmt.Sprintf("add `data-component=\"%s\"` to the root element", expected),
					})
					continue
				}
				if value != expected {
					issues = append(issues, DoctorIssue{
						Code:    "DX024",
						File:    filepath.ToSlash(mustRel(root, path)),
						Message: fmt.Sprintf("templ %s has data-component=%q but should use %q", fn, value, expected),
						Fix:     fmt.Sprintf("set `data-component=\"%s\"` on the root element", expected),
					})
				}
			}
			return nil
		})
	}

	return issues
}

func findRootElementLine(lines []string, start int) (string, bool) {
	for i := start; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "<!--") || strings.HasPrefix(line, "@") {
			continue
		}
		if strings.HasPrefix(line, "<") {
			return line, true
		}
	}
	return "", false
}

func extractDataComponent(line string) (bool, string) {
	re := regexp.MustCompile(`data-component\s*=\s*"(.*?)"`)
	match := re.FindStringSubmatch(line)
	if match == nil {
		return false, ""
	}
	return true, match[1]
}

func toKebabCase(name string) string {
	var b strings.Builder
	for i, r := range name {
		if i > 0 && unicode.IsUpper(r) {
			b.WriteByte('-')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

func templFunctionsMissingRenders(path string) []string {
	lines, err := readLines(path)
	if err != nil {
		return nil
	}
	missing := make([]string, 0)
	re := regexp.MustCompile(`^\s*templ\s+([A-Z][A-Za-z0-9_]*)\s*\(`)
	for i, line := range lines {
		match := re.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		if hasRendersComment(lines, i) {
			continue
		}
		missing = append(missing, match[1])
	}
	return missing
}

func hasRendersComment(lines []string, idx int) bool {
	for j := idx - 1; j >= 0; j-- {
		trim := strings.TrimSpace(lines[j])
		if trim == "" {
			continue
		}
		return strings.HasPrefix(trim, "// Renders:")
	}
	return false
}

func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	lines := make([]string, 0)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func parseDoctorGoFile(path string) (*ast.File, error) {
	fset := token.NewFileSet()
	return parser.ParseFile(fset, path, nil, 0)
}

func funcHasEchoContextParam(fn *ast.FuncDecl) bool {
	if fn.Type == nil || fn.Type.Params == nil {
		return false
	}
	for _, field := range fn.Type.Params.List {
		if isEchoContextType(field.Type) {
			return true
		}
	}
	return false
}

func funcHasReceiver(fn *ast.FuncDecl) bool {
	return fn.Recv != nil && len(fn.Recv.List) > 0
}

func funcReturnsOnlyError(fn *ast.FuncDecl) bool {
	if fn.Type == nil || fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
		return false
	}
	return isErrorType(fn.Type.Results.List[0].Type)
}

func isEchoContextType(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return pkg.Name == "echo" && sel.Sel != nil && sel.Sel.Name == "Context"
}

func isErrorType(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "error"
}

func fileHasSelectorCall(file *ast.File, names map[string]struct{}) bool {
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil {
			return true
		}
		if _, ok := names[sel.Sel.Name]; ok {
			found = true
			return false
		}
		return true
	})
	return found
}

func callHasStringLiteralArg(call *ast.CallExpr) bool {
	if len(call.Args) == 0 {
		return false
	}
	lit, ok := call.Args[0].(*ast.BasicLit)
	return ok && lit.Kind == token.STRING
}

func fileHasRouteRegistration(file *ast.File, names map[string]struct{}) bool {
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil {
			return true
		}
		if _, ok := names[sel.Sel.Name]; !ok {
			return true
		}
		if !callHasStringLiteralArg(call) {
			return true
		}
		found = true
		return false
	})
	return found
}

func fileHasInlineSQLCall(file *ast.File, callArgIndex map[string]int) bool {
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil {
			return true
		}
		argIndex, ok := callArgIndex[sel.Sel.Name]
		if !ok || argIndex >= len(call.Args) {
			return true
		}
		if doctorIsSQLLiteral(call.Args[argIndex]) {
			found = true
			return false
		}
		return true
	})
	return found
}

func doctorIsSQLLiteral(expr ast.Expr) bool {
	switch v := expr.(type) {
	case *ast.BasicLit:
		if v.Kind != token.STRING {
			return false
		}
		text := strings.Trim(v.Value, "`\"")
		text = strings.TrimSpace(strings.ToUpper(text))
		return strings.HasPrefix(text, "SELECT ") ||
			strings.HasPrefix(text, "INSERT ") ||
			strings.HasPrefix(text, "UPDATE ") ||
			strings.HasPrefix(text, "DELETE ") ||
			strings.HasPrefix(text, "CREATE ") ||
			strings.HasPrefix(text, "ALTER ") ||
			strings.HasPrefix(text, "DROP ")
	case *ast.BinaryExpr:
		return doctorIsSQLLiteral(v.X) || doctorIsSQLLiteral(v.Y)
	}
	return false
}

func fileHasConfigStruct(file *ast.File) bool {
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
			if _, ok := typeSpec.Type.(*ast.StructType); !ok {
				continue
			}
			if strings.HasSuffix(typeSpec.Name.Name, "Config") {
				return true
			}
		}
	}
	return false
}

func doctorAllowsInlineSQL(rel string) bool {
	if strings.HasSuffix(rel, "_store.go") || strings.HasSuffix(rel, "_store_sql.go") || strings.Contains(rel, "/store_sql.go") {
		return true
	}
	if strings.HasPrefix(rel, "db/gen/") || strings.HasPrefix(rel, "db/queries/") {
		return true
	}
	if strings.HasPrefix(rel, "framework/tests/") {
		return true
	}
	if rel == "app/foundation/container_migrations.go" || rel == "framework/repos/storage/storagerepo.go" {
		return true
	}
	return false
}

func doctorIsAllowedMigrationPath(rel string) bool {
	if strings.HasPrefix(rel, "db/migrate/migrations/") {
		return true
	}
	if strings.HasPrefix(rel, "modules/") && strings.Contains(rel, "/db/migrate/migrations/") {
		return true
	}
	return false
}

type doctorMarkerPair struct {
	file       string
	start      string
	end        string
	missingFix string
	orderFix   string
}

func checkMarkerIntegrity(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	pairs := []doctorMarkerPair{
		{
			file:       filepath.ToSlash(filepath.Join("app", "router.go")),
			start:      "// ship:routes:public:start",
			end:        "// ship:routes:public:end",
			missingFix: "restore route markers in app/router.go to keep generator wiring deterministic",
			orderFix:   "place start marker before end marker to keep generator wiring deterministic",
		},
		{
			file:       filepath.ToSlash(filepath.Join("app", "router.go")),
			start:      "// ship:routes:auth:start",
			end:        "// ship:routes:auth:end",
			missingFix: "restore route markers in app/router.go to keep generator wiring deterministic",
			orderFix:   "place start marker before end marker to keep generator wiring deterministic",
		},
		{
			file:       filepath.ToSlash(filepath.Join("app", "foundation", "container.go")),
			start:      "// ship:container:start",
			end:        "// ship:container:end",
			missingFix: "restore container markers in app/foundation/container.go to keep module wiring deterministic",
			orderFix:   "place ship:container:start before ship:container:end to keep module wiring deterministic",
		},
	}

	fileContents := map[string]string{}
	readFailed := map[string]bool{}
	for _, pair := range pairs {
		path := filepath.Join(root, filepath.FromSlash(pair.file))
		if !hasFile(path) {
			continue
		}
		content, ok := fileContents[pair.file]
		if !ok && !readFailed[pair.file] {
			b, err := os.ReadFile(path)
			if err != nil {
				issues = append(issues, DoctorIssue{
					Code:    "DX004",
					File:    pair.file,
					Message: fmt.Sprintf("failed to read %s for marker checks", pair.file),
					Fix:     err.Error(),
				})
				readFailed[pair.file] = true
				continue
			}
			content = string(b)
			fileContents[pair.file] = content
		}
		if readFailed[pair.file] {
			continue
		}

		hasStart := strings.Contains(content, pair.start)
		hasEnd := strings.Contains(content, pair.end)
		switch {
		case !hasStart && !hasEnd:
			issues = append(issues, DoctorIssue{
				Code:    "DX005",
				File:    pair.file,
				Message: fmt.Sprintf("missing required marker pair in %s: %s / %s", pair.file, pair.start, pair.end),
				Fix:     pair.missingFix,
			})
		case hasStart != hasEnd:
			missing := pair.start
			present := pair.end
			if hasStart {
				missing = pair.end
				present = pair.start
			}
			issues = append(issues, DoctorIssue{
				Code:     "DX005",
				File:     pair.file,
				Message:  fmt.Sprintf("unpaired marker in %s: missing %s for %s", pair.file, missing, present),
				Fix:      pair.missingFix,
				Severity: "warning",
			})
		default:
			startIdx := strings.Index(content, pair.start)
			endIdx := strings.Index(content, pair.end)
			if startIdx > endIdx {
				issues = append(issues, DoctorIssue{
					Code:    "DX011",
					File:    pair.file,
					Message: fmt.Sprintf("marker order invalid in %s: %s appears after %s", pair.file, pair.start, pair.end),
					Fix:     pair.orderFix,
				})
			}
		}
	}

	return issues
}

func checkModulesManifestFormat(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	path := filepath.Join(root, "config", "modules.yaml")
	if !hasFile(path) {
		return issues
	}
	_, err := rt.LoadModulesManifest(path)
	if err != nil {
		issues = append(issues, DoctorIssue{
			Code:    "DX018",
			Message: "invalid config/modules.yaml format",
			Fix:     fmt.Sprintf("use YAML shape `modules: []` with tokens [a-z0-9_-]: %v", err),
		})
	}
	return issues
}

func checkEnabledModuleDBArtifacts(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	path := filepath.Join(root, "config", "modules.yaml")
	if !hasFile(path) {
		return issues
	}
	manifest, err := rt.LoadModulesManifest(path)
	if err != nil {
		return issues
	}

	for _, name := range manifest.Modules {
		moduleRoot := filepath.Join(root, "modules", name)
		if !isDir(moduleRoot) {
			issues = append(issues, DoctorIssue{
				Code:    "DX019",
				Message: fmt.Sprintf("enabled module directory missing: modules/%s", name),
				Fix:     fmt.Sprintf("add modules/%s or remove %q from config/modules.yaml", name, name),
			})
			continue
		}

		migrationsDir := filepath.Join(moduleRoot, "db", "migrate", "migrations")
		if !isDir(migrationsDir) {
			issues = append(issues, DoctorIssue{
				Code:    "DX019",
				Message: fmt.Sprintf("enabled module missing migrations directory: modules/%s/db/migrate/migrations", name),
				Fix:     fmt.Sprintf("add module migrations under modules/%s/db/migrate/migrations", name),
			})
		}

		bobgenPath := filepath.Join(moduleRoot, "db", "bobgen.yaml")
		if !hasFile(bobgenPath) {
			issues = append(issues, DoctorIssue{
				Code:    "DX019",
				Message: fmt.Sprintf("enabled module missing bobgen config: modules/%s/db/bobgen.yaml", name),
				Fix:     fmt.Sprintf("add modules/%s/db/bobgen.yaml", name),
			})
		}
	}

	return issues
}

func checkForbiddenCrossBoundaryImports(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)

	// Controllers must not directly import DB implementation packages.
	controllerDir := filepath.Join(root, "app", "web", "controllers")
	issues = append(issues, checkImportPrefixForbidden(controllerDir, "github.com/leomorpho/goship/db/gen", "DX020",
		"controller db boundary violated: app/web/controllers must not import db/gen directly",
		"move DB access behind foundation/service seams or auth/profile helpers")...)

	// Controllers must not call QueryProfile() directly.
	issues = append(issues, checkTextForbiddenInDir(controllerDir, "QueryProfile(", "DX020",
		"controller auth boundary violated: direct QueryProfile() usage is not allowed in app/web/controllers",
		"use middleware auth identity keys + service/store lookup by id")...)

	for _, path := range []string{
		filepath.Join(root, "modules", "jobs", "config.go"),
		filepath.Join(root, "modules", "jobs", "module.go"),
		filepath.Join(root, "modules", "jobs", "drivers", "sql", "client.go"),
	} {
		issues = append(issues, checkTextForbidden(path, "github.com/leomorpho/goship/db/gen", "DX020",
			fmt.Sprintf("jobs SQL boundary violated: db/gen import found in %s", filepath.ToSlash(mustRel(root, path))),
			"keep jobs SQL path module-local and adapter-agnostic")...)
	}

	// Notifications module must not depend on framework/core directly for pubsub contracts.
	for _, path := range []string{
		filepath.Join(root, "modules", "notifications", "module.go"),
		filepath.Join(root, "modules", "notifications", "notifier.go"),
		filepath.Join(root, "modules", "notifications", "notifier_test.go"),
	} {
		issues = append(issues, checkTextForbidden(path, "github.com/leomorpho/goship/framework/core", "DX020",
			fmt.Sprintf("notifications pubsub boundary violated: framework/core import found in %s", filepath.ToSlash(mustRel(root, path))),
			"use module-local contracts and app-level bridge adapters")...)
	}

	// Module isolation: no direct imports from root app/framework packages, except explicit allowlist paths.
	issues = append(issues, checkModuleSourceIsolation(root)...)

	return issues
}

func checkModuleSourceIsolation(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	modulesRoot := filepath.Join(root, "modules")
	if !isDir(modulesRoot) {
		return issues
	}
	allowlist := loadModuleIsolationAllowlist(filepath.Join(root, "tools", "scripts", "test", "module-isolation-allowlist.txt"))
	_ = filepath.WalkDir(modulesRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel := filepath.ToSlash(mustRel(root, path))
		if _, ok := allowlist[rel]; ok {
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			issues = append(issues, DoctorIssue{
				Code:    "DX020",
				Message: fmt.Sprintf("failed reading file for module isolation check: %s", rel),
				Fix:     readErr.Error(),
			})
			return nil
		}
		if strings.Contains(string(b), "\"github.com/leomorpho/goship/") {
			issues = append(issues, DoctorIssue{
				Code:    "DX020",
				Message: fmt.Sprintf("module isolation violated: forbidden root import in %s", rel),
				Fix:     "remove direct github.com/leomorpho/goship/* imports from module runtime code or add a deliberate allowlist entry",
			})
		}
		return nil
	})
	return issues
}

func loadModuleIsolationAllowlist(path string) map[string]struct{} {
	result := map[string]struct{}{}
	if !hasFile(path) {
		return result
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return result
	}
	for _, raw := range strings.Split(string(b), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		result[filepath.ToSlash(line)] = struct{}{}
	}
	return result
}

func checkImportPrefixForbidden(dir string, forbiddenPrefix string, code string, message string, fix string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	if !isDir(dir) {
		return issues
	}
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		if strings.Contains(string(b), "\""+forbiddenPrefix) {
			issues = append(issues, DoctorIssue{
				Code:    code,
				Message: message,
				Fix:     fix,
			})
		}
		return nil
	})
	return issues
}

func checkTextForbidden(path string, token string, code string, message string, fix string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	if !hasFile(path) {
		return issues
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return append(issues, DoctorIssue{
			Code:    code,
			Message: fmt.Sprintf("failed to read boundary file: %s", filepath.ToSlash(path)),
			Fix:     err.Error(),
		})
	}
	if strings.Contains(string(b), token) {
		issues = append(issues, DoctorIssue{
			Code:    code,
			Message: message,
			Fix:     fix,
		})
	}
	return issues
}

func checkTextForbiddenInDir(dir string, token string, code string, message string, fix string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	if !isDir(dir) {
		return issues
	}
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		if strings.Contains(string(b), token) {
			issues = append(issues, DoctorIssue{
				Code:    code,
				Message: message,
				Fix:     fix,
			})
		}
		return nil
	})
	return issues
}

func mustRel(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}

func checkPackageNaming(root, relDir, expected string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	dir := filepath.Join(root, relDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return issues
		}
		return append(issues, DoctorIssue{
			Code:    "DX006",
			Message: fmt.Sprintf("failed reading package directory %s", relDir),
			Fix:     err.Error(),
		})
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		filePath := filepath.Join(dir, entry.Name())
		pkg, err := readPackageClause(filePath)
		if err != nil {
			issues = append(issues, DoctorIssue{
				Code:    "DX006",
				Message: fmt.Sprintf("failed reading package clause in %s", filepath.ToSlash(filepath.Join(relDir, entry.Name()))),
				Fix:     err.Error(),
			})
			continue
		}

		allowed := map[string]struct{}{expected: {}}
		if strings.HasSuffix(entry.Name(), "_test.go") {
			allowed[expected+"_test"] = struct{}{}
		}
		if _, ok := allowed[pkg]; !ok {
			issues = append(issues, DoctorIssue{
				Code:    "DX007",
				Message: fmt.Sprintf("package mismatch in %s: got %q, want %q (or %q for tests)", filepath.ToSlash(filepath.Join(relDir, entry.Name())), pkg, expected, expected+"_test"),
				Fix:     "align package name with directory convention",
			})
		}
	}

	return issues
}

func readPackageClause(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(b), "\n")
	for _, line := range lines {
		s := strings.TrimSpace(line)
		if s == "" || strings.HasPrefix(s, "//") {
			continue
		}
		if strings.HasPrefix(s, "package ") {
			return strings.TrimSpace(strings.TrimPrefix(s, "package ")), nil
		}
		break
	}
	return "", fmt.Errorf("package clause not found")
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func hasFile(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func checkFileSizes(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	hardCapAllowlist := map[string]struct{}{
		filepath.ToSlash(filepath.Join("tools", "cli", "ship", "internal", "policies", "doctor.go")): {},
		filepath.ToSlash(filepath.Join("app", "views", "web", "pages", "home_feed.templ")):           {},
		filepath.ToSlash(filepath.Join("app", "views", "web", "pages", "landing_page.templ")):        {},
		filepath.ToSlash(filepath.Join("app", "views", "web", "pages", "preferences.templ")):         {},
		filepath.ToSlash(filepath.Join("app", "views", "emails", "password_reset.templ")):            {},
		filepath.ToSlash(filepath.Join("app", "views", "emails", "registration_confirmation.templ")): {},
		filepath.ToSlash(filepath.Join("app", "views", "emails", "update.templ")):                    {},
	}

	scanRoots := []string{
		filepath.Join(root, "app"),
		filepath.Join(root, "framework"),
		filepath.Join(root, "tools"),
		filepath.Join(root, "config"),
	}
	for _, scanRoot := range scanRoots {
		if !isDir(scanRoot) {
			continue
		}
		_ = filepath.WalkDir(scanRoot, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			rel := filepath.ToSlash(mustRel(root, path))

			if d.IsDir() {
				if rel == "vendor" ||
					strings.HasPrefix(rel, "vendor/") ||
					rel == ".git" ||
					rel == "node_modules" ||
					rel == ".cache" ||
					filepath.Base(rel) == ".cache" ||
					strings.Contains(rel, "/.cache/") ||
					strings.HasSuffix(rel, "/gen") {
					return filepath.SkipDir
				}
				return nil
			}

			kind, warnThreshold, errorThreshold, skip := doctorFileSizeKind(rel)
			if skip {
				return nil
			}

			lines, lineErr := countNonBlankLines(path)
			if lineErr != nil {
				issues = append(issues, DoctorIssue{
					Code:    "DX010",
					File:    rel,
					Message: fmt.Sprintf("failed counting non-blank lines for %s", rel),
					Fix:     lineErr.Error(),
				})
				return nil
			}
			if lines <= warnThreshold {
				return nil
			}

			severity := "warning"
			message := fmt.Sprintf("%s file exceeds recommended size (%d > %d non-blank lines): %s", kind, lines, warnThreshold, rel)
			if lines > errorThreshold {
				if _, ok := hardCapAllowlist[rel]; ok {
					message = fmt.Sprintf("%s file exceeds hard size cap but is grandfathered (%d > %d non-blank lines): %s", kind, lines, errorThreshold, rel)
				} else {
					severity = "error"
					message = fmt.Sprintf("%s file exceeds hard size cap (%d > %d non-blank lines): %s", kind, lines, errorThreshold, rel)
				}
			}

			issues = append(issues, DoctorIssue{
				Code:     "DX010",
				File:     rel,
				Message:  message,
				Fix:      "split by responsibility to keep files LLM-friendly",
				Severity: severity,
			})
			return nil
		})
	}

	return issues
}

func doctorFileSizeKind(rel string) (kind string, warnThreshold int, errorThreshold int, skip bool) {
	switch {
	case strings.HasSuffix(rel, ".go"):
		if strings.HasSuffix(rel, "_test.go") ||
			strings.HasSuffix(rel, ".templ.go") ||
			strings.HasSuffix(rel, "_sql.go") ||
			strings.HasPrefix(filepath.Base(rel), "bob_") {
			return "", 0, 0, true
		}
		return "Go", 800, 1000, false
	case strings.HasSuffix(rel, ".templ"):
		return "templ", 600, 800, false
	default:
		return "", 0, 0, true
	}
}

func checkTopLevelDirs(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	entries, err := os.ReadDir(root)
	if err != nil {
		return append(issues, DoctorIssue{
			Code:    "DX013",
			Message: "failed to read repository root",
			Fix:     err.Error(),
		})
	}

	allowed := map[string]struct{}{
		".cache":     {},
		".git":       {},
		".github":    {},
		".githooks":  {},
		".kamal":     {},
		".vscode":    {},
		"app":        {},
		"db":         {},
		"cmd":        {},
		"config":     {},
		"data":       {},
		"dbs":        {},
		"docs":       {},
		"framework":  {},
		"infra":      {},
		"javascript": {},
		"modules":    {},
		"tests":      {},
		"tmp":        {},
		"tools":      {},
		"frontend":   {},
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if _, ok := allowed[name]; !ok {
			issues = append(issues, DoctorIssue{
				Code:    "DX013",
				Message: fmt.Sprintf("unexpected top-level directory: %s", name),
				Fix:     "move it under app/, db/, cmd/, modules/, framework/, tools/, infra/, tests/, or mark as intentional in doctor allow-list",
			})
		}
	}

	return issues
}

func countNonBlankLines(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	lines := 0
	for s.Scan() {
		if strings.TrimSpace(s.Text()) == "" {
			continue
		}
		lines++
	}
	if err := s.Err(); err != nil {
		return 0, err
	}
	return lines, nil
}

func checkCLIDocsCoverage(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	cliRefPath := filepath.Join(root, "docs", "reference", "01-cli.md")
	if !hasFile(cliRefPath) {
		return issues
	}
	b, err := os.ReadFile(cliRefPath)
	if err != nil {
		return append(issues, DoctorIssue{
			Code:    "DX012",
			Message: "failed to read docs/reference/01-cli.md",
			Fix:     err.Error(),
		})
	}
	text := string(b)
	requiredSections := []string{
		"## Minimal V1 Command Set",
		"## Implementation Mapping (Current Repo)",
		"## Generator test strategy",
	}
	for _, section := range requiredSections {
		if !strings.Contains(text, section) {
			issues = append(issues, DoctorIssue{
				Code:    "DX012",
				Message: fmt.Sprintf("cli docs missing required section: %q", section),
				Fix:     "restore required sections in docs/reference/01-cli.md",
			})
		}
	}

	required := []string{
		"ship doctor",
		"ship agent:setup",
		"ship agent:check",
		"ship agent:status",
		"ship new <app>",
		"ship upgrade",
		"ship make:resource",
		"ship make:model",
		"ship make:controller",
		"ship make:scaffold",
		"ship make:module",
		"ship db:migrate",
		"ship test --integration",
	}
	for _, token := range required {
		if !strings.Contains(text, token) {
			issues = append(issues, DoctorIssue{
				Code:    "DX012",
				Message: fmt.Sprintf("cli docs missing required command token: %q", token),
				Fix:     "update docs/reference/01-cli.md to cover implemented core commands",
			})
		}
	}
	return issues
}

func checkGoWorkModules(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	goWorkPath := filepath.Join(root, "go.work")
	if !hasFile(goWorkPath) {
		return issues
	}
	b, err := os.ReadFile(goWorkPath)
	if err != nil {
		return append(issues, DoctorIssue{
			Code:    "DX014",
			Message: "failed to read go.work",
			Fix:     err.Error(),
		})
	}
	modules := parseGoWorkUseModules(string(b))
	for _, modPath := range modules {
		p := filepath.Clean(filepath.Join(root, filepath.FromSlash(modPath)))
		if !hasFile(filepath.Join(p, "go.mod")) {
			issues = append(issues, DoctorIssue{
				Code:    "DX014",
				Message: fmt.Sprintf("go.work references missing module go.mod: %s", modPath),
				Fix:     fmt.Sprintf("create %s/go.mod or remove %s from go.work use()", filepath.ToSlash(filepath.Join(modPath)), modPath),
			})
		}
	}
	return issues
}

func parseGoWorkUseModules(content string) []string {
	modules := make([]string, 0)
	lines := strings.Split(content, "\n")
	inUseBlock := false
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if strings.HasPrefix(line, "use ") {
			rest := strings.TrimSpace(strings.TrimPrefix(line, "use"))
			if rest == "(" {
				inUseBlock = true
				continue
			}
			rest = trimInlineComment(rest)
			rest = strings.Trim(rest, "\"")
			if rest != "" {
				modules = append(modules, rest)
			}
			continue
		}
		if inUseBlock {
			if line == ")" {
				inUseBlock = false
				continue
			}
			line = trimInlineComment(line)
			line = strings.Trim(line, "\"")
			if line != "" {
				modules = append(modules, line)
			}
		}
	}
	return modules
}

func checkDockerIgnoreCoverage(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	if !hasFile(filepath.Join(root, "infra", "docker", "Dockerfile")) {
		return issues
	}
	path := filepath.Join(root, ".dockerignore")
	if !hasFile(path) {
		return append(issues, DoctorIssue{
			Code:    "DX015",
			Message: "missing .dockerignore",
			Fix:     "add .dockerignore with heavy-path exclusions to keep docker build context small",
		})
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return append(issues, DoctorIssue{
			Code:    "DX015",
			Message: "failed to read .dockerignore",
			Fix:     err.Error(),
		})
	}
	text := string(b)
	requiredEntries := []string{
		".git",
		"node_modules",
		"frontend/node_modules",
		"tmp",
		"tools/scripts/venv",
	}
	for _, entry := range requiredEntries {
		if !containsDockerIgnoreEntry(text, entry) {
			issues = append(issues, DoctorIssue{
				Code:    "DX015",
				Message: fmt.Sprintf(".dockerignore missing required context exclusion: %s", entry),
				Fix:     "add required exclusion to keep docker build context small and stable",
			})
		}
	}
	return issues
}

func containsDockerIgnoreEntry(content, token string) bool {
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if line == token || line == "/"+token {
			return true
		}
	}
	return false
}

func checkDockerLocalReplaceOrder(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	dockerfile := filepath.Join(root, "infra", "docker", "Dockerfile")
	if !hasFile(dockerfile) {
		return issues
	}
	localReplaces := collectLocalReplaces(root)
	if len(localReplaces) == 0 {
		return issues
	}
	b, err := os.ReadFile(dockerfile)
	if err != nil {
		return append(issues, DoctorIssue{
			Code:    "DX016",
			Message: "failed to read infra/docker/Dockerfile",
			Fix:     err.Error(),
		})
	}
	lines := strings.Split(string(b), "\n")
	downloadIdx := -1
	copyAllIdx := -1
	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if downloadIdx == -1 && strings.Contains(line, "go mod download") {
			downloadIdx = i
		}
		if copyAllIdx == -1 && strings.HasPrefix(line, "COPY ") && strings.Contains(line, ". .") {
			copyAllIdx = i
		}
	}
	if downloadIdx == -1 {
		return append(issues, DoctorIssue{
			Code:    "DX016",
			Message: "Dockerfile does not run go mod download",
			Fix:     "add a deterministic go mod download step in builder stage",
		})
	}
	if copyAllIdx != -1 && copyAllIdx < downloadIdx {
		return issues
	}
	for _, rel := range localReplaces {
		found := false
		for i, raw := range lines {
			if i >= downloadIdx {
				break
			}
			line := strings.TrimSpace(raw)
			if !strings.HasPrefix(line, "COPY ") {
				continue
			}
			if strings.Contains(line, rel) || strings.Contains(line, filepath.ToSlash(rel)) {
				found = true
				break
			}
		}
		if !found {
			issues = append(issues, DoctorIssue{
				Code:    "DX016",
				Message: fmt.Sprintf("Dockerfile may fail local replace before go mod download: missing COPY for %s", rel),
				Fix:     "copy local replace paths (or COPY . .) before the first go mod download",
			})
		}
	}
	return issues
}

func checkAgentPolicyArtifacts(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	policyPath := filepath.Join(root, AgentPolicyFilePath)
	if !hasFile(policyPath) {
		return append(issues, DoctorIssue{
			Code:    "DX017",
			Message: fmt.Sprintf("missing agent policy file: %s", filepath.ToSlash(AgentPolicyFilePath)),
			Fix:     "add tools/agent-policy/allowed-commands.yaml and run ship agent:setup",
		})
	}
	policy, err := LoadPolicy(policyPath)
	if err != nil {
		return append(issues, DoctorIssue{
			Code:    "DX017",
			Message: "invalid agent policy file",
			Fix:     err.Error(),
		})
	}
	expected, err := RenderPolicyArtifacts(policy)
	if err != nil {
		return append(issues, DoctorIssue{
			Code:    "DX017",
			Message: "failed to render agent policy artifacts",
			Fix:     err.Error(),
		})
	}
	drifted, err := DiffPolicyArtifacts(root, expected)
	if err != nil {
		return append(issues, DoctorIssue{
			Code:    "DX017",
			Message: "failed to compare generated agent artifacts",
			Fix:     err.Error(),
		})
	}
	for _, rel := range drifted {
		issues = append(issues, DoctorIssue{
			Code:    "DX017",
			Message: fmt.Sprintf("agent artifact out of sync: %s", rel),
			Fix:     "run ship agent:setup",
		})
	}
	return issues
}

func printDoctorHelp(w io.Writer) {
	fmt.Fprintln(w, "ship doctor commands:")
	fmt.Fprintln(w, "  ship doctor [--json]")
	fmt.Fprintln(w, "  (validates canonical app structure and LLM/DX conventions)")
}

func collectLocalReplaces(root string) []string {
	paths := make([]string, 0)
	seen := map[string]struct{}{}
	goModFiles := []string{
		filepath.Join(root, "go.mod"),
	}
	for _, gm := range goModFiles {
		if !hasFile(gm) {
			continue
		}
		moduleRoot := filepath.Dir(gm)
		for _, p := range parseLocalReplacePaths(gm) {
			abs := filepath.Clean(filepath.Join(moduleRoot, filepath.FromSlash(p)))
			rel, err := filepath.Rel(root, abs)
			if err != nil {
				continue
			}
			rel = filepath.ToSlash(rel)
			if strings.HasPrefix(rel, "..") {
				continue
			}
			if _, ok := seen[rel]; ok {
				continue
			}
			seen[rel] = struct{}{}
			paths = append(paths, rel)
		}
	}
	return paths
}

func parseLocalReplacePaths(goModPath string) []string {
	b, err := os.ReadFile(goModPath)
	if err != nil {
		return nil
	}
	paths := make([]string, 0)
	inReplaceBlock := false
	replaceRe := regexp.MustCompile(`\s+=>\s+([^\s]+)`)
	lines := strings.Split(string(b), "\n")
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if strings.HasPrefix(line, "replace ") {
			rest := strings.TrimSpace(strings.TrimPrefix(line, "replace"))
			if rest == "(" {
				inReplaceBlock = true
				continue
			}
			if p := parseReplacePath(rest, replaceRe); p != "" {
				paths = append(paths, p)
			}
			continue
		}
		if inReplaceBlock {
			if line == ")" {
				inReplaceBlock = false
				continue
			}
			if p := parseReplacePath(line, replaceRe); p != "" {
				paths = append(paths, p)
			}
		}
	}
	return paths
}

func parseReplacePath(line string, re *regexp.Regexp) string {
	line = trimInlineComment(line)
	m := re.FindStringSubmatch(line)
	if len(m) != 2 {
		return ""
	}
	p := strings.TrimSpace(strings.Trim(m[1], "\""))
	if strings.HasPrefix(p, ".") {
		return filepath.ToSlash(p)
	}
	return ""
}

func trimInlineComment(line string) string {
	if idx := strings.Index(line, "//"); idx >= 0 {
		return strings.TrimSpace(line[:idx])
	}
	return strings.TrimSpace(line)
}
