package ship

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

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
		printRootHelp(c.Out)
		return 0
	}
	if code, handled := c.runNamespaced(args); handled {
		return code
	}

	switch args[0] {
	case "help", "-h", "--help":
		printRootHelp(c.Out)
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
			printDBHelp(c.Out)
			return 0
		}
		fmt.Fprintf(c.Err, "use namespaced DB commands, e.g. ship db:%s\n", args[1])
		return 1
	case "infra":
		if len(args) == 1 || args[1] == "help" || args[1] == "-h" || args[1] == "--help" {
			printInfraHelp(c.Out)
			return 0
		}
		fmt.Fprintf(c.Err, "use namespaced infra commands, e.g. ship infra:%s\n", args[1])
		return 1
	case "make":
		if len(args) == 1 || args[1] == "help" || args[1] == "-h" || args[1] == "--help" {
			printMakeHelp(c.Out)
			return 0
		}
		fmt.Fprintf(c.Err, "use namespaced make commands, e.g. ship make:%s\n", args[1])
		return 1
	case "templ":
		return c.runTempl(args[1:])
	default:
		fmt.Fprintf(c.Err, "unknown command: %s\n\n", args[0])
		printRootHelp(c.Err)
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
		printRootHelp(c.Err)
		return 1, true
	}
}
