package policies

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	rt "github.com/leomorpho/goship/tools/cli/ship/internal/runtime"
)

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
			file:       filepath.ToSlash(filepath.Join("app", "router.go")),
			start:      "// ship:routes:external:start",
			end:        "// ship:routes:external:end",
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
				Code:    "DX005",
				File:    pair.file,
				Message: fmt.Sprintf("unpaired marker in %s: missing %s for %s", pair.file, missing, present),
				Fix:     pair.missingFix,
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

	// Framework/runtime code must stay decoupled from control-plane source imports.
	for _, scanDir := range []string{
		filepath.Join(root, "app"),
		filepath.Join(root, "cmd"),
		filepath.Join(root, "config"),
		filepath.Join(root, "framework"),
		filepath.Join(root, "modules"),
	} {
		for _, forbiddenPrefix := range []string{
			"github.com/leomorpho/goship/tools/private/control-plane",
			"github.com/leomorpho/goship/fleet/control-plane",
		} {
			issues = append(issues, checkGoImportPrefixForbidden(
				scanDir,
				forbiddenPrefix,
				"DX020",
				"control-plane source coupling violated: runtime code must not import control-plane source packages",
				"remove control-plane imports and keep runtime/framework boundaries control-plane agnostic",
			)...)
		}
	}
	for _, scanDir := range []string{
		filepath.Join(root, "app"),
		filepath.Join(root, "cmd"),
		filepath.Join(root, "config"),
		filepath.Join(root, "framework"),
		filepath.Join(root, "modules"),
	} {
		for _, forbiddenToken := range []string{
			"tools/private/control-plane",
			"fleet/control-plane",
		} {
			issues = append(issues, checkTextForbiddenInDirNonTest(
				scanDir,
				forbiddenToken,
				"DX020",
				"control-plane private path assumption violated: runtime code must not hardcode private control-plane paths",
				"remove private control-plane path assumptions and route contracts through runtime-owned shared seams",
			)...)
		}
	}

	return issues
}

func checkContractUsage(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	controllerDir := filepath.Join(root, "app", "web", "controllers")
	if !isDir(controllerDir) {
		return issues
	}

	_ = filepath.WalkDir(controllerDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return nil
		}

		contractAliases := doctorContractsImportAliases(file)
		localStructTypes := doctorLocalStructTypeNames(file)
		contractVars := map[string]struct{}{}
		rawBind := false
		rawFormValue := false

		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.ValueSpec:
				if doctorExprUsesContractsType(node.Type, contractAliases, localStructTypes) {
					for _, name := range node.Names {
						contractVars[name.Name] = struct{}{}
					}
				}
				for i, value := range node.Values {
					if doctorExprUsesContractsType(value, contractAliases, localStructTypes) && i < len(node.Names) {
						contractVars[node.Names[i].Name] = struct{}{}
					}
				}
			case *ast.AssignStmt:
				upper := len(node.Rhs)
				if len(node.Lhs) < upper {
					upper = len(node.Lhs)
				}
				for i := 0; i < upper; i++ {
					if !doctorExprUsesContractsType(node.Rhs[i], contractAliases, localStructTypes) {
						continue
					}
					if ident, ok := node.Lhs[i].(*ast.Ident); ok {
						contractVars[ident.Name] = struct{}{}
					}
				}
			case *ast.CallExpr:
				sel, ok := node.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel == nil {
					return true
				}
				switch sel.Sel.Name {
				case "FormValue":
					rawFormValue = true
				case "Bind":
					if len(node.Args) == 0 || !doctorBindArgUsesContractsType(node.Args[0], contractVars, contractAliases, localStructTypes) {
						rawBind = true
					}
				}
			}
			return true
		})

		if !rawBind && !rawFormValue {
			return nil
		}

		detail := "controller uses raw form parsing without typed request contract structs"
		switch {
		case rawBind && !rawFormValue:
			detail = "controller uses Bind without typed request contract structs"
		case rawFormValue && !rawBind:
			detail = "controller uses FormValue directly (prefer typed request contract structs)"
		}

		issues = append(issues, DoctorIssue{
			Code:    "DX027",
			File:    filepath.ToSlash(mustRel(root, path)),
			Message: detail,
			Fix:     "bind/parse request payloads into typed request structs (local or owned contracts packages) instead of raw FormValue/untyped map payloads",
		})
		return nil
	})

	return issues
}

func doctorContractsImportAliases(file *ast.File) map[string]struct{} {
	aliases := map[string]struct{}{}
	if file == nil {
		return aliases
	}

	for _, imp := range file.Imports {
		if imp == nil || imp.Path == nil {
			continue
		}
		path := strings.Trim(imp.Path.Value, "\"")
		if !doctorIsContractsImportPath(path) {
			continue
		}

		alias := "contracts"
		if imp.Name != nil {
			if imp.Name.Name == "_" || imp.Name.Name == "." {
				continue
			}
			alias = imp.Name.Name
		}
		aliases[alias] = struct{}{}
	}
	return aliases
}

func doctorIsContractsImportPath(path string) bool {
	clean := strings.Trim(path, "/")
	if clean == "" {
		return false
	}
	if clean == "contracts" {
		return true
	}
	return strings.HasSuffix(clean, "/contracts") || strings.Contains(clean, "/contracts/")
}

func doctorLocalStructTypeNames(file *ast.File) map[string]struct{} {
	names := map[string]struct{}{}
	if file == nil {
		return names
	}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name == nil {
				continue
			}
			if _, ok := typeSpec.Type.(*ast.StructType); ok {
				names[typeSpec.Name.Name] = struct{}{}
			}
		}
	}
	return names
}

func doctorExprUsesContractsType(expr ast.Expr, aliases map[string]struct{}, localStructTypes map[string]struct{}) bool {
	if expr == nil {
		return false
	}

	switch node := expr.(type) {
	case *ast.ParenExpr:
		return doctorExprUsesContractsType(node.X, aliases, localStructTypes)
	case *ast.StarExpr:
		return doctorExprUsesContractsType(node.X, aliases, localStructTypes)
	case *ast.UnaryExpr:
		return doctorExprUsesContractsType(node.X, aliases, localStructTypes)
	case *ast.CompositeLit:
		return doctorExprUsesContractsType(node.Type, aliases, localStructTypes)
	case *ast.SelectorExpr:
		ident, ok := node.X.(*ast.Ident)
		if !ok {
			return false
		}
		_, ok = aliases[ident.Name]
		return ok
	case *ast.Ident:
		_, ok := localStructTypes[node.Name]
		return ok
	case *ast.IndexExpr:
		return doctorExprUsesContractsType(node.X, aliases, localStructTypes)
	case *ast.IndexListExpr:
		return doctorExprUsesContractsType(node.X, aliases, localStructTypes)
	default:
		return false
	}
}

func doctorBindArgUsesContractsType(arg ast.Expr, contractVars map[string]struct{}, aliases map[string]struct{}, localStructTypes map[string]struct{}) bool {
	if doctorExprUsesContractsType(arg, aliases, localStructTypes) {
		return true
	}

	switch node := arg.(type) {
	case *ast.ParenExpr:
		return doctorBindArgUsesContractsType(node.X, contractVars, aliases, localStructTypes)
	case *ast.UnaryExpr:
		ident, ok := node.X.(*ast.Ident)
		if !ok {
			return false
		}
		_, ok = contractVars[ident.Name]
		return ok
	case *ast.Ident:
		_, ok := contractVars[node.Name]
		return ok
	default:
		return false
	}
}

func checkModuleSourceIsolation(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	modulesRoot := filepath.Join(root, "modules")
	if !isDir(modulesRoot) {
		return issues
	}
	allowlist := readModuleIsolationAllowlist(root)
	entries, err := os.ReadDir(modulesRoot)
	if err != nil {
		return append(issues, DoctorIssue{
			Code:    "DX020",
			Message: "failed to read modules directory for isolation check",
			Fix:     err.Error(),
		})
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		moduleDir := filepath.Join(modulesRoot, entry.Name())
		if !hasFile(filepath.Join(moduleDir, "go.mod")) {
			continue
		}
		_ = filepath.WalkDir(moduleDir, func(path string, d os.DirEntry, err error) error {
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
			if doctorAllowsModuleRootImport(rel, string(b)) {
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
	}
	return issues
}

func readModuleIsolationAllowlist(root string) map[string]struct{} {
	path := filepath.Join(root, "tools", "scripts", "test", "module-isolation-allowlist.txt")
	content, err := os.ReadFile(path)
	if err != nil {
		return map[string]struct{}{}
	}
	allowlist := make(map[string]struct{})
	for _, line := range strings.Split(string(content), "\n") {
		entry := strings.TrimSpace(line)
		if entry == "" || strings.HasPrefix(entry, "#") {
			continue
		}
		allowlist[filepath.ToSlash(entry)] = struct{}{}
	}
	return allowlist
}

func doctorAllowsModuleRootImport(rel string, content string) bool {
	if rel == "modules/storage/module.go" && strings.Contains(content, "\"github.com/leomorpho/goship/framework/core\"") {
		return true
	}
	return false
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

func checkGoImportPrefixForbidden(dir string, forbiddenPrefix string, code string, message string, fix string) []DoctorIssue {
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

		fset := token.NewFileSet()
		file, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if parseErr != nil {
			return nil
		}
		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, "\"")
			if !strings.HasPrefix(importPath, forbiddenPrefix) {
				continue
			}
			issues = append(issues, DoctorIssue{
				Code:    code,
				File:    filepath.ToSlash(path),
				Message: message,
				Fix:     fix,
			})
			break
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

func checkTextForbiddenInDirNonTest(dir string, token string, code string, message string, fix string) []DoctorIssue {
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
