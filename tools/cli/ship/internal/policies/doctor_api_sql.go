package policies

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func doctorCheckAPIRoutes(root string) []DoctorIssue {
	path := filepath.Join(root, "app", "router.go")
	if !hasFile(path) {
		return nil
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return []DoctorIssue{{
			Code:     "DX026",
			Message:  fmt.Sprintf("failed to inspect API routes in app/router.go: %v", err),
			File:     filepath.ToSlash(mustRel(root, path)),
			Severity: "warning",
		}}
	}

	apiGroups := map[string]struct{}{}
	apiHandlers := map[string]struct{}{}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		ast.Inspect(fn.Body, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.AssignStmt:
				for i, rhs := range node.Rhs {
					call, ok := rhs.(*ast.CallExpr)
					if !ok || len(call.Args) == 0 {
						continue
					}
					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok || sel.Sel == nil || sel.Sel.Name != "Group" {
						continue
					}
					if !doctorExprContainsAPIPath(call.Args[0]) || i >= len(node.Lhs) {
						continue
					}
					if ident, ok := node.Lhs[i].(*ast.Ident); ok {
						apiGroups[ident.Name] = struct{}{}
					}
				}
			case *ast.CallExpr:
				sel, ok := node.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel == nil {
					return true
				}
				method := sel.Sel.Name
				if method != "GET" && method != "POST" && method != "PUT" && method != "DELETE" && method != "PATCH" {
					return true
				}
				if len(node.Args) < 2 {
					return true
				}
				if receiver, ok := sel.X.(*ast.Ident); ok {
					if _, isAPIGroup := apiGroups[receiver.Name]; !isAPIGroup && !doctorExprContainsAPIPath(node.Args[0]) {
						return true
					}
				} else if !doctorExprContainsAPIPath(node.Args[0]) {
					return true
				}
				if handlerName := doctorHandlerName(node.Args[1]); handlerName != "" {
					apiHandlers[handlerName] = struct{}{}
				}
			}
			return true
		})
	}

	if len(apiHandlers) == 0 {
		return nil
	}

	controllerBodies := doctorControllerBodies(root)
	issues := make([]DoctorIssue, 0)
	for handlerName := range apiHandlers {
		for _, body := range controllerBodies[handlerName] {
			if doctorHandlerUsesHTMLRendering(body.body) && !doctorHandlerUsesAPIHelpers(body.body) {
				issues = append(issues, DoctorIssue{
					Code:     "DX026",
					Message:  fmt.Sprintf("API route handler %s appears to render HTML directly", handlerName),
					Fix:      "use framework/api helpers (api.OK/api.Fail) or gate HTML rendering behind api.IsAPIRequest",
					File:     body.file,
					Severity: "warning",
				})
				break
			}
		}
	}
	return issues
}

func doctorControllerBodies(root string) map[string][]doctorHandlerBody {
	controllersDir := filepath.Join(root, "app", "web", "controllers")
	entries, err := os.ReadDir(controllersDir)
	if err != nil {
		return nil
	}

	result := map[string][]doctorHandlerBody{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(controllersDir, entry.Name())
		src, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, src, 0)
		if err != nil {
			continue
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			start := fset.Position(fn.Body.Pos()).Offset
			end := fset.Position(fn.Body.End()).Offset
			if start < 0 || end <= start || end > len(src) {
				continue
			}
			result[fn.Name.Name] = append(result[fn.Name.Name], doctorHandlerBody{
				file: filepath.ToSlash(mustRel(root, path)),
				body: string(src[start:end]),
			})
		}
	}
	return result
}

func checkSoftDeleteQueryFilters(root string) []DoctorIssue {
	tables := discoverSoftDeleteTables(root)
	if len(tables) == 0 {
		return nil
	}

	queriesDir := filepath.Join(root, "db", "queries")
	if !isDir(queriesDir) {
		return nil
	}

	issues := make([]DoctorIssue, 0)
	_ = filepath.WalkDir(queriesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".sql") {
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}

		statements := strings.Split(string(content), ";")
		for _, raw := range statements {
			stmt := strings.TrimSpace(stripSQLLineComments(raw))
			if stmt == "" {
				continue
			}

			normalized := normalizeSQLForDoctor(stmt)
			if !strings.HasPrefix(strings.TrimSpace(normalized), "select ") {
				continue
			}
			if strings.Contains(normalized, " deleted_at is null ") || strings.Contains(normalized, " deleted_at is not null ") {
				continue
			}

			for table := range tables {
				if !doctorStatementReferencesTable(normalized, table) {
					continue
				}
				issues = append(issues, DoctorIssue{
					Code:     "DX028",
					Message:  fmt.Sprintf("query references soft-delete table %q without deleted_at filter", table),
					Fix:      "add `deleted_at IS NULL` for active rows (or `deleted_at IS NOT NULL` for trash queries)",
					File:     filepath.ToSlash(mustRel(root, path)),
					Severity: "warning",
				})
				break
			}
		}

		return nil
	})

	return issues
}

func discoverSoftDeleteTables(root string) map[string]struct{} {
	migrationsDir := filepath.Join(root, "db", "migrate", "migrations")
	if !isDir(migrationsDir) {
		return nil
	}

	tables := make(map[string]struct{})
	createTableRE := regexp.MustCompile(`(?i)create\s+table\s+(?:if\s+not\s+exists\s+)?["` + "`" + `]?([a-z0-9_]+)["` + "`" + `]?`)
	alterTableRE := regexp.MustCompile(`(?i)alter\s+table\s+["` + "`" + `]?([a-z0-9_]+)["` + "`" + `]?\s+add\s+column\s+["` + "`" + `]?deleted_at["` + "`" + `]?`)

	_ = filepath.WalkDir(migrationsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".sql") {
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}

		for _, raw := range strings.Split(string(content), ";") {
			stmt := strings.TrimSpace(stripSQLLineComments(raw))
			if stmt == "" {
				continue
			}
			lower := strings.ToLower(stmt)
			if !strings.Contains(lower, "deleted_at") {
				continue
			}

			if matches := alterTableRE.FindStringSubmatch(lower); len(matches) == 2 {
				tables[strings.TrimSpace(matches[1])] = struct{}{}
				continue
			}
			if !strings.HasPrefix(strings.TrimSpace(lower), "create table") {
				continue
			}
			if matches := createTableRE.FindStringSubmatch(lower); len(matches) == 2 {
				tables[strings.TrimSpace(matches[1])] = struct{}{}
			}
		}

		return nil
	})

	return tables
}

func normalizeSQLForDoctor(sql string) string {
	replacer := strings.NewReplacer("\n", " ", "\r", " ", "\t", " ", "\"", "", "`", "")
	normalized := replacer.Replace(strings.ToLower(sql))
	return " " + strings.Join(strings.Fields(normalized), " ") + " "
}

func stripSQLLineComments(sql string) string {
	lines := strings.Split(sql, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

func doctorStatementReferencesTable(normalizedStmt string, table string) bool {
	t := strings.TrimSpace(strings.ToLower(table))
	if t == "" {
		return false
	}

	from := " from " + t + " "
	join := " join " + t + " "
	return strings.Contains(normalizedStmt, from) || strings.Contains(normalizedStmt, join)
}

func doctorExprContainsAPIPath(expr ast.Expr) bool {
	return strings.Contains(doctorExprLiteralString(expr), "/api/")
}

func doctorExprLiteralString(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.BasicLit:
		return strings.Trim(v.Value, "`\"")
	default:
		return ""
	}
}

func doctorHandlerName(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.SelectorExpr:
		return v.Sel.Name
	case *ast.Ident:
		return v.Name
	default:
		return ""
	}
}

func doctorHandlerUsesHTMLRendering(body string) bool {
	return strings.Contains(body, "RenderPage(") || strings.Contains(body, ".HTML(") || strings.Contains(body, "ctx.HTML(")
}

func doctorHandlerUsesAPIHelpers(body string) bool {
	return strings.Contains(body, "api.OK(") || strings.Contains(body, "api.Fail(") || strings.Contains(body, "IsAPIRequest(")
}
