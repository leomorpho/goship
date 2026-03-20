package commands

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type i18nInstrumentOptions struct {
	apply bool
	paths []string
	limit int
}

type i18nInstrumentOutput struct {
	Version  string                 `json:"version"`
	Apply    bool                   `json:"apply"`
	Applied  int                    `json:"applied"`
	Rewrites []i18nInstrumentRewrite `json:"rewrites"`
	Skipped  []i18nInstrumentSkip    `json:"skipped"`
}

type i18nInstrumentRewrite struct {
	ID           string `json:"id"`
	File         string `json:"file"`
	Line         int    `json:"line"`
	Column       int    `json:"column"`
	Before       string `json:"before"`
	After        string `json:"after"`
	Message      string `json:"message"`
	SuggestedKey string `json:"suggested_key"`
	Confidence   string `json:"confidence"`
}

type i18nInstrumentSkip struct {
	ID         string `json:"id"`
	File       string `json:"file"`
	Line       int    `json:"line"`
	Column     int    `json:"column"`
	Message    string `json:"message"`
	Reason     string `json:"reason"`
	Confidence string `json:"confidence"`
}

type i18nInstrumentCandidate struct {
	rewrite      i18nInstrumentRewrite
	file         string
	startOffset  int
	endOffset    int
	literalValue string
}

func runI18nInstrument(args []string, d I18nDeps, root string) int {
	opts, showHelp, err := parseI18nInstrumentArgs(args)
	if showHelp {
		printI18nInstrumentUsage(d.Out)
		return 0
	}
	if err != nil {
		fmt.Fprintf(d.Err, "%v\n", err)
		printI18nInstrumentUsage(d.Err)
		return 1
	}

	out, selected, err := buildI18nInstrumentPlan(root, opts.paths, opts.limit)
	if err != nil {
		fmt.Fprintf(d.Err, "i18n:instrument failed: %v\n", err)
		return 1
	}
	out.Apply = opts.apply
	out.Applied = 0
	if opts.apply {
		appliedCount, applyErr := applyI18nInstrument(root, selected)
		if applyErr != nil {
			fmt.Fprintf(d.Err, "i18n:instrument failed to apply rewrites: %v\n", applyErr)
			return 1
		}
		out.Applied = appliedCount
	}

	encoder := json.NewEncoder(d.Out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(out); err != nil {
		fmt.Fprintf(d.Err, "i18n:instrument failed to encode JSON: %v\n", err)
		return 1
	}
	return 0
}

func parseI18nInstrumentArgs(args []string) (i18nInstrumentOptions, bool, error) {
	opts := i18nInstrumentOptions{
		apply: false,
		limit: 0,
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			return opts, true, nil
		case "--apply":
			opts.apply = true
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
			value := strings.TrimSpace(args[i+1])
			limit, err := strconv.Atoi(value)
			if err != nil || limit < 0 {
				return opts, false, fmt.Errorf("invalid --limit value: %q", args[i+1])
			}
			opts.limit = limit
			i++
		default:
			return opts, false, fmt.Errorf("unknown i18n:instrument argument: %s", args[i])
		}
	}
	return opts, false, nil
}

func printI18nInstrumentUsage(w anyWriter) {
	fmt.Fprintln(w, "usage: ship i18n:instrument [--apply] [--paths <path1,path2,...>] [--limit <n>]")
}

func buildI18nInstrumentPlan(root string, paths []string, limit int) (i18nInstrumentOutput, []i18nInstrumentCandidate, error) {
	findings, err := CollectI18nScanFindings(root, paths)
	if err != nil {
		return i18nInstrumentOutput{}, nil, err
	}
	if limit > 0 && limit < len(findings) {
		findings = findings[:limit]
	}

	candidates, err := collectGoInstrumentCandidates(root, paths)
	if err != nil {
		return i18nInstrumentOutput{}, nil, err
	}

	output := i18nInstrumentOutput{
		Version:  "v1",
		Rewrites: make([]i18nInstrumentRewrite, 0),
		Skipped:  make([]i18nInstrumentSkip, 0),
	}
	selected := make([]i18nInstrumentCandidate, 0)

	for _, finding := range findings {
		locKey := instrumentLocationKey(finding.File, finding.Line, finding.Column)
		switch {
		case finding.Confidence != "high":
			output.Skipped = append(output.Skipped, i18nInstrumentSkip{
				ID:         finding.ID,
				File:       finding.File,
				Line:       finding.Line,
				Column:     finding.Column,
				Message:    finding.Message,
				Reason:     "low_confidence",
				Confidence: finding.Confidence,
			})
		case !strings.HasSuffix(strings.ToLower(finding.File), ".go"):
			output.Skipped = append(output.Skipped, i18nInstrumentSkip{
				ID:         finding.ID,
				File:       finding.File,
				Line:       finding.Line,
				Column:     finding.Column,
				Message:    finding.Message,
				Reason:     "unsupported_source_type",
				Confidence: finding.Confidence,
			})
		default:
			candidate, ok := candidates[locKey]
			if !ok {
				output.Skipped = append(output.Skipped, i18nInstrumentSkip{
					ID:         finding.ID,
					File:       finding.File,
					Line:       finding.Line,
					Column:     finding.Column,
					Message:    finding.Message,
					Reason:     "unsupported_go_context",
					Confidence: finding.Confidence,
				})
				continue
			}
			selected = append(selected, candidate)
			output.Rewrites = append(output.Rewrites, candidate.rewrite)
		}
	}

	sort.Slice(output.Rewrites, func(i, j int) bool {
		left := output.Rewrites[i]
		right := output.Rewrites[j]
		if left.File != right.File {
			return left.File < right.File
		}
		if left.Line != right.Line {
			return left.Line < right.Line
		}
		if left.Column != right.Column {
			return left.Column < right.Column
		}
		return left.ID < right.ID
	})
	sort.Slice(output.Skipped, func(i, j int) bool {
		left := output.Skipped[i]
		right := output.Skipped[j]
		if left.File != right.File {
			return left.File < right.File
		}
		if left.Line != right.Line {
			return left.Line < right.Line
		}
		if left.Column != right.Column {
			return left.Column < right.Column
		}
		if left.Reason != right.Reason {
			return left.Reason < right.Reason
		}
		return left.ID < right.ID
	})
	sort.Slice(selected, func(i, j int) bool {
		left := selected[i]
		right := selected[j]
		if left.file != right.file {
			return left.file < right.file
		}
		return left.startOffset < right.startOffset
	})

	return output, selected, nil
}

func instrumentLocationKey(file string, line, column int) string {
	return fmt.Sprintf("%s:%d:%d", file, line, column)
}

func collectGoInstrumentCandidates(root string, paths []string) (map[string]i18nInstrumentCandidate, error) {
	targets := []string{root}
	if len(paths) > 0 {
		targets = targets[:0]
		for _, raw := range paths {
			clean := filepath.Clean(raw)
			if clean == "." {
				targets = append(targets, root)
				continue
			}
			targets = append(targets, filepath.Join(root, clean))
		}
	}

	visited := map[string]struct{}{}
	candidates := map[string]i18nInstrumentCandidate{}

	for _, target := range targets {
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
			if err := collectGoInstrumentCandidatesFromFile(root, target, candidates); err != nil {
				return nil, err
			}
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
			return collectGoInstrumentCandidatesFromFile(root, path, candidates)
		})
		if err != nil {
			return nil, err
		}
	}
	return candidates, nil
}

func collectGoInstrumentCandidatesFromFile(root, path string, out map[string]i18nInstrumentCandidate) error {
	if strings.ToLower(filepath.Ext(path)) != ".go" {
		return nil
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return err
	}
	rel = filepath.ToSlash(rel)
	if strings.HasSuffix(rel, "_test.go") {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, rel, data, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse go file %s: %w", rel, err)
	}

	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel == nil || selector.Sel.Name != "String" {
			return true
		}
		receiver, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}
		receiverName := strings.TrimSpace(receiver.Name)
		if receiverName == "" {
			return true
		}
		if len(call.Args) < 2 {
			return true
		}
		lit, ok := call.Args[1].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}

		literal, unquoteErr := strconv.Unquote(lit.Value)
		if unquoteErr != nil {
			literal = strings.Trim(lit.Value, "\"`")
		}
		literal = strings.TrimSpace(literal)
		if !isUserFacingLiteral(literal) {
			return true
		}

		start := fset.Position(lit.Pos())
		end := fset.Position(lit.End())
		if start.Offset < 0 || end.Offset <= start.Offset || end.Offset > len(data) {
			return true
		}
		issue := newI18nScanIssue(rel, start.Line, start.Column, literal)
		key := instrumentLocationKey(rel, start.Line, start.Column)
		out[key] = i18nInstrumentCandidate{
			rewrite: i18nInstrumentRewrite{
				ID:           issue.ID,
				File:         rel,
				Line:         start.Line,
				Column:       start.Column,
				Before:       lit.Value,
				After:        fmt.Sprintf(`%s.Container.I18n.T(%s.Request().Context(), %q)`, receiverName, receiverName, issue.SuggestedKey),
				Message:      issue.Message,
				SuggestedKey: issue.SuggestedKey,
				Confidence:   issue.Confidence,
			},
			file:         rel,
			startOffset:  start.Offset,
			endOffset:    end.Offset,
			literalValue: literal,
		}
		return true
	})
	return nil
}

func applyI18nInstrument(root string, candidates []i18nInstrumentCandidate) (int, error) {
	if len(candidates) == 0 {
		return 0, nil
	}

	grouped := map[string][]i18nInstrumentCandidate{}
	for _, candidate := range candidates {
		grouped[candidate.file] = append(grouped[candidate.file], candidate)
	}

	keysToAdd := map[string]string{}
	applied := 0

	files := make([]string, 0, len(grouped))
	for file := range grouped {
		files = append(files, file)
	}
	sort.Strings(files)

	for _, rel := range files {
		candidatesForFile := grouped[rel]
		sort.Slice(candidatesForFile, func(i, j int) bool {
			return candidatesForFile[i].startOffset > candidatesForFile[j].startOffset
		})

		abs := filepath.Join(root, rel)
		raw, err := os.ReadFile(abs)
		if err != nil {
			return applied, fmt.Errorf("read %s: %w", rel, err)
		}
		updated := raw

		for _, candidate := range candidatesForFile {
			if candidate.endOffset > len(updated) || candidate.startOffset < 0 || candidate.endOffset <= candidate.startOffset {
				continue
			}
			updated = append(updated[:candidate.startOffset], append([]byte(candidate.rewrite.After), updated[candidate.endOffset:]...)...)
			keysToAdd[candidate.rewrite.SuggestedKey] = candidate.literalValue
			applied++
		}

		fset := token.NewFileSet()
		if _, err := parser.ParseFile(fset, rel, updated, parser.ParseComments); err != nil {
			return applied, fmt.Errorf("instrumented file %s is not valid go syntax: %w", rel, err)
		}
		if err := os.WriteFile(abs, updated, 0o644); err != nil {
			return applied, fmt.Errorf("write %s: %w", rel, err)
		}
	}

	baselinePath := resolveEnglishLocalePathForWrite(filepath.Join(root, "locales"))
	if err := ensureInstrumentLocaleKeys(baselinePath, keysToAdd); err != nil {
		return applied, err
	}
	return applied, nil
}

func ensureInstrumentLocaleKeys(path string, keys map[string]string) error {
	if len(keys) == 0 {
		return nil
	}
	existing := map[string]string{}
	loaded, err := loadLocaleFlatFromFile(path)
	if err == nil {
		existing = loaded
	}

	merged := make(map[string]string, len(existing)+len(keys))
	for key, value := range existing {
		merged[key] = value
	}
	missing := make([]string, 0, len(keys))
	for key := range keys {
		if _, ok := existing[key]; ok {
			continue
		}
		missing = append(missing, key)
		merged[key] = keys[key]
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)

	switch detectLocaleFormat(path) {
	case localeFormatTOML:
		if err := os.WriteFile(path, []byte(renderCanonicalTOML(merged)), 0o644); err != nil {
			return fmt.Errorf("write baseline locale %s: %w", path, err)
		}
		return nil
	case localeFormatYAML:
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read baseline locale %s: %w", path, err)
		}
		builder := strings.Builder{}
		builder.Write(raw)
		if len(raw) > 0 && raw[len(raw)-1] != '\n' {
			builder.WriteByte('\n')
		}
		for _, key := range missing {
			builder.WriteString(fmt.Sprintf("%s: %s\n", key, strconv.Quote(keys[key])))
		}
		if err := os.WriteFile(path, []byte(builder.String()), 0o644); err != nil {
			return fmt.Errorf("write baseline locale %s: %w", path, err)
		}
		return nil
	default:
		if err := os.WriteFile(path, []byte(renderCanonicalTOML(merged)), 0o644); err != nil {
			return fmt.Errorf("write baseline locale %s: %w", path, err)
		}
		return nil
	}
}
