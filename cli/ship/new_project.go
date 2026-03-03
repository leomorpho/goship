package ship

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type newProjectOptions struct {
	Name    string
	Module  string
	DryRun  bool
	Force   bool
	AppPath string
}

func (c CLI) runNew(args []string) int {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			fmt.Fprintln(c.Out, "ship new:")
			fmt.Fprintln(c.Out, "  ship new <app> [--module <module-path>] [--dry-run] [--force]")
			return 0
		}
	}

	opts, err := parseNewArgs(args)
	if err != nil {
		fmt.Fprintf(c.Err, "invalid new arguments: %v\n", err)
		return 1
	}
	if strings.TrimSpace(opts.Name) == "" {
		fmt.Fprintln(c.Err, "usage: ship new <app> [--module <module-path>] [--dry-run] [--force]")
		return 1
	}

	if opts.Module == "" {
		opts.Module = "example.com/" + opts.Name
	}
	if opts.AppPath == "" {
		opts.AppPath = opts.Name
	}

	if err := scaffoldNewProject(opts); err != nil {
		fmt.Fprintf(c.Err, "failed to create project: %v\n", err)
		return 1
	}

	fmt.Fprintf(c.Out, "Created project scaffold at %s\n", opts.AppPath)
	if opts.DryRun {
		fmt.Fprintln(c.Out, "Dry-run mode: no files were written.")
	}
	return 0
}

func parseNewArgs(args []string) (newProjectOptions, error) {
	opts := newProjectOptions{}
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

func scaffoldNewProject(opts newProjectOptions) error {
	if _, err := os.Stat(opts.AppPath); err == nil && !opts.Force {
		return fmt.Errorf("path already exists: %s (use --force to overwrite files)", opts.AppPath)
	}

	files := map[string]string{
		filepath.Join(opts.AppPath, "go.mod"):                                        renderGoMod(opts),
		filepath.Join(opts.AppPath, "app", "goship", "router.go"):                    renderRouterSkeleton(),
		filepath.Join(opts.AppPath, "pkg", "routing", "routenames", "routenames.go"): renderRouteNamesSkeleton(),
		filepath.Join(opts.AppPath, "app", "goship", "views", "templates.go"):        renderTemplatesSkeleton(),
		filepath.Join(opts.AppPath, "app", "goship", "web", "routes", "routes.go"):   renderRoutesSkeleton(),
		filepath.Join(opts.AppPath, "cmd", "web", "main.go"):                         renderWebMain(),
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

func renderGoMod(opts newProjectOptions) string {
	return fmt.Sprintf("module %s\n\ngo 1.25\n", opts.Module)
}

func renderRouterSkeleton() string {
	return `package goship

import (
	routeNames "github.com/leomorpho/goship/pkg/routing/routenames"
	"github.com/leomorpho/goship/app/goship/web/routes"
)

func registerPublicRoutes() {
	_ = routeNames.RouteNameLandingPage
	_ = routes.NewLandingPageRoute
	// ship:routes:public:start
	// ship:routes:public:end
}

func registerAuthRoutes() {
	// ship:routes:auth:start
	// ship:routes:auth:end
}
`
}

func renderRouteNamesSkeleton() string {
	return `package routenames

const (
	RouteNameLandingPage = "landing_page"
)
`
}

func renderTemplatesSkeleton() string {
	return `package templates

type (
	Page string
)
`
}

func renderRoutesSkeleton() string {
	return `package routes

type landingPage struct{}

func NewLandingPageRoute() landingPage {
	return landingPage{}
}
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
