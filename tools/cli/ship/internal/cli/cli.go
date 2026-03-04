package ship

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	cmd "github.com/leomorpho/goship/tools/cli/ship/internal/commands"
	gen "github.com/leomorpho/goship/tools/cli/ship/internal/generators"
	policies "github.com/leomorpho/goship/tools/cli/ship/internal/policies"
	rt "github.com/leomorpho/goship/tools/cli/ship/internal/runtime"
)

const (
	atlasDir      = "file://apps/db/migrate/migrations"
	entSchemaDir  = "apps/db/schema"
	atlasGoRunRef = "ariga.io/atlas/cmd/atlas@v0.27.1"
)

var (
	isExecRunnerFn = func(r CmdRunner) bool {
		_, ok := r.(rt.ExecRunner)
		return ok
	}
	atlasLookPathFn = exec.LookPath
	atlasInstallFn  = func(out io.Writer, errOut io.Writer) (string, error) {
		return rt.InstallAtlasBinary(out, errOut, atlasGoRunRef)
	}
)

type CmdRunner = rt.CmdRunner
type ExecRunner = rt.ExecRunner

type CLI struct {
	Out            io.Writer
	Err            io.Writer
	Runner         CmdRunner
	RunDevAll      func() int
	ResolveCompose func() ([]string, error)
	ResolveDBURL   func() (string, error)
}

func New() CLI {
	return CLI{
		Out:    os.Stdout,
		Err:    os.Stderr,
		Runner: rt.ExecRunner{},
	}
}

// Run executes the ship CLI.
func Run(args []string) int {
	return New().Run(args)
}

func (c CLI) Run(args []string) int {
	if len(args) == 0 {
		cmd.PrintRootHelp(c.Out)
		return 0
	}
	if code, handled := c.runNamespaced(args); handled {
		return code
	}

	switch args[0] {
	case "help", "-h", "--help":
		cmd.PrintRootHelp(c.Out)
		return 0
	case "dev", "shipdev":
		return c.runDev(args[1:])
	case "new":
		return c.runNew(args[1:])
	case "check":
		return c.runCheck(args[1:])
	case "doctor":
		return c.runDoctor(args[1:])
	case "test":
		return c.runTest(args[1:])
	case "agent":
		return c.runAgent(args[1:])
	case "upgrade":
		return c.runUpgrade(args[1:])
	case "db":
		if len(args) == 1 || args[1] == "help" || args[1] == "-h" || args[1] == "--help" {
			cmd.PrintDBHelp(c.Out)
			return 0
		}
		fmt.Fprintf(c.Err, "use namespaced DB commands, e.g. ship db:%s\n", args[1])
		return 1
	case "infra":
		if len(args) == 1 || args[1] == "help" || args[1] == "-h" || args[1] == "--help" {
			cmd.PrintInfraHelp(c.Out)
			return 0
		}
		fmt.Fprintf(c.Err, "use namespaced infra commands, e.g. ship infra:%s\n", args[1])
		return 1
	case "make":
		if len(args) == 1 || args[1] == "help" || args[1] == "-h" || args[1] == "--help" {
			cmd.PrintMakeHelp(c.Out)
			return 0
		}
		fmt.Fprintf(c.Err, "use namespaced make commands, e.g. ship make:%s\n", args[1])
		return 1
	case "templ":
		return c.runTempl(args[1:])
	default:
		fmt.Fprintf(c.Err, "unknown command: %s\n\n", args[0])
		cmd.PrintRootHelp(c.Err)
		return 1
	}
}

func (c CLI) runNamespaced(args []string) (int, bool) {
	ns, sub, ok := strings.Cut(args[0], ":")
	if !ok || ns == "" || sub == "" {
		return 0, false
	}
	rest := append([]string{sub}, args[1:]...)
	switch ns {
	case "db":
		return c.runDB(rest), true
	case "agent":
		return c.runAgent(rest), true
	case "infra":
		return c.runInfra(rest), true
	case "make":
		return c.runMake(rest), true
	case "templ":
		return c.runTempl(rest), true
	default:
		fmt.Fprintf(c.Err, "unknown command namespace: %s\n\n", ns)
		cmd.PrintRootHelp(c.Err)
		return 1, true
	}
}

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

func (c CLI) runInfra(args []string) int {
	resolver := c.ResolveCompose
	if resolver == nil {
		resolver = resolveComposeCommand
	}
	return cmd.RunInfra(args, cmd.InfraDeps{Out: c.Out, Err: c.Err, ResolveCompose: resolver, RunCmd: c.runCmd})
}

func (c CLI) runTempl(args []string) int {
	return cmd.RunTempl(args, cmd.TemplDeps{
		Out:               c.Out,
		Err:               c.Err,
		RunCmd:            c.runCmd,
		RelocateGenerated: relocateTemplGenerated,
	})
}

func (c CLI) runMake(args []string) int {
	if len(args) == 0 {
		cmd.PrintMakeHelp(c.Err)
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
		cmd.PrintMakeHelp(c.Out)
		return 0
	default:
		fmt.Fprintf(c.Err, "unknown make command: %s\n\n", args[0])
		cmd.PrintMakeHelp(c.Err)
		return 1
	}
}

func (c CLI) runNew(args []string) int {
	return cmd.RunNew(args, cmd.NewDeps{
		Out:                        c.Out,
		Err:                        c.Err,
		ParseAgentPolicyBytes:      policies.ParsePolicyBytes,
		RenderAgentPolicyArtifacts: policies.RenderPolicyArtifacts,
		AgentPolicyFilePath:        policies.AgentPolicyFilePath,
	})
}

func (c CLI) runUpgrade(args []string) int {
	return cmd.RunUpgrade(args, cmd.UpgradeDeps{Out: c.Out, Err: c.Err, FindGoModule: findGoModule})
}

func (c CLI) runDoctor(args []string) int {
	return policies.RunDoctor(args, policies.DoctorDeps{Out: c.Out, Err: c.Err, FindGoModule: findGoModule})
}

func (c CLI) runAgent(args []string) int {
	return cmd.RunAgent(args, cmd.AgentDeps{Out: c.Out, Err: c.Err, FindGoModule: findGoModule})
}

func (c CLI) runDBMake(args []string) int {
	return c.runDB(append([]string{"make"}, args...))
}

func (c CLI) runGenerateModel(args []string) int {
	return gen.RunGenerateModel(args, gen.GenerateModelDeps{Out: c.Out, Err: c.Err, RunCmd: c.runCmd, HasFile: hasFile, EntSchemaDir: entSchemaDir})
}

func (c CLI) runGenerateResource(args []string) int {
	return gen.RunGenerateResource(args, c.Out, c.Err)
}

func (c CLI) runMakeController(args []string) int {
	return gen.RunMakeController(args, gen.ControllerDeps{
		Out:                    c.Out,
		Err:                    c.Err,
		HasFile:                hasFile,
		EnsureRouteNamesImport: gen.EnsureRouteNamesImport,
		WireRouteSnippet:       gen.WireRouteSnippet,
	})
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
