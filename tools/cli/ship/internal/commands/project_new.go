package commands

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	policies "github.com/leomorpho/goship/tools/cli/ship/internal/policies"
	startertemplate "github.com/leomorpho/goship/tools/cli/ship/internal/templates/starter"
)

type NewProjectOptions struct {
	Name              string
	Module            string
	DryRun            bool
	Force             bool
	AppPath           string
	I18nEnabled       bool
	I18nSet           bool
	I18nLocalePack    string
	I18nLocalePackSet bool
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
			fmt.Fprintln(d.Out, "  ship new <app> [--module <module-path>] [--dry-run] [--force] [--i18n|--no-i18n] [--i18n-locale-pack starter|top15]")
			return 0
		}
	}

	opts, err := ParseNewArgs(args)
	if err != nil {
		fmt.Fprintf(d.Err, "invalid new arguments: %v\n", err)
		return 1
	}
	if strings.TrimSpace(opts.Name) == "" {
		fmt.Fprintln(d.Err, "usage: ship new <app> [--module <module-path>] [--dry-run] [--force] [--i18n|--no-i18n] [--i18n-locale-pack starter|top15]")
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
	fmt.Fprintf(d.Out, "Next: cd %s && ship module:add <module> && make run\n", opts.AppPath)
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
		case strings.HasPrefix(arg, "--i18n-locale-pack="):
			opts.I18nLocalePack = strings.TrimSpace(strings.TrimPrefix(arg, "--i18n-locale-pack="))
			opts.I18nLocalePackSet = true
		case arg == "--i18n-locale-pack":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing value for --i18n-locale-pack")
			}
			i++
			opts.I18nLocalePack = strings.TrimSpace(args[i])
			opts.I18nLocalePackSet = true
		case strings.HasPrefix(arg, "--module="):
			opts.Module = strings.TrimPrefix(arg, "--module=")
		case arg == "--module":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing value for --module")
			}
			i++
			opts.Module = args[i]
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
	if err := validateAppName(opts.Name); err != nil {
		return opts, err
	}
	if opts.I18nLocalePackSet {
		if !isValidI18nLocalePack(opts.I18nLocalePack) {
			return opts, fmt.Errorf("invalid --i18n-locale-pack value %q (supported: starter, top15)", opts.I18nLocalePack)
		}
	}
	return opts, nil
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

func baseScaffoldFiles(opts NewProjectOptions) map[string]string {
	return map[string]string{
		filepath.Join(opts.AppPath, "go.mod"):                                        renderGoMod(opts),
		filepath.Join(opts.AppPath, "Makefile"):                                      renderStarterMakefile(),
		filepath.Join(opts.AppPath, "Procfile"):                                      renderProcfile(),
		filepath.Join(opts.AppPath, "Procfile.dev"):                                  renderProcfileDev(),
		filepath.Join(opts.AppPath, "Procfile.worker"):                               renderProcfileWorker(),
		filepath.Join(opts.AppPath, "config", "modules.yaml"):                        renderModulesManifestSkeleton(),
		filepath.Join(opts.AppPath, "db", "bobgen.yaml"):                             renderBobgenConfigSkeleton(),
		filepath.Join(opts.AppPath, "db", "queries", "user.sql"):                     renderUserQuerySkeleton(),
		filepath.Join(opts.AppPath, "db", "gen", ".gitkeep"):                         "",
		filepath.Join(opts.AppPath, "db", "migrate", "migrations", ".gitkeep"):       "",
		filepath.Join(opts.AppPath, "app", "web", "controllers", "controllers.go"):   renderControllersSkeleton(),
		filepath.Join(opts.AppPath, "app", "web", "middleware", "middleware.go"):     renderMiddlewareSkeleton(),
		filepath.Join(opts.AppPath, "app", "web", "ui", "ui.go"):                     renderUISkeleton(),
		filepath.Join(opts.AppPath, "app", "web", "viewmodels", "viewmodels.go"):     renderViewModelsSkeleton(),
		filepath.Join(opts.AppPath, "app", "jobs", "jobs.go"):                        renderJobsSkeleton(),
		filepath.Join(opts.AppPath, "app", "profiles", "repo.go"):                    renderProfilesDomainSkeleton(),
		filepath.Join(opts.AppPath, "app", "notifications", "notifier.go"):           renderNotificationsDomainSkeleton(),
		filepath.Join(opts.AppPath, "app", "subscriptions", "repo.go"):               renderSubscriptionsDomainSkeleton(),
		filepath.Join(opts.AppPath, "app", "emailsubscriptions", "repo.go"):          renderEmailSubscriptionsDomainSkeleton(),
		filepath.Join(opts.AppPath, "docs", "00-index.md"):                           renderDocsIndexSkeleton(),
		filepath.Join(opts.AppPath, "docs", "architecture", "01-architecture.md"):    renderArchitectureSkeleton(),
		filepath.Join(opts.AppPath, "docs", "architecture", "08-cognitive-model.md"): renderCognitiveModelSkeleton(),
		filepath.Join(opts.AppPath, ".github", "workflows", "ci.yml"):                renderGithubCI(),
		filepath.Join(opts.AppPath, ".github", "workflows", "deploy.yml"):            renderGithubDeploy(),
		filepath.Join(opts.AppPath, ".github", "workflows", "security.yml"):          renderGithubSecurity(),
		filepath.Join(opts.AppPath, ".github", "dependabot.yml"):                     renderGithubDependabot(),
	}
}

func renderStarterTemplateFiles(opts NewProjectOptions) (map[string]string, error) {
	files := make(map[string]string)
	err := fs.WalkDir(startertemplate.Files, "testdata/scaffold", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		relPath := strings.TrimPrefix(path, "testdata/scaffold/")
		if relPath == "config/modules.yaml" {
			return nil
		}

		b, readErr := fs.ReadFile(startertemplate.Files, path)
		if readErr != nil {
			return readErr
		}
		content := rewriteStarterTemplate(string(b), opts)
		content = rewriteStarterI18nTemplate(relPath, content, opts)
		files[filepath.Join(opts.AppPath, relPath)] = content
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
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

require github.com/a-h/templ v0.3.1001
`, opts.Module)
}

func renderModulesManifestSkeleton() string {
	return `# Workspace-level module enablement.
# Modules apply to the monolith as a whole (not per mini-app).
modules: []
`
}

func renderStarterMakefile() string {
	return `.PHONY: run
run:
	go run ./cmd/web
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
      - run: go run ./tools/cli/ship/cmd/ship verify --skip-tests
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
