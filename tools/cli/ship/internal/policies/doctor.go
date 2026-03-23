package policies

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
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
	LookPath     func(file string) (string, error)
	RunCmd       func(dir string, name string, args ...string) (int, string, error)
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
	issues = append(issues, runNilawayChecks(root, d)...)
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

type doctorHandlerBody struct {
	file string
	body string
}

func RunDoctorChecks(root string) []DoctorIssue {
	issues := make([]DoctorIssue, 0)
	isFrameworkRepo := looksLikeCanonicalFrameworkRepo(root)

	if !isFrameworkRepo {
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
	}

	issues = append(issues, doctorCheckAPIRoutes(root)...)

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
	issues = append(issues, checkRequiredConfigEnv(root)...)
	if !isFrameworkRepo {
		issues = append(issues, checkPackageNaming(root, filepath.Join("app", "web", "ui"), "ui")...)
		issues = append(issues, checkPackageNaming(root, filepath.Join("app", "web", "viewmodels"), "viewmodels")...)
		issues = append(issues, checkTopLevelDirs(root)...)
	} else {
		issues = append(issues, CheckCanonicalRepoTopLevelPaths(root)...)
		issues = append(issues, checkFrameworkCIVerifyGate(root)...)
	}
	issues = append(issues, checkFileSizes(root)...)
	issues = append(issues, checkCLIDocsCoverage(root)...)
	issues = append(issues, checkExtensionZoneManifest(root)...)
	issues = append(issues, checkCanonicalDocsHardReset(root)...)
	issues = append(issues, checkGoWorkModules(root)...)
	issues = append(issues, checkDockerIgnoreCoverage(root)...)
	issues = append(issues, checkDockerLocalReplaceOrder(root)...)
	issues = append(issues, checkAgentPolicyArtifacts(root)...)
	issues = append(issues, checkModulesManifestFormat(root)...)
	issues = append(issues, checkEnabledModuleDBArtifacts(root)...)
	issues = append(issues, checkForbiddenCrossBoundaryImports(root)...)
	issues = append(issues, checkContractUsage(root)...)
	issues = append(issues, checkCanonicalFilePlacement(root)...)
	issues = append(issues, checkSoftDeleteQueryFilters(root)...)
	issues = append(issues, checkI18nLiteralEnforcement(root)...)

	return issues
}

func checkI18nLiteralEnforcement(root string) []DoctorIssue {
	mode := resolveI18nStrictMode(root)
	switch mode {
	case "off":
		return nil
	case "warn", "error":
	default:
		return []DoctorIssue{{
			Code:     "DX029",
			Message:  fmt.Sprintf("invalid PAGODA_I18N_STRICT_MODE value: %q", mode),
			Fix:      "use one of: off, warn, error",
			Severity: "error",
		}}
	}

	findings, err := collectI18nStrictFindings(root, []string{
		filepath.Join("app", "web", "controllers"),
		filepath.Join("app", "views"),
		filepath.Join("frontend", "islands"),
	})
	if err != nil {
		return []DoctorIssue{{
			Code:     "DX029",
			Message:  fmt.Sprintf("i18n strict-mode scan failed: %v", err),
			Fix:      "fix scanner/runtime errors before enabling strict mode",
			Severity: "error",
		}}
	}
	allowlist := loadI18nStrictAllowlist(filepath.Join(root, ".i18n-allowlist"))
	severity := "warning"
	if mode == "error" {
		severity = "error"
	}

	issues := make([]DoctorIssue, 0, len(findings))
	for _, finding := range findings {
		locationKey := fmt.Sprintf("%s:%d", finding.File, finding.Line)
		if _, ok := allowlist[finding.ID]; ok {
			continue
		}
		if _, ok := allowlist[finding.StableID]; ok {
			continue
		}
		if _, ok := allowlist[locationKey]; ok {
			continue
		}
		issues = append(issues, DoctorIssue{
			Code:     "DX029",
			Message:  fmt.Sprintf("i18n literal %s:%d:%d (%s, stable %s)", finding.File, finding.Line, finding.Column, finding.ID, finding.StableID),
			Fix:      "replace with i18n key usage or add the stable issue ID (preferred), scan issue ID, or legacy path:line to .i18n-allowlist",
			File:     finding.File,
			Severity: severity,
		})
	}

	completenessIssues, err := collectI18nCompletenessIssues(root)
	if err != nil {
		issues = append(issues, DoctorIssue{
			Code:     "DX029",
			Message:  fmt.Sprintf("i18n completeness check failed: %v", err),
			Fix:      "fix locale catalog parse issues before enabling strict mode",
			Severity: "error",
		})
		return issues
	}
	for _, issue := range completenessIssues {
		if _, ok := allowlist[issue.ID]; ok {
			continue
		}
		issues = append(issues, DoctorIssue{
			Code:     "DX029",
			Message:  fmt.Sprintf("i18n completeness %s %s (%s)", issue.Locale, issue.Kind, issue.ID),
			Fix:      "add required plural/select fallback keys or allowlist the issue ID in .i18n-allowlist",
			File:     issue.File,
			Severity: severity,
		})
	}
	return issues
}

func resolveI18nStrictMode(root string) string {
	if fromEnv := strings.ToLower(strings.TrimSpace(os.Getenv("PAGODA_I18N_STRICT_MODE"))); fromEnv != "" {
		return fromEnv
	}

	dotEnvPath := filepath.Join(root, ".env")
	content, err := os.ReadFile(dotEnvPath)
	if err != nil {
		return "off"
	}
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if strings.TrimSpace(key) != "PAGODA_I18N_STRICT_MODE" {
			continue
		}
		clean := strings.ToLower(strings.TrimSpace(value))
		if clean == "" {
			return "off"
		}
		return clean
	}
	return "off"
}

func loadI18nStrictAllowlist(path string) map[string]struct{} {
	values := map[string]struct{}{}
	content, err := os.ReadFile(path)
	if err != nil {
		return values
	}
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		values[line] = struct{}{}
	}
	return values
}

type i18nStrictFinding struct {
	ID       string
	StableID string
	File     string
	Line     int
	Column   int
}

func collectI18nStrictFindings(root string, relPaths []string) ([]i18nStrictFinding, error) {
	findings := make([]i18nStrictFinding, 0)
	for _, rel := range relPaths {
		target := filepath.Join(root, rel)
		info, err := os.Stat(target)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		if !info.IsDir() {
			scanned, err := scanI18nStrictFile(root, target)
			if err != nil {
				return nil, err
			}
			findings = append(findings, scanned...)
			continue
		}

		err = filepath.WalkDir(target, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				if d.Name() == "gen" || d.Name() == "node_modules" {
					return filepath.SkipDir
				}
				return nil
			}
			scanned, err := scanI18nStrictFile(root, path)
			if err != nil {
				return err
			}
			findings = append(findings, scanned...)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].File != findings[j].File {
			return findings[i].File < findings[j].File
		}
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		if findings[i].Column != findings[j].Column {
			return findings[i].Column < findings[j].Column
		}
		return findings[i].ID < findings[j].ID
	})

	return findings, nil
}

var (
	doctorI18nStringLiteralPattern = regexp.MustCompile(`"([^"\\]*(\\.[^"\\]*)*)"|'([^'\\]*(\\.[^'\\]*)*)'|` + "`" + `([^` + "`" + `]*)` + "`")
	doctorI18nTemplTextPattern     = regexp.MustCompile(`>([^<>{]+)<`)
)

func scanI18nStrictFile(root, path string) ([]i18nStrictFinding, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go", ".templ", ".js", ".ts", ".svelte", ".vue":
	default:
		return nil, nil
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return nil, err
	}
	rel = filepath.ToSlash(rel)
	if ext == ".go" && strings.HasSuffix(rel, "_test.go") {
		return nil, nil
	}
	if (ext == ".js" || ext == ".ts" || ext == ".svelte" || ext == ".vue") && !strings.HasPrefix(strings.ToLower(rel), "frontend/islands/") {
		return nil, nil
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if ext == ".go" {
		return scanI18nStrictGo(rel, raw)
	}
	return scanI18nStrictText(rel, ext, string(raw)), nil
}

func scanI18nStrictGo(rel string, raw []byte) ([]i18nStrictFinding, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, rel, raw, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	findings := make([]i18nStrictFinding, 0)
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
		value, unquoteErr := strconv.Unquote(lit.Value)
		if unquoteErr != nil {
			value = strings.Trim(lit.Value, "\"`")
		}
		value = strings.TrimSpace(value)
		if !doctorLooksUserFacingLiteral(value) || doctorIgnoreGoI18nLiteral(stack, value) {
			return true
		}
		pos := fset.Position(lit.Pos())
		findings = append(findings, i18nStrictFinding{
			ID:       doctorI18nIssueID(rel, pos.Line, pos.Column, value),
			StableID: doctorI18nStableIssueID(rel, value),
			File:     rel,
			Line:     pos.Line,
			Column:   pos.Column,
		})
		return true
	})
	return findings, nil
}

func scanI18nStrictText(rel, ext, raw string) []i18nStrictFinding {
	lines := strings.Split(raw, "\n")
	findings := make([]i18nStrictFinding, 0)
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.Contains(line, "I18n.T(") || strings.Contains(line, "i18n.T(") ||
			strings.Contains(line, "I18n.TC(") || strings.Contains(line, "i18n.TC(") ||
			strings.Contains(line, "I18n.TS(") || strings.Contains(line, "i18n.TS(") {
			continue
		}
		if (ext == ".js" || ext == ".ts" || ext == ".svelte" || ext == ".vue") && strings.Contains(strings.ToLower(line), "i18n.t(") {
			continue
		}

		if ext == ".templ" {
			matches := doctorI18nTemplTextPattern.FindAllStringSubmatchIndex(line, -1)
			for _, m := range matches {
				if len(m) < 4 || m[2] < 0 || m[3] <= m[2] {
					continue
				}
				value := strings.TrimSpace(line[m[2]:m[3]])
				if !doctorLooksUserFacingLiteral(value) {
					continue
				}
				findings = append(findings, i18nStrictFinding{
					ID:       doctorI18nIssueID(rel, i+1, m[2]+1, value),
					StableID: doctorI18nStableIssueID(rel, value),
					File:     rel,
					Line:     i + 1,
					Column:   m[2] + 1,
				})
			}
			continue
		}

		matches := doctorI18nStringLiteralPattern.FindAllStringSubmatchIndex(line, -1)
		for _, m := range matches {
			if len(m) < 2 || m[0] < 0 || m[1] <= m[0] {
				continue
			}
			value := strings.TrimSpace(strings.Trim(line[m[0]:m[1]], "\"'`"))
			if !doctorLooksUserFacingLiteral(value) {
				continue
			}
			findings = append(findings, i18nStrictFinding{
				ID:       doctorI18nIssueID(rel, i+1, m[0]+1, value),
				StableID: doctorI18nStableIssueID(rel, value),
				File:     rel,
				Line:     i + 1,
				Column:   m[0] + 1,
			})
		}
	}
	return findings
}

func doctorLooksUserFacingLiteral(value string) bool {
	if len(value) < 3 {
		return false
	}
	hasLetter := false
	for _, r := range value {
		if unicode.IsLetter(r) {
			hasLetter = true
			break
		}
	}
	if !hasLetter {
		return false
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return false
	}
	upper := strings.ToUpper(strings.TrimSpace(value))
	return !strings.HasPrefix(upper, "SELECT ") &&
		!strings.HasPrefix(upper, "INSERT ") &&
		!strings.HasPrefix(upper, "UPDATE ") &&
		!strings.HasPrefix(upper, "DELETE ") &&
		!strings.HasPrefix(upper, "CREATE ") &&
		!strings.HasPrefix(upper, "ALTER ") &&
		!strings.HasPrefix(upper, "DROP ")
}

func doctorIgnoreGoI18nLiteral(stack []ast.Node, value string) bool {
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(value)), "SELECT ") {
		return true
	}
	for i := len(stack) - 1; i >= 0; i-- {
		switch node := stack[i].(type) {
		case *ast.ImportSpec:
			return true
		case *ast.CallExpr:
			sel, ok := node.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel == nil {
				continue
			}
			switch sel.Sel.Name {
			case "Debug", "Info", "Warn", "Error", "Print", "Printf", "Println", "Fatal", "Fatalf", "Fatalln", "New", "Errorf":
				return true
			case "T", "TC", "TS":
				return true
			}
		}
	}
	return false
}

func doctorI18nIssueID(file string, line, column int, literal string) string {
	base := fmt.Sprintf("%s|%d|%d|%s", file, line, column, literal)
	sum := 0
	for _, r := range base {
		sum = ((sum << 5) - sum) + int(r)
	}
	if sum < 0 {
		sum = -sum
	}
	return fmt.Sprintf("I18N-%d", sum)
}

func doctorI18nStableIssueID(file, literal string) string {
	base := fmt.Sprintf("%s|%s", file, strings.TrimSpace(literal))
	sum := sha1.Sum([]byte(base))
	return "I18N-S-" + strings.ToUpper(hex.EncodeToString(sum[:]))[:12]
}
