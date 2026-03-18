package ship

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/leomorpho/goship/config"
	cmd "github.com/leomorpho/goship/tools/cli/ship/internal/commands"
	gen "github.com/leomorpho/goship/tools/cli/ship/internal/generators"
	policies "github.com/leomorpho/goship/tools/cli/ship/internal/policies"
	rt "github.com/leomorpho/goship/tools/cli/ship/internal/runtime"
)

const (
	gooseDir      = "db/migrate/migrations"
	modelQueryDir = "db/queries"
	eventTypesDir = "framework/events/types"
	gooseGoRunRef = "github.com/pressly/goose/v3/cmd/goose@v3.26.0"
)

var (
	isExecRunnerFn = func(r CmdRunner) bool {
		_, ok := r.(rt.ExecRunner)
		return ok
	}
	gooseLookPathFn = exec.LookPath
)

type CmdRunner = rt.CmdRunner
type ExecRunner = rt.ExecRunner

type CLI struct {
	Out             io.Writer
	Err             io.Writer
	Runner          CmdRunner
	RunDevAll       func() int
	ResolveDevMode  func() (string, error)
	ResolveCompose  func() ([]string, error)
	ResolveDBURL    func() (string, error)
	ResolveDBDriver func() (string, error)
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
	case "dev":
		return c.runDev(args[1:])
	case "new":
		return c.runNew(args[1:])
	case "doctor":
		return c.runDoctor(args[1:])
	case "config":
		return c.runConfig(args[1:])
	case "profile":
		if len(args) == 1 || args[1] == "help" || args[1] == "-h" || args[1] == "--help" {
			cmd.PrintProfileHelp(c.Out)
			return 0
		}
		fmt.Fprintf(c.Err, "use namespaced profile commands, e.g. ship profile:%s\n", args[1])
		return 1
	case "i18n":
		if len(args) == 1 || args[1] == "help" || args[1] == "-h" || args[1] == "--help" {
			cmd.PrintI18nHelp(c.Out)
			return 0
		}
		fmt.Fprintf(c.Err, "use namespaced i18n commands, e.g. ship i18n:%s\n", args[1])
		return 1
	case "routes":
		return c.runRoutes(args[1:])
	case "describe":
		return c.runDescribe(args[1:])
	case "runtime":
		if len(args) == 1 || args[1] == "help" || args[1] == "-h" || args[1] == "--help" {
			cmd.PrintRuntimeReportHelp(c.Out)
			return 0
		}
		fmt.Fprintf(c.Err, "use namespaced runtime commands, e.g. ship runtime:%s\n", args[1])
		return 1
	case "verify":
		return c.runVerify(args[1:])
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
	case "module":
		return c.runModule(rest), true
	case "config":
		return c.runConfig(rest), true
	case "profile":
		return c.runProfile(rest), true
	case "i18n":
		return c.runI18n(rest), true
	case "run":
		return c.runRun(rest), true
	case "runtime":
		return c.runRuntime(rest), true
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
	resolveMode := c.ResolveDevMode
	if resolveMode == nil {
		resolveMode = rt.ResolveDevDefaultMode
	}
	return cmd.RunDev(args, cmd.DevDeps{
		Out:                c.Out,
		Err:                c.Err,
		RunCmd:             c.runCmd,
		RunDevAll:          runner,
		ResolveDefaultMode: resolveMode,
		ResolveWebURL:      rt.ResolveDevWebURL,
		IsInteractive:      rt.IsInteractiveTerminal,
		PromptOpenURL:      rt.PromptOpenBrowser,
		OpenBrowser:        rt.OpenBrowserURL,
	})
}

func (c CLI) runTest(args []string) int {
	return cmd.RunTest(args, cmd.QualityDeps{Out: c.Out, Err: c.Err, RunCmd: c.runCmd, HasFile: hasFile})
}

func (c CLI) runVerify(args []string) int {
	return cmd.RunVerify(args, cmd.VerifyDeps{
		Out:           c.Out,
		Err:           c.Err,
		FindGoModule:  findGoModule,
		RunStep:       c.runCmdCapture,
		LookPath:      exec.LookPath,
		RelocateTempl: relocateTemplGenerated,
	})
}

func (c CLI) runDescribe(args []string) int {
	return cmd.RunDescribe(args, cmd.DescribeDeps{
		Out:          c.Out,
		Err:          c.Err,
		FindGoModule: findGoModule,
	})
}

func (c CLI) runRoutes(args []string) int {
	return cmd.RunRoutes(args, cmd.RoutesDeps{
		Out:          c.Out,
		Err:          c.Err,
		FindGoModule: findGoModule,
	})
}

func (c CLI) runRuntime(args []string) int {
	return cmd.RunRuntimeReport(args, cmd.RuntimeReportDeps{
		Out:        c.Out,
		Err:        c.Err,
		LoadConfig: config.GetConfig,
	})
}

func (c CLI) runDB(args []string) int {
	return cmd.RunDB(args, cmd.DBDeps{
		Out: c.Out, Err: c.Err,
		LoadConfig:      config.GetConfig,
		ResolveDBURL:    c.resolveDBURL,
		ResolveDBDriver: c.resolveDBDriver,
		RunGoose:        c.runGooseCmd,
		RunCmd:          c.runCmd,
		GooseDir:        gooseDir,
		FindGoModule:    findGoModule,
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
	case "event":
		return c.runGenerateEvent(args[1:])
	case "command":
		return c.runMakeCommand(args[1:])
	case "job":
		return c.runMakeJob(args[1:])
	case "mailer":
		return c.runMakeMailer(args[1:])
	case "schedule":
		return c.runMakeSchedule(args[1:])
	case "resource":
		return c.runGenerateResource(args[1:])
	case "factory":
		return c.runMakeFactory(args[1:])
	case "locale":
		return c.runMakeLocale(args[1:])
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
		IsInteractive:              rt.IsInteractiveTerminal,
		PromptI18nEnable: func() (bool, error) {
			return rt.PromptYesNo("Enable i18n in this starter app? [y/N]: ", false)
		},
	})
}

func (c CLI) runUpgrade(args []string) int {
	return cmd.RunUpgrade(args, cmd.UpgradeDeps{Out: c.Out, Err: c.Err, FindGoModule: findGoModule})
}

func (c CLI) runDoctor(args []string) int {
	return policies.RunDoctor(args, policies.DoctorDeps{Out: c.Out, Err: c.Err, FindGoModule: findGoModule})
}

func (c CLI) runConfig(args []string) int {
	return cmd.RunConfig(args, cmd.ConfigDeps{Out: c.Out, Err: c.Err, FindGoModule: findGoModule})
}

func (c CLI) runProfile(args []string) int {
	return cmd.RunProfile(args, cmd.ProfileDeps{Out: c.Out, Err: c.Err})
}

func (c CLI) runI18n(args []string) int {
	return cmd.RunI18n(args, cmd.I18nDeps{Out: c.Out, Err: c.Err, FindGoModule: findGoModule})
}

func (c CLI) runAgent(args []string) int {
	return cmd.RunAgent(args, cmd.AgentDeps{Out: c.Out, Err: c.Err, FindGoModule: findGoModule})
}

func (c CLI) runModule(args []string) int {
	return cmd.RunModule(args, cmd.ModuleDeps{
		Out:          c.Out,
		Err:          c.Err,
		FindGoModule: findGoModule,
	})
}

func (c CLI) runDBMake(args []string) int {
	return c.runDB(append([]string{"make"}, args...))
}

func (c CLI) runGenerateModel(args []string) int {
	return gen.RunGenerateModel(args, gen.GenerateModelDeps{
		Out: c.Out, Err: c.Err,
		RunCmd: c.runCmd, HasFile: hasFile, QueryDir: modelQueryDir,
	})
}

func (c CLI) runGenerateEvent(args []string) int {
	return gen.RunGenerateEvent(args, gen.GenerateEventDeps{
		Out: c.Out, Err: c.Err,
		HasFile: hasFile, TypesDir: eventTypesDir,
	})
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
		RunModel:      c.runGenerateModel,
		RunDBMake:     c.runDBMake,
		RunDBMigrate:  func(args []string) int { return c.runDB(append([]string{"migrate"}, args...)) },
		RunController: c.runMakeController,
		RunResource:   c.runGenerateResource,
	})
}

func (c CLI) runMakeSchedule(args []string) int {
	return gen.RunMakeSchedule(args, gen.ScheduleDeps{
		Out: c.Out,
		Err: c.Err,
	})
}

func (c CLI) runMakeCommand(args []string) int {
	return gen.RunMakeCommand(args, gen.MakeCommandDeps{
		Out: c.Out,
		Err: c.Err,
	})
}

func (c CLI) runMakeJob(args []string) int {
	return gen.RunMakeJob(args, gen.MakeJobDeps{
		Out: c.Out,
		Err: c.Err,
	})
}

func (c CLI) runMakeMailer(args []string) int {
	return gen.RunMakeMailer(args, gen.MakeMailerDeps{
		Out: c.Out,
		Err: c.Err,
	})
}

func (c CLI) runMakeFactory(args []string) int {
	return gen.RunMakeFactory(args, gen.FactoryDeps{
		Out: c.Out,
		Err: c.Err,
	})
}

func (c CLI) runMakeLocale(args []string) int {
	return gen.RunMakeLocale(args, gen.LocaleDeps{
		Out: c.Out,
		Err: c.Err,
	})
}

func (c CLI) runRun(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(c.Err, "usage: ship run:command <name> [-- <args...>]")
		return 1
	}
	switch args[0] {
	case "command":
		return c.runRunCommand(args[1:])
	default:
		fmt.Fprintf(c.Err, "unknown run command: %s\n", args[0])
		return 1
	}
}

func (c CLI) runRunCommand(args []string) int {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		fmt.Fprintln(c.Err, "usage: ship run:command <name> [-- <args...>]")
		return 1
	}

	name := strings.TrimSpace(args[0])
	passthrough := []string{}
	if len(args) > 1 {
		if args[1] == "--" {
			passthrough = append(passthrough, args[2:]...)
		} else {
			passthrough = append(passthrough, args[1:]...)
		}
	}

	runArgs := []string{"run", "./cmd/cli/main.go", name}
	runArgs = append(runArgs, passthrough...)
	return c.runCmd("go", runArgs...)
}
