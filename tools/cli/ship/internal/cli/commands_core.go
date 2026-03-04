package ship

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"

	cmd "github.com/leomorpho/goship/tools/cli/ship/internal/commands"
	gen "github.com/leomorpho/goship/tools/cli/ship/internal/generators"
)

func (c CLI) runDev(args []string) int {
	runner := c.RunDevAll
	if runner == nil {
		runner = c.runDevAll
	}
	return cmd.RunDev(args, cmd.DevDeps{Out: c.Out, Err: c.Err, RunCmd: c.runCmd, RunDevAll: runner})
}

func (c CLI) runCheck(args []string) int {
	return cmd.RunCheck(args, cmd.QualityDeps{Out: c.Out, Err: c.Err, RunCmd: c.runCmd, HasFile: hasFile})
}

func (c CLI) runTest(args []string) int {
	return cmd.RunTest(args, cmd.QualityDeps{Out: c.Out, Err: c.Err, RunCmd: c.runCmd})
}

func (c CLI) runDB(args []string) int {
	return cmd.RunDB(args, cmd.DBDeps{
		Out: c.Out, Err: c.Err,
		ResolveDBURL: c.resolveDBURL,
		RunAtlas:     c.runAtlasCmd,
		RunCmd:       c.runCmd,
		AtlasDir:     atlasDir,
		EntSchemaDir: entSchemaDir,
	})
}

func isLocalDBURL(dbURL string) bool { return cmd.IsLocalDBURL(dbURL) }

func (c CLI) runDBMake(args []string) int {
	return c.runDB(append([]string{"make"}, args...))
}

func (c CLI) runInfra(args []string) int {
	resolver := c.ResolveCompose
	if resolver == nil {
		resolver = resolveComposeCommand
	}
	return cmd.RunInfra(args, cmd.InfraDeps{Out: c.Out, Err: c.Err, ResolveCompose: resolver, RunCmd: c.runCmd})
}

func (c CLI) runTempl(args []string) int {
	if len(args) == 0 {
		printTemplHelp(c.Err)
		return 1
	}

	switch args[0] {
	case "generate":
		return c.runTemplGenerate(args[1:])
	case "help", "-h", "--help":
		printTemplHelp(c.Out)
		return 0
	default:
		fmt.Fprintf(c.Err, "unknown templ command: %s\n\n", args[0])
		printTemplHelp(c.Err)
		return 1
	}
}

func (c CLI) runMake(args []string) int {
	if len(args) == 0 {
		printMakeHelp(c.Err)
		return 1
	}

	switch args[0] {
	case "scaffold":
		return c.runMakeScaffold(args[1:])
	case "controller":
		return c.runMakeController(args[1:])
	case "module":
		return c.runMakeModule(args[1:])
	case "model":
		return c.runGenerateModel(args[1:])
	case "resource":
		return c.runGenerateResource(args[1:])
	case "help", "-h", "--help":
		printMakeHelp(c.Out)
		return 0
	default:
		fmt.Fprintf(c.Err, "unknown make command: %s\n\n", args[0])
		printMakeHelp(c.Err)
		return 1
	}
}

func (c CLI) runTemplGenerate(args []string) int {
	fs := flag.NewFlagSet("templ generate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", ".", "path to generate templ files from")
	file := fs.String("file", "", "single .templ file to generate")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(c.Err, "invalid templ generate arguments: %v\n", err)
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(c.Err, "unexpected templ generate arguments: %v\n", fs.Args())
		return 1
	}

	cmdArgs := []string{"generate"}
	if *file != "" {
		cmdArgs = append(cmdArgs, "-f", *file)
	} else {
		cmdArgs = append(cmdArgs, "-path", *path)
	}

	if code := c.runCmd("templ", cmdArgs...); code != 0 {
		return code
	}

	rootPath := *path
	if *file != "" {
		rootPath = filepath.Dir(*file)
	}
	if err := relocateTemplGenerated(rootPath); err != nil {
		fmt.Fprintf(c.Err, "failed to move generated templ files into gen directories: %v\n", err)
		return 1
	}

	return 0
}

func (c CLI) runGenerateModel(args []string) int {
	return gen.RunGenerateModel(args, gen.GenerateModelDeps{Out: c.Out, Err: c.Err, RunCmd: c.runCmd, HasFile: hasFile, EntSchemaDir: entSchemaDir})
}

func (c CLI) runGenerateResource(args []string) int {
	return gen.RunGenerateResource(args, c.Out, c.Err)
}

func (c CLI) runMakeController(args []string) int {
	return gen.RunMakeController(args, gen.ControllerDeps{Out: c.Out, Err: c.Err, HasFile: hasFile, EnsureRouteNamesImport: gen.EnsureRouteNamesImport, WireRouteSnippet: gen.WireRouteSnippet})
}

func (c CLI) runMakeModule(args []string) int {
	return gen.RunMakeModule(args, gen.ModuleDeps{Out: c.Out, Err: c.Err, PathExists: pathExists})
}

func (c CLI) runMakeScaffold(args []string) int {
	return gen.RunMakeScaffold(args, gen.ScaffoldDeps{
		Out:           c.Out,
		Err:           c.Err,
		ParseDBURL:    c.resolveDBURL,
		RunCmd:        c.runCmd,
		RunModel:      c.runGenerateModel,
		RunDBMake:     c.runDBMake,
		RunController: c.runMakeController,
		RunResource:   c.runGenerateResource,
		AtlasDir:      atlasDir,
	})
}
