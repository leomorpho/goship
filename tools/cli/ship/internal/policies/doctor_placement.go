package policies

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	appconfig "github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/core/adapters"
	rt "github.com/leomorpho/goship/tools/cli/ship/internal/runtime"
)

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

func checkRequiredConfigEnv(root string) []DoctorIssue {
	missing, err := appconfig.MissingRequiredEnv(root)
	if err != nil {
		return []DoctorIssue{{
			Code:    "DX022",
			Message: "failed to validate required config environment variables",
			Fix:     err.Error(),
		}}
	}
	if len(missing) == 0 {
		return checkConfigEnvSemantics(root)
	}

	issues := make([]DoctorIssue, 0, len(missing))
	for _, item := range missing {
		issues = append(issues, DoctorIssue{
			Code:    "DX022",
			Message: fmt.Sprintf("missing required config environment variable: %s", item.Name),
			Fix:     "set it in the shell or add it to .env",
		})
	}
	issues = append(issues, checkConfigEnvSemantics(root)...)
	return issues
}

func checkConfigEnvSemantics(root string) []DoctorIssue {
	values, err := loadDoctorEnvValues(root)
	if err != nil {
		return []DoctorIssue{{
			Code:    "DX022",
			Message: "failed to read config environment values for semantic validation",
			Fix:     err.Error(),
		}}
	}

	issues := make([]DoctorIssue, 0, 2)
	if raw := strings.TrimSpace(values["PAGODA_APP_FIREBASEBASE64ACCESSKEYS"]); raw != "" {
		if _, decodeErr := base64.StdEncoding.DecodeString(raw); decodeErr != nil {
			issues = append(issues, DoctorIssue{
				Code:    "DX022",
				Message: fmt.Sprintf("invalid config secret PAGODA_APP_FIREBASEBASE64ACCESSKEYS: %v", decodeErr),
				Fix:     "set PAGODA_APP_FIREBASEBASE64ACCESSKEYS to a valid base64-encoded payload or unset it",
			})
		}
	}

	selection := adapters.Selection{
		DB:     "sqlite",
		Cache:  "otter",
		Jobs:   "backlite",
		PubSub: "inproc",
	}
	if value := strings.TrimSpace(values["PAGODA_ADAPTERS_DB"]); value != "" {
		selection.DB = value
	}
	if value := firstNonEmptyEnvValue(values, "PAGODA_ADAPTERS_CACHE", "PAGODA_CACHE_DRIVER"); value != "" {
		selection.Cache = value
	}
	if value := firstNonEmptyEnvValue(values, "PAGODA_ADAPTERS_JOBS", "PAGODA_JOBS_DRIVER"); value != "" {
		selection.Jobs = value
	}
	if value := strings.TrimSpace(values["PAGODA_ADAPTERS_PUBSUB"]); value != "" {
		selection.PubSub = value
	}
	if selectionErr := adapters.NewDefaultRegistry().ValidateSelection(selection); selectionErr != nil {
		issues = append(issues, DoctorIssue{
			Code:    "DX022",
			Message: fmt.Sprintf("invalid adapter config: %v", selectionErr),
			Fix:     "set PAGODA_ADAPTERS_DB/PAGODA_ADAPTERS_CACHE/PAGODA_ADAPTERS_JOBS/PAGODA_ADAPTERS_PUBSUB to supported values",
		})
	}

	if cfg, cfgErr := appconfig.GetConfig(); cfgErr != nil {
		issues = append(issues, DoctorIssue{
			Code:    "DX022",
			Message: "failed to load config for semantic validation",
			Fix:     cfgErr.Error(),
		})
	} else {
		for _, issue := range appconfig.ValidateConfigSemantics(cfg) {
			issues = append(issues, DoctorIssue{
				Code:    "DX022",
				Message: fmt.Sprintf("invalid config semantics: %s", issue.Error()),
				Fix:     "set concrete runtime values that satisfy the selected driver/profile requirements",
			})
		}
	}

	return issues
}

func loadDoctorEnvValues(root string) (map[string]string, error) {
	values := map[string]string{}

	envPath := filepath.Join(root, ".env")
	if hasFile(envPath) {
		file, err := os.Open(envPath)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if strings.HasPrefix(line, "export ") {
				line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
			}
			key, value, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			values[key] = strings.TrimSpace(value)
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}

	for _, raw := range os.Environ() {
		key, value, ok := strings.Cut(raw, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		values[key] = strings.TrimSpace(value)
	}

	return values, nil
}

func firstNonEmptyEnvValue(values map[string]string, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(values[key]); value != "" {
			return value
		}
	}
	return ""
}

func runNilawayChecks(root string, d DoctorDeps) []DoctorIssue {
	lookPath := d.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	if _, err := lookPath("nilaway"); err != nil {
		return nil
	}

	runCmd := d.RunCmd
	if runCmd == nil {
		runCmd = defaultDoctorRunCmd
	}

	code, output, err := runCmd(root, "nilaway", "./...")
	if code == 0 && err == nil {
		return nil
	}

	text := strings.TrimSpace(output)
	if err != nil && text == "" {
		text = err.Error()
	}
	if text == "" {
		text = "nilaway reported issues"
	}

	lines := strings.Split(text, "\n")
	issues := make([]DoctorIssue, 0, len(lines))
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		issue := DoctorIssue{
			Code:     "DX025",
			Message:  "nilaway: " + line,
			Fix:      "run nilaway ./... and remove the nil flow or add a justified suppression",
			Severity: "warning",
		}

		if file := parseNilawayFile(root, line); file != "" {
			issue.File = file
		}
		issues = append(issues, issue)
	}
	if len(issues) == 0 {
		return []DoctorIssue{{
			Code:     "DX025",
			Message:  "nilaway reported issues",
			Fix:      "run nilaway ./... and inspect the analyzer output",
			Severity: "warning",
		}}
	}
	return issues
}

func parseNilawayFile(root, line string) string {
	parts := strings.Split(line, ":")
	if len(parts) == 0 {
		return ""
	}

	candidate := strings.TrimSpace(parts[0])
	if candidate == "" {
		return ""
	}

	if rel, err := filepath.Rel(root, candidate); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel)
	}

	normalized := filepath.Clean(candidate)
	if filepath.IsAbs(normalized) {
		return ""
	}
	if hasFile(filepath.Join(root, normalized)) {
		return filepath.ToSlash(normalized)
	}
	return ""
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
			if err != nil {
				return nil
			}
			rel := filepath.ToSlash(mustRel(root, path))
			if d.IsDir() {
				if rel == ".cache" || filepath.Base(rel) == ".cache" || strings.Contains(rel, "/.cache/") {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
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
			if rel == ".git" || rel == ".worktrees" || rel == "node_modules" || rel == ".cache" || strings.Contains(rel, "/.cache/") {
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
		if doctorAllowsConfigStructFile(rel) {
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

func doctorAllowsConfigStructFile(rel string) bool {
	if rel == "config/config.go" {
		return true
	}
	if !strings.HasPrefix(rel, "config/") {
		return false
	}
	base := filepath.Base(rel)
	return strings.HasPrefix(base, "config_") && strings.HasSuffix(base, ".go")
}

func checkRendersComments(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	searchDirs := []string{filepath.Join(root, "app", "views")}
	modulesConfig := filepath.Join(root, "config", "modules.yaml")
	if manifest, err := rt.LoadModulesManifest(modulesConfig); err == nil {
		for _, moduleID := range manifest.Modules {
			viewsDir := filepath.Join(root, "modules", moduleID, "views")
			if isDir(viewsDir) {
				searchDirs = append(searchDirs, viewsDir)
			}
		}
	}

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
	type searchTarget struct {
		dir              string
		skipLayoutSuffix bool
	}
	searchTargets := []searchTarget{
		{dir: filepath.Join(root, "app", "views", "web", "components"), skipLayoutSuffix: true},
		{dir: filepath.Join(root, "app", "views", "web", "layouts"), skipLayoutSuffix: false},
	}
	modulesConfig := filepath.Join(root, "config", "modules.yaml")
	if manifest, err := rt.LoadModulesManifest(modulesConfig); err == nil {
		for _, moduleID := range manifest.Modules {
			compDir := filepath.Join(root, "modules", moduleID, "views", "web", "components")
			if isDir(compDir) {
				searchTargets = append(searchTargets, searchTarget{dir: compDir, skipLayoutSuffix: true})
			}
			layoutDir := filepath.Join(root, "modules", moduleID, "views", "web", "layouts")
			if isDir(layoutDir) {
				searchTargets = append(searchTargets, searchTarget{dir: layoutDir, skipLayoutSuffix: false})
			}
		}
	}

	for _, target := range searchTargets {
		if !isDir(target.dir) {
			continue
		}
		_ = filepath.WalkDir(target.dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || filepath.Ext(path) != ".templ" {
				return nil
			}
			if target.skipLayoutSuffix && strings.HasSuffix(filepath.Base(path), "_layout.templ") {
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
		if !strings.HasPrefix(trim, "//") {
			return false
		}
		if strings.HasPrefix(trim, "// Renders:") {
			return true
		}
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
	if strings.HasPrefix(rel, "modules/") && strings.HasSuffix(rel, "/store.go") {
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
