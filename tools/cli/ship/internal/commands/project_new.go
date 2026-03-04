package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	policies "github.com/leomorpho/goship/tools/cli/ship/internal/policies"
)

type NewProjectOptions struct {
	Name    string
	Module  string
	DryRun  bool
	Force   bool
	AppPath string
}

type NewDeps struct {
	Out                        io.Writer
	Err                        io.Writer
	ParseAgentPolicyBytes      func(b []byte) (policies.AgentPolicy, error)
	RenderAgentPolicyArtifacts func(policy policies.AgentPolicy) (map[string][]byte, error)
	AgentPolicyFilePath        string
}

func RunNew(args []string, d NewDeps) int {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			fmt.Fprintln(d.Out, "ship new:")
			fmt.Fprintln(d.Out, "  ship new <app> [--module <module-path>] [--dry-run] [--force]")
			return 0
		}
	}

	opts, err := ParseNewArgs(args)
	if err != nil {
		fmt.Fprintf(d.Err, "invalid new arguments: %v\n", err)
		return 1
	}
	if strings.TrimSpace(opts.Name) == "" {
		fmt.Fprintln(d.Err, "usage: ship new <app> [--module <module-path>] [--dry-run] [--force]")
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

	files := map[string]string{
		filepath.Join(opts.AppPath, "go.mod"):                                               renderGoMod(opts),
		filepath.Join(opts.AppPath, "config", "modules.yaml"):                               renderModulesManifestSkeleton(),
		filepath.Join(opts.AppPath, "apps", "site", "router.go"):                            renderRouterSkeleton(opts.Module),
		filepath.Join(opts.AppPath, "apps", "site", "web", "routenames", "routenames.go"):   renderRouteNamesSkeleton(),
		filepath.Join(opts.AppPath, "apps", "db", "schema", "user.go"):                      renderUserSchemaSkeleton(),
		filepath.Join(opts.AppPath, "apps", "db", "migrate", "migrations", ".gitkeep"):      "",
		filepath.Join(opts.AppPath, "apps", "site", "views", "templates.go"):                renderTemplatesSkeleton(),
		filepath.Join(opts.AppPath, "apps", "site", "web", "controllers", "controllers.go"): renderControllersSkeleton(),
		filepath.Join(opts.AppPath, "apps", "site", "web", "middleware", "middleware.go"):   renderMiddlewareSkeleton(),
		filepath.Join(opts.AppPath, "apps", "site", "web", "ui", "ui.go"):                   renderUISkeleton(),
		filepath.Join(opts.AppPath, "apps", "site", "web", "viewmodels", "viewmodels.go"):   renderViewModelsSkeleton(),
		filepath.Join(opts.AppPath, "apps", "site", "jobs", "jobs.go"):                      renderJobsSkeleton(),
		filepath.Join(opts.AppPath, "apps", "site", "foundation", "container.go"):           renderContainerSkeleton(),
		filepath.Join(opts.AppPath, "apps", "site", "app", "profiles", "repo.go"):           renderProfilesDomainSkeleton(),
		filepath.Join(opts.AppPath, "apps", "site", "app", "notifications", "notifier.go"):  renderNotificationsDomainSkeleton(),
		filepath.Join(opts.AppPath, "apps", "site", "app", "subscriptions", "repo.go"):      renderSubscriptionsDomainSkeleton(),
		filepath.Join(opts.AppPath, "apps", "site", "app", "emailsubscriptions", "repo.go"): renderEmailSubscriptionsDomainSkeleton(),
		filepath.Join(opts.AppPath, "docs", "00-index.md"):                                  renderDocsIndexSkeleton(),
		filepath.Join(opts.AppPath, "docs", "architecture", "01-architecture.md"):           renderArchitectureSkeleton(),
		filepath.Join(opts.AppPath, "docs", "architecture", "08-cognitive-model.md"):        renderCognitiveModelSkeleton(),
		filepath.Join(opts.AppPath, "cmd", "web", "main.go"):                                renderWebMain(),
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

	for path, content := range files {
		if err := writeScaffoldFile(path, content, opts.DryRun, opts.Force); err != nil {
			return err
		}
	}
	return nil
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

require entgo.io/ent v0.14.0
`, opts.Module)
}

func renderModulesManifestSkeleton() string {
	return `# Workspace-level module enablement.
# Modules apply to the monolith as a whole (not per mini-app).
modules: []
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

func renderRouterSkeleton(module string) string {
	return fmt.Sprintf(`package goship

import (
	routeNames "%s/apps/site/web/routenames"
	"%s/apps/site/web/controllers"
)

func registerPublicRoutes() {
	_ = routeNames.RouteNameLandingPage
	_ = controllers.NewLandingPageRoute
	// ship:routes:public:start
	// ship:routes:public:end
}

func registerAuthRoutes() {
	// ship:routes:auth:start
	// ship:routes:auth:end
}
`, module, module)
}

func renderRouteNamesSkeleton() string {
	return `package routenames

const (
	RouteNameLandingPage = "landing_page"
)
`
}

func renderTemplatesSkeleton() string {
	return `package views

type (
	Page string
)
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

func renderContainerSkeleton() string {
	return `package foundation

type Container struct{}
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

func renderWebMain() string {
	return `package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("GoShip app"))
	})
	if err := http.ListenAndServe(":8000", mux); err != nil {
		log.Fatal(err)
	}
}
`
}

func renderUserSchemaSkeleton() string {
	return `package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("email").NotEmpty(),
	}
}
`
}
