package commands

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type i18nScanOptions struct {
	format string
	paths  []string
	limit  int
}

type i18nScanOutput struct {
	Version string          `json:"version"`
	Issues  []i18nScanIssue `json:"issues"`
}

type i18nScanIssue struct {
	ID           string `json:"id"`
	Kind         string `json:"kind"`
	Severity     string `json:"severity"`
	File         string `json:"file"`
	Line         int    `json:"line"`
	Column       int    `json:"column"`
	Message      string `json:"message"`
	SuggestedKey string `json:"suggested_key"`
	Confidence   string `json:"confidence"`
}

// I18nScanFinding is the exported i18n scanner diagnostic record.
type I18nScanFinding struct {
	ID           string
	Kind         string
	Severity     string
	File         string
	Line         int
	Column       int
	Message      string
	SuggestedKey string
	Confidence   string
}

var (
	scanStringLiteralPattern = regexp.MustCompile(`"([^"\\]*(\\.[^"\\]*)*)"|'([^'\\]*(\\.[^'\\]*)*)'|` + "`" + `([^` + "`" + `]*)` + "`")
	scanTemplTextPattern     = regexp.MustCompile(`>([^<>{]+)<`)
)

func runI18nScan(args []string, d I18nDeps, root string) int {
	opts, showHelp, err := parseI18nScanArgs(args)
	if showHelp {
		printI18nScanUsage(d.Out)
		return 0
	}
	if err != nil {
		fmt.Fprintf(d.Err, "%v\n", err)
		printI18nScanUsage(d.Err)
		return 1
	}

	issues, err := collectI18nScanIssues(root, opts.paths)
	if err != nil {
		fmt.Fprintf(d.Err, "i18n:scan failed: %v\n", err)
		return 1
	}
	sortI18nScanIssues(issues)
	if opts.limit > 0 && opts.limit < len(issues) {
		issues = issues[:opts.limit]
	}

	out := i18nScanOutput{
		Version: "v1",
		Issues:  issues,
	}
	encoder := json.NewEncoder(d.Out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(out); err != nil {
		fmt.Fprintf(d.Err, "i18n:scan failed to encode JSON: %v\n", err)
		return 1
	}
	return 0
}

// CollectI18nScanFindings returns deterministic scanner findings for the given project root.
func CollectI18nScanFindings(root string, paths []string) ([]I18nScanFinding, error) {
	issues, err := collectI18nScanIssues(root, paths)
	if err != nil {
		return nil, err
	}
	sortI18nScanIssues(issues)
	out := make([]I18nScanFinding, 0, len(issues))
	for _, issue := range issues {
		out = append(out, I18nScanFinding{
			ID:           issue.ID,
			Kind:         issue.Kind,
			Severity:     issue.Severity,
			File:         issue.File,
			Line:         issue.Line,
			Column:       issue.Column,
			Message:      issue.Message,
			SuggestedKey: issue.SuggestedKey,
			Confidence:   issue.Confidence,
		})
	}
	return out, nil
}

func parseI18nScanArgs(args []string) (i18nScanOptions, bool, error) {
	opts := i18nScanOptions{
		format: "json",
		limit:  0,
	}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			return opts, true, nil
		case "--format":
			if i+1 >= len(args) {
				return opts, false, fmt.Errorf("missing value for --format")
			}
			opts.format = strings.TrimSpace(args[i+1])
			i++
		case "--paths":
			if i+1 >= len(args) {
				return opts, false, fmt.Errorf("missing value for --paths")
			}
			opts.paths = parseCSVArgs(args[i+1])
			i++
		case "--limit":
			if i+1 >= len(args) {
				return opts, false, fmt.Errorf("missing value for --limit")
			}
			limit, err := strconv.Atoi(strings.TrimSpace(args[i+1]))
			if err != nil || limit < 0 {
				return opts, false, fmt.Errorf("invalid --limit value: %q", args[i+1])
			}
			opts.limit = limit
			i++
		default:
			return opts, false, fmt.Errorf("unknown i18n:scan argument: %s", args[i])
		}
	}

	if opts.format != "json" {
		return opts, false, fmt.Errorf("unsupported --format value %q (supported: json)", opts.format)
	}
	return opts, false, nil
}

func parseCSVArgs(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		clean := strings.TrimSpace(part)
		if clean != "" {
			result = append(result, clean)
		}
	}
	return result
}

func printI18nScanUsage(w anyWriter) {
	fmt.Fprintln(w, "usage: ship i18n:scan [--format json] [--paths <path1,path2,...>] [--limit <n>]")
}

type anyWriter interface {
	Write(p []byte) (n int, err error)
}

func collectI18nScanIssues(root string, paths []string) ([]i18nScanIssue, error) {
	scanTargets := []string{root}
	if len(paths) > 0 {
		scanTargets = scanTargets[:0]
		for _, raw := range paths {
			clean := filepath.Clean(raw)
			if clean == "." {
				scanTargets = append(scanTargets, root)
				continue
			}
			scanTargets = append(scanTargets, filepath.Join(root, clean))
		}
	}

	visited := map[string]struct{}{}
	all := make([]i18nScanIssue, 0)

	for _, target := range scanTargets {
		info, err := os.Stat(target)
		if err != nil {
			return nil, fmt.Errorf("invalid scan target %q: %w", target, err)
		}
		if !info.IsDir() {
			abs, err := filepath.Abs(target)
			if err != nil {
				return nil, err
			}
			if _, ok := visited[abs]; ok {
				continue
			}
			visited[abs] = struct{}{}
			fileIssues, err := scanI18nFile(root, target)
			if err != nil {
				return nil, err
			}
			all = append(all, fileIssues...)
			continue
		}

		err = filepath.WalkDir(target, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				switch d.Name() {
				case ".git", ".docket", "node_modules", "tmp", "vendor", "gen":
					return filepath.SkipDir
				default:
					return nil
				}
			}

			abs, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			if _, ok := visited[abs]; ok {
				return nil
			}
			visited[abs] = struct{}{}

			fileIssues, err := scanI18nFile(root, path)
			if err != nil {
				return err
			}
			all = append(all, fileIssues...)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return all, nil
}

func scanI18nFile(root, path string) ([]i18nScanIssue, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go", ".templ", ".js", ".ts", ".svelte", ".vue":
	default:
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	relativePath, err := filepath.Rel(root, path)
	if err != nil {
		return nil, err
	}
	relativePath = filepath.ToSlash(relativePath)

	if isJavaScriptLike(ext) && !isIslandsSourcePath(relativePath) {
		return nil, nil
	}

	if ext == ".go" {
		if strings.HasSuffix(relativePath, "_test.go") {
			return nil, nil
		}
		return scanGoFileAST(relativePath, data)
	}

	lines := strings.Split(string(data), "\n")
	issues := make([]i18nScanIssue, 0)

	for idx, rawLine := range lines {
		lineNo := idx + 1
		line := strings.TrimSpace(rawLine)
		if shouldSkipScanLine(ext, line) {
			continue
		}

		switch ext {
		case ".templ":
			templIssues := scanTemplTextLine(relativePath, rawLine, lineNo)
			issues = append(issues, templIssues...)
		default:
			literalIssues := scanStringLiteralLine(relativePath, rawLine, lineNo)
			issues = append(issues, literalIssues...)
		}
	}
	return issues, nil
}

func isJavaScriptLike(ext string) bool {
	switch ext {
	case ".js", ".ts", ".svelte", ".vue":
		return true
	default:
		return false
	}
}

func isIslandsSourcePath(relativePath string) bool {
	clean := strings.TrimPrefix(strings.ToLower(relativePath), "./")
	return strings.HasPrefix(clean, "frontend/islands/")
}

func scanGoFileAST(relativePath string, data []byte) ([]i18nScanIssue, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, relativePath, data, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse go file %s: %w", relativePath, err)
	}

	issues := make([]i18nScanIssue, 0)
	stack := make([]ast.Node, 0, 32)
	ast.Inspect(file, func(node ast.Node) bool {
		if node == nil {
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			return true
		}
		stack = append(stack, node)

		lit, ok := node.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		cleaned, unquoteErr := strconv.Unquote(lit.Value)
		if unquoteErr != nil {
			cleaned = strings.Trim(lit.Value, "\"`")
		}
		cleaned = strings.TrimSpace(cleaned)
		if !isUserFacingLiteral(cleaned) {
			return true
		}
		if shouldIgnoreGoLiteral(stack, cleaned) {
			return true
		}

		pos := fset.Position(lit.Pos())
		issues = append(issues, newI18nScanIssue(relativePath, pos.Line, pos.Column, cleaned))
		return true
	})
	return issues, nil
}

func shouldIgnoreGoLiteral(stack []ast.Node, literal string) bool {
	if isSQLLiteral(literal) {
		return true
	}

	for i := len(stack) - 1; i >= 0; i-- {
		switch node := stack[i].(type) {
		case *ast.ImportSpec:
			return true
		case *ast.CallExpr:
			if exprIsI18nCall(node.Fun) || exprIsLoggingCall(node.Fun) || exprIsErrorCall(node.Fun) {
				return true
			}
		}
	}
	return false
}

func exprIsI18nCall(expr ast.Expr) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok || selector.Sel == nil {
		return false
	}
	switch selector.Sel.Name {
	case "T", "TC", "TS":
	default:
		return false
	}
	ident, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}
	name := strings.ToLower(strings.TrimSpace(ident.Name))
	return name == "i18n"
}

func exprIsLoggingCall(expr ast.Expr) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok || selector.Sel == nil {
		return false
	}
	switch selector.Sel.Name {
	case "Debug", "Info", "Warn", "Error", "Print", "Printf", "Println", "Fatal", "Fatalf", "Fatalln":
		return true
	default:
		return false
	}
}

func exprIsErrorCall(expr ast.Expr) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok || selector.Sel == nil {
		return false
	}
	if ident, ok := selector.X.(*ast.Ident); ok {
		if ident.Name == "errors" && (selector.Sel.Name == "New" || selector.Sel.Name == "Join") {
			return true
		}
		if ident.Name == "fmt" && selector.Sel.Name == "Errorf" {
			return true
		}
	}
	return false
}

func isSQLLiteral(value string) bool {
	upper := strings.ToUpper(strings.TrimSpace(value))
	switch {
	case strings.HasPrefix(upper, "SELECT "):
		return true
	case strings.HasPrefix(upper, "INSERT "):
		return true
	case strings.HasPrefix(upper, "UPDATE "):
		return true
	case strings.HasPrefix(upper, "DELETE "):
		return true
	case strings.HasPrefix(upper, "CREATE "):
		return true
	case strings.HasPrefix(upper, "ALTER "):
		return true
	case strings.HasPrefix(upper, "DROP "):
		return true
	default:
		return false
	}
}

func shouldSkipScanLine(ext, line string) bool {
	if line == "" {
		return true
	}
	if strings.Contains(line, "I18n.T(") || strings.Contains(line, "i18n.T(") ||
		strings.Contains(line, "I18n.TC(") || strings.Contains(line, "i18n.TC(") ||
		strings.Contains(line, "I18n.TS(") || strings.Contains(line, "i18n.TS(") {
		return true
	}
	if isJavaScriptLike(ext) && strings.Contains(strings.ToLower(line), "i18n.t(") {
		return true
	}
	if ext == ".go" {
		if strings.HasPrefix(line, "package ") || strings.HasPrefix(line, "import ") || strings.HasPrefix(line, "//") {
			return true
		}
	}
	return false
}

func scanStringLiteralLine(file, rawLine string, lineNo int) []i18nScanIssue {
	matches := scanStringLiteralPattern.FindAllStringSubmatchIndex(rawLine, -1)
	out := make([]i18nScanIssue, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 || match[0] < 0 || match[1] <= match[0] {
			continue
		}
		literal := rawLine[match[0]:match[1]]
		cleaned := strings.TrimSpace(strings.Trim(literal, "\"'`"))
		if !isUserFacingLiteral(cleaned) {
			continue
		}
		column := match[0] + 1
		out = append(out, newI18nScanIssue(file, lineNo, column, cleaned))
	}
	return out
}

func scanTemplTextLine(file, rawLine string, lineNo int) []i18nScanIssue {
	matches := scanTemplTextPattern.FindAllStringSubmatchIndex(rawLine, -1)
	out := make([]i18nScanIssue, 0, len(matches))
	for _, match := range matches {
		if len(match) < 4 || match[2] < 0 || match[3] <= match[2] {
			continue
		}
		value := strings.TrimSpace(rawLine[match[2]:match[3]])
		if !isUserFacingLiteral(value) {
			continue
		}
		column := match[2] + 1
		out = append(out, newI18nScanIssue(file, lineNo, column, value))
	}
	return out
}

func isUserFacingLiteral(value string) bool {
	if len(value) < 3 {
		return false
	}
	if !containsLetter(value) {
		return false
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return false
	}
	if strings.Contains(value, "github.com/") {
		return false
	}
	return true
}

func containsLetter(value string) bool {
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return true
		}
	}
	return false
}

func newI18nScanIssue(file string, line, column int, literal string) i18nScanIssue {
	key := suggestI18nKey(literal)
	kind := "missing_i18n_key"
	confidence := "medium"
	if strings.Contains(literal, " ") {
		confidence = "high"
	}
	base := fmt.Sprintf("%s|%s|%d|%d|%s", kind, file, line, column, literal)
	hash := sha1.Sum([]byte(base))

	return i18nScanIssue{
		ID:           "I18N-" + strings.ToUpper(hex.EncodeToString(hash[:]))[:12],
		Kind:         kind,
		Severity:     "warning",
		File:         file,
		Line:         line,
		Column:       column,
		Message:      fmt.Sprintf("hardcoded user-facing string %q should use i18n key", literal),
		SuggestedKey: key,
		Confidence:   confidence,
	}
}

func suggestI18nKey(literal string) string {
	clean := strings.ToLower(strings.TrimSpace(literal))
	replacer := strings.NewReplacer(
		".", " ",
		",", " ",
		":", " ",
		";", " ",
		"!", " ",
		"?", " ",
		"(", " ",
		")", " ",
		"/", " ",
		"\\", " ",
		"-", " ",
		"_", " ",
		"'", "",
		`"`, "",
	)
	clean = replacer.Replace(clean)
	parts := strings.Fields(clean)
	if len(parts) == 0 {
		return "app.message"
	}
	if len(parts) > 6 {
		parts = parts[:6]
	}
	return "app." + strings.Join(parts, "_")
}

func sortI18nScanIssues(issues []i18nScanIssue) {
	sort.Slice(issues, func(i, j int) bool {
		left := issues[i]
		right := issues[j]
		if left.File != right.File {
			return left.File < right.File
		}
		if left.Line != right.Line {
			return left.Line < right.Line
		}
		if left.Column != right.Column {
			return left.Column < right.Column
		}
		if left.Kind != right.Kind {
			return left.Kind < right.Kind
		}
		return left.ID < right.ID
	})
}
