package commands

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	policies "github.com/leomorpho/goship/tools/cli/ship/internal/policies"
	startertemplate "github.com/leomorpho/goship/tools/cli/ship/internal/templates/starter"
)

const starterTemplateRoot = "testdata/scaffold"

const (
	newUIProviderFranken = "franken"
	newUIProviderDaisy   = "daisy"
	newUIProviderBare    = "bare"
)

type NewProjectOptions struct {
	Name        string
	Module      string
	DryRun      bool
	Force       bool
	AppPath     string
	UIProvider  string
	APIMode     bool
	I18nEnabled bool
	I18nSet     bool
}

type NewDeps struct {
	Out                        io.Writer
	Err                        io.Writer
	ParseAgentPolicyBytes      func(b []byte) (policies.AgentPolicy, error)
	RenderAgentPolicyArtifacts func(policy policies.AgentPolicy) (map[string][]byte, error)
	AgentPolicyFilePath        string
	IsInteractive              func() bool
	PromptI18nEnable           func() (bool, error)
}

func RunNew(args []string, d NewDeps) int {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			fmt.Fprintln(d.Out, "ship new:")
			fmt.Fprintln(d.Out, "  ship new <app> [--module <module-path>] [--dry-run] [--force] [--ui <franken|daisy|bare>] [--api|--api-only] [--i18n|--no-i18n]")
			return 0
		}
	}

	opts, err := ParseNewArgs(args)
	if err != nil {
		fmt.Fprintf(d.Err, "invalid new arguments: %v\n", err)
		return 1
	}
	if strings.TrimSpace(opts.Name) == "" {
		fmt.Fprintln(d.Err, "usage: ship new <app> [--module <module-path>] [--dry-run] [--force] [--ui <franken|daisy|bare>] [--api|--api-only] [--i18n|--no-i18n]")
		return 1
	}
	opts, err = resolveNewI18nOptions(opts, d)
	if err != nil {
		fmt.Fprintf(d.Err, "invalid new arguments: %v\n", err)
		return 1
	}

	if opts.Module == "" {
		opts.Module = "example.com/" + opts.Name
	}
	if opts.AppPath == "" {
		opts.AppPath = opts.Name
	}

	if err := ScaffoldNewProject(opts, d); err != nil {
		fmt.Fprintf(d.Err, "failed to create project: %v\n", err)
		return 1
	}

	fmt.Fprintf(d.Out, "Created project scaffold at %s\n", opts.AppPath)
	if opts.DryRun {
		fmt.Fprintln(d.Out, "Dry-run mode: no files were written.")
	}
	fmt.Fprintf(d.Out, "GitHub Actions workflows created. Add DEPLOY_KEY secret to enable deployment.\n")
	printNewI18nStatus(d.Out, opts)
	fmt.Fprintf(d.Out, "Next: cd %s && ship db:migrate && ship dev\n", opts.AppPath)
	return 0
}

func ParseNewArgs(args []string) (NewProjectOptions, error) {
	opts := NewProjectOptions{}
	var positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			positionals = append(positionals, arg)
			continue
		}
		switch {
		case arg == "--dry-run":
			opts.DryRun = true
		case arg == "--force":
			opts.Force = true
		case arg == "--api" || arg == "--api-only":
			opts.APIMode = true
		case arg == "--i18n":
			if opts.I18nSet && !opts.I18nEnabled {
				return opts, fmt.Errorf("cannot combine --i18n and --no-i18n")
			}
			opts.I18nEnabled = true
			opts.I18nSet = true
		case arg == "--no-i18n":
			if opts.I18nSet && opts.I18nEnabled {
				return opts, fmt.Errorf("cannot combine --i18n and --no-i18n")
			}
			opts.I18nEnabled = false
			opts.I18nSet = true
		case strings.HasPrefix(arg, "--module="):
			opts.Module = strings.TrimPrefix(arg, "--module=")
		case arg == "--module":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing value for --module")
			}
			i++
			opts.Module = args[i]
		case strings.HasPrefix(arg, "--ui="):
			opts.UIProvider = strings.TrimPrefix(arg, "--ui=")
		case arg == "--ui":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing value for --ui")
			}
			i++
			opts.UIProvider = args[i]
		default:
			return opts, fmt.Errorf("unknown option: %s", arg)
		}
	}
	if len(positionals) > 1 {
		return opts, fmt.Errorf("unexpected positional arguments: %v", positionals[1:])
	}
	if len(positionals) == 1 {
		opts.Name = positionals[0]
	}
	normalizedUI, err := normalizeNewUIProvider(opts.UIProvider)
	if err != nil {
		return opts, err
	}
	opts.UIProvider = normalizedUI
	if err := validateAppName(opts.Name); err != nil {
		return opts, err
	}
	return opts, nil
}

func normalizeNewUIProvider(raw string) (string, error) {
	provider := strings.ToLower(strings.TrimSpace(raw))
	switch provider {
	case "":
		return newUIProviderFranken, nil
	case newUIProviderFranken, newUIProviderDaisy, newUIProviderBare:
		return provider, nil
	default:
		return "", fmt.Errorf("unsupported --ui provider %q (expected franken|daisy|bare)", raw)
	}
}

func validateAppName(name string) error {
	if strings.TrimSpace(name) == "" {
		return nil
	}
	ok, err := regexp.MatchString(`^[a-zA-Z][a-zA-Z0-9_-]*$`, name)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("invalid app name %q", name)
	}
	return nil
}

func ScaffoldNewProject(opts NewProjectOptions, d NewDeps) error {
	if _, err := os.Stat(opts.AppPath); err == nil && !opts.Force {
		return fmt.Errorf("path already exists: %s (use --force to overwrite files)", opts.AppPath)
	}

	files := baseScaffoldFiles(opts)
	starterFiles, err := renderStarterTemplateFiles(opts)
	if err != nil {
		return err
	}
	for path, content := range starterFiles {
		files[path] = content
	}
	if opts.APIMode {
		applyAPIModeScaffold(files, opts)
	}
	for path, content := range i18nScaffoldFiles(opts) {
		files[path] = content
	}
	policyYAML := renderScaffoldAgentPolicyYAML()
	files[filepath.Join(opts.AppPath, d.AgentPolicyFilePath)] = policyYAML
	policy, err := d.ParseAgentPolicyBytes([]byte(policyYAML))
	if err != nil {
		return err
	}
	artifacts, err := d.RenderAgentPolicyArtifacts(policy)
	if err != nil {
		return err
	}
	for rel, content := range artifacts {
		files[filepath.Join(opts.AppPath, rel)] = string(content)
	}

	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		content := files[path]
		if err := writeScaffoldFile(path, content, opts.DryRun, opts.Force); err != nil {
			return err
		}
	}
	return nil
}

func applyAPIModeScaffold(files map[string]string, opts NewProjectOptions) {
	relDelete := []string{
		filepath.Join("app", "views", "web", "pages", "home_feed.templ"),
		filepath.Join("app", "views", "web", "pages", "gen", "home_feed_templ.go"),
		filepath.Join("app", "views", "web", "pages", "landing.templ"),
		filepath.Join("app", "views", "web", "pages", "gen", "landing_templ.go"),
		filepath.Join("app", "views", "web", "pages", "profile.templ"),
		filepath.Join("app", "views", "web", "pages", "gen", "profile_templ.go"),
		filepath.Join("app", "views", "web", "layouts", "base.templ"),
		filepath.Join("static", "styles_bundle.css"),
		filepath.Join("styles", "styles.css"),
	}
	for _, rel := range relDelete {
		delete(files, filepath.Join(opts.AppPath, rel))
	}
	files[filepath.Join(opts.AppPath, "app", "router.go")] = renderAPIOnlyStarterRouter(opts.Module)
	files[filepath.Join(opts.AppPath, "app", "router_test.go")] = renderAPIOnlyStarterRouterTest(opts.Module)
	files[filepath.Join(opts.AppPath, "cmd", "web", "main.go")] = renderAPIOnlyStarterWebMain(opts.Module)
}

func baseScaffoldFiles(opts NewProjectOptions) map[string]string {
	return map[string]string{
		filepath.Join(opts.AppPath, ".env"):                                                       renderStarterDotEnv(opts),
		filepath.Join(opts.AppPath, ".env.example"):                                               renderStarterDotEnvExample(),
		filepath.Join(opts.AppPath, "go.mod"):                                                     renderGoMod(opts),
		filepath.Join(opts.AppPath, "go.sum"):                                                     renderGoSum(),
		filepath.Join(opts.AppPath, "Makefile"):                                                   renderStarterMakefile(),
		filepath.Join(opts.AppPath, "Procfile"):                                                   renderProcfile(),
		filepath.Join(opts.AppPath, "Procfile.dev"):                                               renderProcfileDev(),
		filepath.Join(opts.AppPath, "Procfile.worker"):                                            renderProcfileWorker(),
		filepath.Join(opts.AppPath, "config", "modules.yaml"):                                     renderModulesManifestSkeleton(),
		filepath.Join(opts.AppPath, "db", "bobgen.yaml"):                                          renderBobgenConfigSkeleton(),
		filepath.Join(opts.AppPath, "db", "queries", "user.sql"):                                  renderUserQuerySkeleton(),
		filepath.Join(opts.AppPath, "db", "gen", ".gitkeep"):                                      "",
		filepath.Join(opts.AppPath, "db", "migrate", "migrations", ".gitkeep"):                    "",
		filepath.Join(opts.AppPath, "tmp", ".gitkeep"):                                            "",
		filepath.Join(opts.AppPath, "app", "web", "controllers", "controllers.go"):                renderControllersSkeleton(),
		filepath.Join(opts.AppPath, "app", "web", "middleware", "middleware.go"):                  renderMiddlewareSkeleton(),
		filepath.Join(opts.AppPath, "app", "web", "ui", "ui.go"):                                  renderUISkeleton(),
		filepath.Join(opts.AppPath, "app", "web", "viewmodels", "viewmodels.go"):                  renderViewModelsSkeleton(),
		filepath.Join(opts.AppPath, "app", "jobs", "jobs.go"):                                     renderJobsSkeleton(),
		filepath.Join(opts.AppPath, "app", "profiles", "repo.go"):                                 renderProfilesDomainSkeleton(),
		filepath.Join(opts.AppPath, "app", "notifications", "notifier.go"):                        renderNotificationsDomainSkeleton(),
		filepath.Join(opts.AppPath, "app", "subscriptions", "repo.go"):                            renderSubscriptionsDomainSkeleton(),
		filepath.Join(opts.AppPath, "app", "emailsubscriptions", "repo.go"):                       renderEmailSubscriptionsDomainSkeleton(),
		filepath.Join(opts.AppPath, "app", "views", "web", "layouts", "base.templ"):               renderBaseLayoutTempl(opts.UIProvider),
		filepath.Join(opts.AppPath, "cmd", "worker", "main.go"):                                   renderWorkerMain(),
		filepath.Join(opts.AppPath, "docs", "00-index.md"):                                        renderDocsIndexSkeleton(),
		filepath.Join(opts.AppPath, "docs", "architecture", "01-architecture.md"):                 renderArchitectureSkeleton(),
		filepath.Join(opts.AppPath, "docs", "architecture", "08-cognitive-model.md"):              renderCognitiveModelSkeleton(),
		filepath.Join(opts.AppPath, "docs", "architecture", "10-extension-zones.md"):              renderExtensionZonesSkeleton(),
		filepath.Join(opts.AppPath, "db", "migrate", "migrations", "00001_starter_bootstrap.sql"): renderStarterMigration(),
		filepath.Join(opts.AppPath, ".github", "workflows", "ci.yml"):                             renderGithubCI(),
		filepath.Join(opts.AppPath, ".github", "workflows", "deploy.yml"):                         renderGithubDeploy(),
		filepath.Join(opts.AppPath, ".github", "workflows", "security.yml"):                       renderGithubSecurity(),
		filepath.Join(opts.AppPath, ".github", "dependabot.yml"):                                  renderGithubDependabot(),
		filepath.Join(opts.AppPath, "static", "styles_bundle.css"):                                renderStarterStylesBundle(),
		filepath.Join(opts.AppPath, "styles", "styles.css"):                                       renderStarterStylesSource(),
	}
}

func renderStarterTemplateFiles(opts NewProjectOptions) (map[string]string, error) {
	return renderStarterTemplateFilesFromFS(opts, startertemplate.Files, starterTemplateRoot)
}

func renderStarterTemplateFilesFromFS(opts NewProjectOptions, templateFS fs.FS, root string) (map[string]string, error) {
	if err := validateStarterScaffoldLayout(templateFS, root); err != nil {
		return nil, err
	}

	files := make(map[string]string)
	err := fs.WalkDir(templateFS, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		relPath := strings.TrimPrefix(path, root+"/")
		if relPath == "config/modules.yaml" {
			return nil
		}

		b, readErr := fs.ReadFile(templateFS, path)
		if readErr != nil {
			return fmt.Errorf("starter scaffold layout invalid: failed to read %q: %w", path, readErr)
		}
		content := rewriteStarterTemplate(string(b), opts)
		content = rewriteStarterI18nTemplate(relPath, content, opts)
		files[filepath.Join(opts.AppPath, relPath)] = content
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("starter scaffold layout invalid: failed to walk template root %q: %w", root, err)
	}
	return files, nil
}

func validateStarterScaffoldLayout(templateFS fs.FS, root string) error {
	entries, err := fs.ReadDir(templateFS, root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("starter scaffold layout invalid: missing template root %q", root)
		}
		return fmt.Errorf("starter scaffold layout invalid: unable to read template root %q: %w", root, err)
	}
	if len(entries) == 0 {
		return fmt.Errorf("starter scaffold layout invalid: template root %q is empty", root)
	}

	for _, rel := range canonicalStarterTemplateFiles {
		requiredPath := path.Join(root, rel)
		if _, statErr := fs.Stat(templateFS, requiredPath); statErr != nil {
			if errors.Is(statErr, fs.ErrNotExist) {
				return fmt.Errorf("starter scaffold layout invalid: missing required starter file %q", requiredPath)
			}
			return fmt.Errorf("starter scaffold layout invalid: unable to stat required starter file %q: %w", requiredPath, statErr)
		}
	}
	return nil
}

func rewriteStarterTemplate(content string, opts NewProjectOptions) string {
	replaced := strings.ReplaceAll(content, "github.com/leomorpho/goship/starter", opts.Module)
	replaced = strings.ReplaceAll(replaced, "GoShip Starter", starterDisplayName(opts.Name))
	return replaced
}

func starterDisplayName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "GoShip Starter"
	}
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '-' || r == '_' || unicode.IsSpace(r)
	})
	if len(parts) == 0 {
		return "GoShip Starter"
	}
	for i, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(strings.ToLower(part))
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, " ")
}

func writeScaffoldFile(path, content string, dryRun bool, force bool) error {
	if _, err := os.Stat(path); err == nil && !force {
		return fmt.Errorf("refusing to overwrite existing file: %s", path)
	}
	if dryRun {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func renderGoMod(opts NewProjectOptions) string {
	return fmt.Sprintf(`module %s

go 1.25

require (
	github.com/a-h/templ v0.3.1001
	modernc.org/sqlite v1.46.1
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	golang.org/x/exp v0.0.0-20251023183803-a4bb9ffd2546 // indirect
	golang.org/x/sys v0.37.0 // indirect
	modernc.org/libc v1.67.6 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)
`, opts.Module)
}

func renderGoSum() string {
	return `github.com/a-h/templ v0.3.1001 h1:yHDTgexACdJttyiyamcTHXr2QkIeVF1MukLy44EAhMY=
github.com/a-h/templ v0.3.1001/go.mod h1:oCZcnKRf5jjsGpf2yELzQfodLphd2mwecwG4Crk5HBo=
github.com/dustin/go-humanize v1.0.1 h1:GzkhY7T5VNhEkwH0PVJgjz+fX1rhBrR7pRT3mDkpeCY=
github.com/dustin/go-humanize v1.0.1/go.mod h1:Mu1zIs6XwVuF/gI1OepvI0qD18qycQx+mFykh5fBlto=
github.com/google/go-cmp v0.6.0 h1:ofyhxvXcZhMsU5ulbFiLKl/XBFqE1GSq7atu8tAmTRI=
github.com/google/go-cmp v0.6.0/go.mod h1:17dUlkBOakJ0+DkrSSNjCkIjxS6bF9zb3elmeNGIjoY=
github.com/google/pprof v0.0.0-20250317173921-a4b03ec1a45e h1:ijClszYn+mADRFY17kjQEVQ1XRhq2/JR1M3sGqeJoxs=
github.com/google/pprof v0.0.0-20250317173921-a4b03ec1a45e/go.mod h1:boTsfXsheKC2y+lKOCMpSfarhxDeIzfZG1jqGcPl3cA=
github.com/google/uuid v1.6.0 h1:NIvaJDMOsjHA8n1jAhLSgzrAzy1Hgr+hNrb57e+94F0=
github.com/google/uuid v1.6.0/go.mod h1:TIyPZe4MgqvfeYDBFedMoGGpEw/LqOeaOT+nhxU+yHo=
github.com/hashicorp/golang-lru/v2 v2.0.7 h1:a+bsQ5rvGLjzHuww6tVxozPZFVghXaHOwFs4luLUK2k=
github.com/hashicorp/golang-lru/v2 v2.0.7/go.mod h1:QeFd9opnmA6QUJc5vARoKUSoFhyfM2/ZepoAG6RGpeM=
github.com/mattn/go-isatty v0.0.20 h1:xfD0iDuEKnDkl03q4limB+vH+GxLEtL/jb4xVJSWWEY=
github.com/mattn/go-isatty v0.0.20/go.mod h1:W+V8PltTTMOvKvAeJH7IuucS94S2C6jfK/D7dTCTo3Y=
github.com/ncruces/go-strftime v1.0.0 h1:HMFp8mLCTPp341M/ZnA4qaf7ZlsbTc+miZjCLOFAw7w=
github.com/ncruces/go-strftime v1.0.0/go.mod h1:Fwc5htZGVVkseilnfgOVb9mKy6w1naJmn9CehxcKcls=
github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec h1:W09IVJc94icq4NjY3clb7Lk8O1qJ8BdBEF8z0ibU0rE=
github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec/go.mod h1:qqbHyh8v60DhA7CoWK5oRCqLrMHRGoxYCSS9EjAz6Eo=
golang.org/x/exp v0.0.0-20251023183803-a4bb9ffd2546 h1:mgKeJMpvi0yx/sU5GsxQ7p6s2wtOnGAHZWCHUM4KGzY=
golang.org/x/exp v0.0.0-20251023183803-a4bb9ffd2546/go.mod h1:j/pmGrbnkbPtQfxEe5D0VQhZC6qKbfKifgD0oM7sR70=
golang.org/x/mod v0.29.0 h1:HV8lRxZC4l2cr3Zq1LvtOsi/ThTgWnUk/y64QSs8GwA=
golang.org/x/mod v0.29.0/go.mod h1:NyhrlYXJ2H4eJiRy/WDBO6HMqZQ6q9nk4JzS3NuCK+w=
golang.org/x/sync v0.17.0 h1:l60nONMj9l5drqw6jlhIELNv9I0A4OFgRsG9k2oT9Ug=
golang.org/x/sync v0.17.0/go.mod h1:9KTHXmSnoGruLpwFjVSX0lNNA75CykiMECbovNTZqGI=
golang.org/x/sys v0.6.0/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
golang.org/x/sys v0.37.0 h1:fdNQudmxPjkdUTPnLn5mdQv7Zwvbvpaxqs831goi9kQ=
golang.org/x/sys v0.37.0/go.mod h1:OgkHotnGiDImocRcuBABYBEXf8A9a87e/uXjp9XT3ks=
golang.org/x/tools v0.38.0 h1:Hx2Xv8hISq8Lm16jvBZ2VQf+RLmbd7wVUsALibYI/IQ=
golang.org/x/tools v0.38.0/go.mod h1:yEsQ/d/YK8cjh0L6rZlY8tgtlKiBNTL14pGDJPJpYQs=
modernc.org/cc/v4 v4.27.1 h1:9W30zRlYrefrDV2JE2O8VDtJ1yPGownxciz5rrbQZis=
modernc.org/cc/v4 v4.27.1/go.mod h1:uVtb5OGqUKpoLWhqwNQo/8LwvoiEBLvZXIQ/SmO6mL0=
modernc.org/ccgo/v4 v4.30.1 h1:4r4U1J6Fhj98NKfSjnPUN7Ze2c6MnAdL0hWw6+LrJpc=
modernc.org/ccgo/v4 v4.30.1/go.mod h1:bIOeI1JL54Utlxn+LwrFyjCx2n2RDiYEaJVSrgdrRfM=
modernc.org/fileutil v1.3.40 h1:ZGMswMNc9JOCrcrakF1HrvmergNLAmxOPjizirpfqBA=
modernc.org/fileutil v1.3.40/go.mod h1:HxmghZSZVAz/LXcMNwZPA/DRrQZEVP9VX0V4LQGQFOc=
modernc.org/gc/v2 v2.6.5 h1:nyqdV8q46KvTpZlsw66kWqwXRHdjIlJOhG6kxiV/9xI=
modernc.org/gc/v2 v2.6.5/go.mod h1:YgIahr1ypgfe7chRuJi2gD7DBQiKSLMPgBQe9oIiito=
modernc.org/gc/v3 v3.1.1 h1:k8T3gkXWY9sEiytKhcgyiZ2L0DTyCQ/nvX+LoCljoRE=
modernc.org/gc/v3 v3.1.1/go.mod h1:HFK/6AGESC7Ex+EZJhJ2Gni6cTaYpSMmU/cT9RmlfYY=
modernc.org/goabi0 v0.2.0 h1:HvEowk7LxcPd0eq6mVOAEMai46V+i7Jrj13t4AzuNks=
modernc.org/goabi0 v0.2.0/go.mod h1:CEFRnnJhKvWT1c1JTI3Avm+tgOWbkOu5oPA8eH8LnMI=
modernc.org/libc v1.67.6 h1:eVOQvpModVLKOdT+LvBPjdQqfrZq+pC39BygcT+E7OI=
modernc.org/libc v1.67.6/go.mod h1:JAhxUVlolfYDErnwiqaLvUqc8nfb2r6S6slAgZOnaiE=
modernc.org/mathutil v1.7.1 h1:GCZVGXdaN8gTqB1Mf/usp1Y/hSqgI2vAGGP4jZMCxOU=
modernc.org/mathutil v1.7.1/go.mod h1:4p5IwJITfppl0G4sUEDtCr4DthTaT47/N3aT6MhfgJg=
modernc.org/memory v1.11.0 h1:o4QC8aMQzmcwCK3t3Ux/ZHmwFPzE6hf2Y5LbkRs+hbI=
modernc.org/memory v1.11.0/go.mod h1:/JP4VbVC+K5sU2wZi9bHoq2MAkCnrt2r98UGeSK7Mjw=
modernc.org/opt v0.1.4 h1:2kNGMRiUjrp4LcaPuLY2PzUfqM/w9N23quVwhKt5Qm8=
modernc.org/opt v0.1.4/go.mod h1:03fq9lsNfvkYSfxrfUhZCWPk1lm4cq4N+Bh//bEtgns=
modernc.org/sortutil v1.2.1 h1:+xyoGf15mM3NMlPDnFqrteY07klSFxLElE2PVuWIJ7w=
modernc.org/sortutil v1.2.1/go.mod h1:7ZI3a3REbai7gzCLcotuw9AC4VZVpYMjDzETGsSMqJE=
modernc.org/sqlite v1.46.1 h1:eFJ2ShBLIEnUWlLy12raN0Z1plqmFX9Qe3rjQTKt6sU=
modernc.org/sqlite v1.46.1/go.mod h1:CzbrU2lSB1DKUusvwGz7rqEKIq+NUd8GWuBBZDs9/nA=
modernc.org/strutil v1.2.1 h1:UneZBkQA+DX2Rp35KcM69cSsNES9ly8mQWD71HKlOA0=
modernc.org/strutil v1.2.1/go.mod h1:EHkiggD70koQxjVdSBM3JKM7k6L0FbGE5eymy9i3B9A=
modernc.org/token v1.1.0 h1:Xl7Ap9dKaEs5kLoOQeQmPWevfnk/DM5qcLcYlA8ys6Y=
modernc.org/token v1.1.0/go.mod h1:UGzOrNV1mAFSEB63lOFHIpNRUVMvYTc6yu1SMY/XTDM=
`
}

func renderModulesManifestSkeleton() string {
	return `# Workspace-level module enablement for the monolith.
# Modules are installed globally for the app workspace, not per mini-app.
modules:
`
}

func renderStarterMakefile() string {
	return `.PHONY: migrate run watch-go worker
migrate:
	ship db:migrate

run:
	go run ./cmd/web

watch-go:
	go run ./cmd/web

worker:
	go run ./cmd/worker
`
}

func renderStarterDotEnv(opts NewProjectOptions) string {
	provider := strings.TrimSpace(opts.UIProvider)
	if provider == "" {
		provider = newUIProviderFranken
	}
	return `APP_ENV=development
DB_DRIVER=sqlite
DATABASE_URL=sqlite://tmp/starter.db
PORT=3000
UI_PROVIDER=` + provider + `
`
}

func renderStarterDotEnvExample() string {
	return `# Required: application secret used for cookie/session signing.
APP_KEY=
# Required: primary database connection string.
DATABASE_URL=sqlite://tmp/starter.db

# Optional runtime defaults for local development.
CACHE_DRIVER=memory
QUEUE_DRIVER=backlite

# Common runtime knobs.
APP_ENV=development
DB_DRIVER=sqlite
PORT=3000

# UI provider used by scaffolded layout assets; valid values: franken, daisy, bare.
UI_PROVIDER=franken
`
}

func renderBobgenConfigSkeleton() string {
	return `# Bob SQL generation config.
# Add SQL files under db/queries and run: ship db:generate
version: "v1"
packages:
  - name: "gen"
    path: "db/gen"
    engine: "postgres"
    queries:
      - "db/queries/*.sql"
`
}

func renderProcfile() string {
	return `watch-js: make watch-js
watch-go: make watch-go
watch-css: make watch-css
watch-go-worker: make worker
`
}

func renderProcfileDev() string {
	return `watch-go: make watch-go
watch-go-worker: make worker
`
}

func renderProcfileWorker() string {
	return `watch-go-worker: make worker
`
}

func renderScaffoldAgentPolicyYAML() string {
	return `version: 1
commands:
  - id: go_test
    description: Run Go tests.
    prefix: ["go", "test"]
`
}

func renderControllersSkeleton() string {
	return `package controllers

type landingPage struct{}

func NewLandingPageRoute() landingPage {
	return landingPage{}
}
`
}

func renderMiddlewareSkeleton() string {
	return `package middleware
`
}

func renderUISkeleton() string {
	return `package ui
`
}

func renderViewModelsSkeleton() string {
	return `package viewmodels
`
}

func renderJobsSkeleton() string {
	return `package jobs
`
}

func renderWorkerMain() string {
	return `package main

import "log"

func main() {
	log.Println("starter worker ready: no background jobs registered yet")
}
`
}

func renderAPIOnlyStarterRouter(module string) string {
	return `package goship

import (
	"` + module + `/app/foundation"
	"` + module + `/app/web/routenames"
)

type Route struct {
	Name string
	Path string
}

func BuildRouter(c *foundation.Container) []Route {
	if c == nil {
		c = foundation.NewContainer()
	}

	return []Route{
		{Name: routenames.RouteNameLandingPage, Path: "/"},
		// ship:routes:public:start
		// ship:routes:public:end
		{Name: routenames.RouteNameAPIStatus, Path: "/api/v1/status"},
		{Name: routenames.RouteNameLogin, Path: "/auth/login"},
		{Name: routenames.RouteNameRegister, Path: "/auth/register"},
		// ship:routes:auth:start
		// ship:routes:auth:end
		{Name: routenames.RouteNameHomeFeed, Path: "/auth/homeFeed"},
		{Name: routenames.RouteNameProfile, Path: "/auth/profile"},
		// ship:routes:external:start
		// ship:routes:external:end
	}
}
`
}

func renderAPIOnlyStarterRouterTest(module string) string {
	return `package goship

import (
	"testing"

	"` + module + `/app/web/routenames"
)

func TestBuildRouterIncludesDefaultRoutes(t *testing.T) {
	routes := BuildRouter(nil)
	if len(routes) != 6 {
		t.Fatalf("expected 6 starter routes, got %d", len(routes))
	}

	want := []struct {
		name string
		path string
	}{
		{name: routenames.RouteNameLandingPage, path: "/"},
		{name: routenames.RouteNameAPIStatus, path: "/api/v1/status"},
		{name: routenames.RouteNameLogin, path: "/auth/login"},
		{name: routenames.RouteNameRegister, path: "/auth/register"},
		{name: routenames.RouteNameHomeFeed, path: "/auth/homeFeed"},
		{name: routenames.RouteNameProfile, path: "/auth/profile"},
	}

	for i, route := range routes {
		if route.Name != want[i].name {
			t.Fatalf("route %d name = %q, want %q", i, route.Name, want[i].name)
		}
		if route.Path != want[i].path {
			t.Fatalf("route %d path = %q, want %q", i, route.Path, want[i].path)
		}
	}
}
`
}

func renderAPIOnlyStarterWebMain(module string) string {
	return `package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	goship "` + module + `/app"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/up", func(w http.ResponseWriter, _ *http.Request) {
		writeText(w, http.StatusOK, "alive")
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeText(w, http.StatusOK, "alive")
	})
	mux.HandleFunc("/health/liveness", func(w http.ResponseWriter, _ *http.Request) {
		writeText(w, http.StatusOK, "alive")
	})
	mux.HandleFunc("/health/readiness", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})

	for _, route := range goship.BuildRouter(nil) {
		route := route
		mux.HandleFunc(route.Path, func(w http.ResponseWriter, _ *http.Request) {
			if route.Path == "/api/v1/status" {
				writeJSON(w, http.StatusOK, map[string]any{
					"data": map[string]string{"status": "ok"},
				})
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"route": route.Name, "path": route.Path})
		})
	}

	addr := ":" + envOrDefault("PORT", "3000")
	log.Printf("starter api web listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeText(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}
`
}

func renderProfilesDomainSkeleton() string {
	return `package profiles

type Repo struct{}
`
}

func renderNotificationsDomainSkeleton() string {
	return `package notifications

type Notifier struct{}
`
}

func renderSubscriptionsDomainSkeleton() string {
	return `package subscriptions

type Repo struct{}
`
}

func renderEmailSubscriptionsDomainSkeleton() string {
	return `package emailsubscriptions

type Repo struct{}
`
}

func renderDocsIndexSkeleton() string {
	return `# Documentation Index

Generated by ship.
`
}

func renderArchitectureSkeleton() string {
	return `# Architecture

Generated by ship.
`
}

func renderCognitiveModelSkeleton() string {
	return `# Cognitive Model

Generated by ship.
`
}

func renderExtensionZonesSkeleton() string {
	return `# Extension Zones

## Extension Zones

- ` + "`app/`" + ` for app-owned behavior, UI composition, and docs
- ` + "`framework/`" + ` for reusable framework packages consumed by the generated app
- ` + "`styles/`" + ` and ` + "`static/`" + ` for app-owned assets

## Protected Contract Zones

- ` + "`app/router.go`" + `
- ` + "`app/foundation/container.go`" + `
- ` + "`config/modules.yaml`" + `
- ` + "`tools/agent-policy/allowed-commands.yaml`" + `
`
}

func renderUserQuerySkeleton() string {
	return `-- Model: User
-- Table: users
-- Fields:
-- - email:string

-- name: GetUserByID :one
SELECT * FROM users WHERE id = ?;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ?;
`
}

func renderStarterMigration() string {
	return `-- +goose Up
CREATE TABLE IF NOT EXISTS starter_bootstrap (
    id INTEGER PRIMARY KEY,
    app_name TEXT NOT NULL,
    created_at TEXT NOT NULL
);

INSERT INTO starter_bootstrap (id, app_name, created_at)
VALUES (1, 'GoShip Starter', CURRENT_TIMESTAMP)
ON CONFLICT(id) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS starter_bootstrap;
`
}

func renderStarterStylesSource() string {
	return `/* Starter scaffold ships structural HTML only. Add your own design system styles here. */
`
}

func renderStarterStylesBundle() string {
	return ""
}

func renderGithubCI() string {
	return `name: CI
on: [push, pull_request]
jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.24' }
      - uses: actions/setup-node@v4
        with: { node-version: '22' }
      - run: go install github.com/a-h/templ/cmd/templ@latest
      - run: npm install --prefix frontend
      - run: go run ./tools/cli/ship/cmd/ship verify --profile fast
      - run: go test ./...
`
}

func renderGithubDeploy() string {
	return `name: Deploy
on:
  push:
    branches: [main]
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: webfactory/ssh-agent@v0.9.0
        with: { ssh-private-key: '${{ secrets.DEPLOY_KEY }}' }
      - run: gem install kamal
      - run: kamal deploy
`
}

func renderGithubSecurity() string {
	return `name: Security
on:
  schedule: [{ cron: '0 9 * * 1' }]
jobs:
  govulncheck:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.24' }
      - run: go install golang.org/x/vuln/cmd/govulncheck@latest
      - run: govulncheck ./...
`
}

func renderGithubDependabot() string {
	return `version: 2
updates:
  - package-ecosystem: gomod
    directory: /
    schedule: { interval: weekly }
  - package-ecosystem: npm
    directory: /frontend
    schedule: { interval: weekly }
  - package-ecosystem: github-actions
    directory: /
    schedule: { interval: weekly }
`
}
