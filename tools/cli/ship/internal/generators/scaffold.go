package generators

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

type ScaffoldMakeOptions struct {
	ModelName string
	Fields    []string
	Path      string
	Views     string
	Auth      string
	API       bool
	Migrate   bool
	DryRun    bool
	Force     bool
	TestFirst bool
}

type ScaffoldDeps struct {
	Out           io.Writer
	Err           io.Writer
	RunModel      func(args []string) int
	RunDBMake     func(args []string) int
	RunDBMigrate  func(args []string) int
	RunController func(args []string) int
	RunResource   func(args []string) int
}

func RunMakeScaffold(args []string, d ScaffoldDeps) int {
	opts, err := ParseMakeScaffoldArgs(args)
	if err != nil {
		fmt.Fprintf(d.Err, "invalid make:scaffold arguments: %v\n", err)
		return 1
	}

	controllerName := PluralizeBasic(opts.ModelName)
	resourceName := ModelFileName(opts.ModelName)
	domainName := PluralizeBasic(ModelFileName(opts.ModelName))
	migrationName := "add_" + PluralizeBasic(ModelFileName(opts.ModelName))

	steps := []string{
		"ship make:model " + opts.ModelName + " " + strings.Join(opts.Fields, " "),
		"ship db:make " + migrationName,
		"ship make:controller " + controllerName + " --actions index,show,create,update,destroy --auth " + opts.Auth + " --domain " + domainName + " --wire --path " + opts.Path,
	}
	if opts.TestFirst {
		steps[2] += " --test-first"
	}
	if !opts.API {
		resourceStep := "ship make:resource " + resourceName + " --path " + opts.Path + " --auth " + opts.Auth + " --views " + opts.Views + " --domain " + domainName + " --wire"
		if opts.TestFirst {
			resourceStep += " --test-first"
		}
		steps = append(steps, resourceStep)
	}
	if opts.Migrate {
		steps = append(steps, "ship db:migrate")
	}

	if opts.DryRun {
		fmt.Fprintln(d.Out, "Scaffold plan (dry-run):")
		for _, s := range steps {
			fmt.Fprintf(d.Out, "- %s\n", strings.TrimSpace(s))
		}
		return 0
	}

	modelArgs := []string{opts.ModelName}
	modelArgs = append(modelArgs, opts.Fields...)
	if opts.Force {
		modelArgs = append(modelArgs, "--force")
	}
	if code := d.RunModel(modelArgs); code != 0 {
		return code
	}
	if code := d.RunDBMake([]string{migrationName}); code != 0 {
		return code
	}

	controllerArgs := []string{controllerName, "--actions", "index,show,create,update,destroy", "--auth", opts.Auth, "--domain", domainName, "--wire", "--path", opts.Path}
	if opts.TestFirst {
		controllerArgs = append(controllerArgs, "--test-first")
	}
	if code := d.RunController(controllerArgs); code != 0 {
		return code
	}

	if !opts.API {
		resourceArgs := []string{resourceName, "--path", opts.Path, "--auth", opts.Auth, "--views", opts.Views, "--domain", domainName, "--wire"}
		if opts.TestFirst {
			resourceArgs = append(resourceArgs, "--test-first")
		}
		if code := d.RunResource(resourceArgs); code != 0 {
			return code
		}
	}

	if opts.Migrate {
		if code := d.RunDBMigrate(nil); code != 0 {
			return code
		}
	}

	if opts.TestFirst {
		fmt.Fprintln(d.Out, "\nTests generated. Make them pass, then remove t.Skip calls.")
	}
	return 0
}

func ParseMakeScaffoldArgs(args []string) (ScaffoldMakeOptions, error) {
	opts := ScaffoldMakeOptions{Path: "app", Views: "templ", Auth: "public"}
	if len(args) == 0 {
		return opts, errors.New("usage: ship make:scaffold <Name> [fields...] [--path app] [--views templ|none] [--auth public|auth] [--api] [--migrate] [--dry-run] [--force]")
	}
	opts.ModelName = strings.TrimSpace(args[0])
	if !ModelNamePattern.MatchString(opts.ModelName) {
		return opts, fmt.Errorf("invalid model name %q: use PascalCase", opts.ModelName)
	}

	for _, token := range args[1:] {
		switch {
		case token == "--api":
			opts.API = true
		case token == "--migrate":
			opts.Migrate = true
		case token == "--test-first":
			opts.TestFirst = true
		case token == "--dry-run":
			opts.DryRun = true
		case token == "--force":
			opts.Force = true
		case strings.HasPrefix(token, "--path="):
			opts.Path = strings.TrimSpace(strings.TrimPrefix(token, "--path="))
		case strings.HasPrefix(token, "--views="):
			opts.Views = strings.TrimSpace(strings.TrimPrefix(token, "--views="))
		case strings.HasPrefix(token, "--auth="):
			opts.Auth = strings.TrimSpace(strings.TrimPrefix(token, "--auth="))
		case strings.HasPrefix(token, "--"):
			return opts, fmt.Errorf("unknown option: %s", token)
		default:
			opts.Fields = append(opts.Fields, token)
		}
	}

	if opts.Views != "templ" && opts.Views != "none" {
		return opts, fmt.Errorf("invalid --views value %q (expected templ|none)", opts.Views)
	}
	if opts.Auth != "public" && opts.Auth != "auth" {
		return opts, fmt.Errorf("invalid --auth value %q (expected public|auth)", opts.Auth)
	}
	return opts, nil
}

func PluralizeBasic(v string) string {
	if strings.HasSuffix(v, "s") {
		return v
	}
	return v + "s"
}
