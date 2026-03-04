package ship

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

const (
	atlasDir      = "file://apps/db/migrate/migrations"
	entSchemaDir  = "apps/db/schema"
	atlasGoRunRef = "ariga.io/atlas/cmd/atlas@v0.27.1"
)

var (
	isExecRunnerFn = func(r CmdRunner) bool {
		_, ok := r.(ExecRunner)
		return ok
	}
	atlasLookPathFn = exec.LookPath
	atlasInstallFn  = installAtlasBinary
)

type CmdRunner interface {
	Run(name string, args ...string) (int, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(name string, args ...string) (int, error) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}
		return 1, err
	}
	return 0, nil
}

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
		Runner: ExecRunner{},
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

func (c CLI) runDev(args []string) int {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			printDevHelp(c.Out)
			return 0
		}
	}

	mode := "web"
	if len(args) > 0 {
		switch args[0] {
		case "worker":
			mode = "worker"
			args = args[1:]
		case "all":
			mode = "all"
			args = args[1:]
		}
	}

	fs := flag.NewFlagSet("dev", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	worker := fs.Bool("worker", false, "run worker-only dev mode")
	all := fs.Bool("all", false, "run full dev mode")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(c.Err, "invalid dev arguments: %v\n", err)
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(c.Err, "unexpected dev arguments: %v\n", fs.Args())
		return 1
	}
	if *worker && *all {
		fmt.Fprintln(c.Err, "cannot set both --worker and --all")
		return 1
	}
	if *worker {
		mode = "worker"
	}
	if *all {
		mode = "all"
	}

	switch mode {
	case "web":
		return c.runCmd("go", "run", "./cmd/web")
	case "worker":
		return c.runCmd("go", "run", "./cmd/worker")
	case "all":
		if c.RunDevAll != nil {
			return c.RunDevAll()
		}
		return c.runDevAll()
	default:
		fmt.Fprintf(c.Err, "unknown dev mode: %s\n", mode)
		return 1
	}
}

type devProcessExit struct {
	name string
	code int
	err  error
}

func (c CLI) runDevAll() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	processes := []struct {
		name string
		args []string
	}{
		{name: "web", args: []string{"run", "./cmd/web"}},
		{name: "worker", args: []string{"run", "./cmd/worker"}},
	}

	cmds := make([]*exec.Cmd, 0, len(processes))
	exitCh := make(chan devProcessExit, len(processes))

	for _, proc := range processes {
		cmd := exec.CommandContext(ctx, "go", proc.args...)
		cmd.Stdout = newPrefixedWriter(c.Out, proc.name)
		cmd.Stderr = newPrefixedWriter(c.Err, proc.name)
		cmd.Stdin = os.Stdin
		if err := cmd.Start(); err != nil {
			stop()
			fmt.Fprintf(c.Err, "failed to start %s: %v\n", proc.name, err)
			return 1
		}
		cmds = append(cmds, cmd)
		go func(name string, started *exec.Cmd) {
			err := started.Wait()
			code := 0
			if err != nil {
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					code = exitErr.ExitCode()
				} else {
					code = 1
				}
			}
			exitCh <- devProcessExit{name: name, code: code, err: err}
		}(proc.name, cmd)
	}

	failed := false
	failedCode := 1
	for range processes {
		exit := <-exitCh
		if exit.code != 0 {
			if ctx.Err() != nil {
				continue
			}
			if !failed {
				failed = true
				failedCode = exit.code
				fmt.Fprintf(c.Err, "%s exited with code %d\n", exit.name, exit.code)
				stop()
				for _, cmd := range cmds {
					if cmd.Process != nil {
						_ = cmd.Process.Signal(syscall.SIGTERM)
					}
				}
			}
		}
	}

	if failed {
		return failedCode
	}
	if ctx.Err() != nil {
		return 130
	}
	return 0
}

func (c CLI) runCheck(args []string) int {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			printCheckHelp(c.Out)
			return 0
		}
	}
	if len(args) > 0 {
		fmt.Fprintf(c.Err, "unexpected check arguments: %v\n", args)
		return 1
	}

	root, hasLists := findProjectRootWithCheckLists()
	if hasLists {
		if err := withWorkingDir(root, func() error {
			unitPkgs, err := readPackageList(filepath.Join("scripts", "test", "unit-packages.txt"))
			if err != nil {
				return err
			}
			for _, pkg := range unitPkgs {
				if code := c.runCmd("go", "test", pkg); code != 0 {
					return fmt.Errorf("go test %s failed with exit code %d", pkg, code)
				}
			}

			compilePkgs, err := readPackageList(filepath.Join("scripts", "test", "compile-packages.txt"))
			if err != nil {
				return err
			}
			for _, pkg := range compilePkgs {
				if code := c.runCmd("go", "test", "-run", "^$", pkg); code != 0 {
					return fmt.Errorf("compile check for %s failed with exit code %d", pkg, code)
				}
			}
			return nil
		}); err != nil {
			fmt.Fprintf(c.Err, "ship check failed: %v\n", err)
			return 1
		}
		return 0
	}
	return c.runCmd("go", "test", "./...")
}

func (c CLI) runTest(args []string) int {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			printTestHelp(c.Out)
			return 0
		}
	}

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	integration := fs.Bool("integration", false, "run integration tests")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(c.Err, "invalid test arguments: %v\n", err)
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(c.Err, "unexpected test arguments: %v\n", fs.Args())
		return 1
	}

	if *integration {
		return c.runCmd("go", "test", "-tags=integration", "./...")
	}
	return c.runCmd("go", "test", "./...")
}

func (c CLI) runDB(args []string) int {
	if len(args) == 0 {
		printDBHelp(c.Err)
		return 1
	}

	switch args[0] {
	case "create":
		return c.runDBCreate(args[1:])
	case "make":
		return c.runDBMake(args[1:])
	case "migrate":
		if len(args) != 1 {
			fmt.Fprintln(c.Err, "usage: ship db:migrate")
			return 1
		}
		dbURL, err := c.resolveDBURL()
		if err != nil {
			fmt.Fprintf(c.Err, "failed to resolve database URL: %v\n", err)
			return 1
		}
		return c.runAtlasCmd("migrate", "apply", "--dir", atlasDir, "--url", dbURL)
	case "status":
		return c.runDBStatus(args[1:])
	case "reset":
		return c.runDBReset(args[1:])
	case "drop":
		return c.runDBDrop(args[1:])
	case "rollback":
		return c.runDBRollback(args[1:])
	case "seed":
		if len(args) != 1 {
			fmt.Fprintln(c.Err, "usage: ship db:seed")
			return 1
		}
		return c.runCmd("go", "run", "./cmd/seed/main.go")
	case "help", "-h", "--help":
		printDBHelp(c.Out)
		return 0
	default:
		fmt.Fprintf(c.Err, "unknown db command: %s\n\n", args[0])
		printDBHelp(c.Err)
		return 1
	}
}

func (c CLI) runDBStatus(args []string) int {
	if len(args) != 0 {
		fmt.Fprintln(c.Err, "usage: ship db:status")
		return 1
	}

	dbURL, err := c.resolveDBURL()
	if err != nil {
		fmt.Fprintf(c.Err, "failed to resolve database URL: %v\n", err)
		return 1
	}

	return c.runAtlasCmd("migrate", "status", "--dir", atlasDir, "--url", dbURL)
}

func (c CLI) runDBReset(args []string) int {
	fs := flag.NewFlagSet("db:reset", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	seed := fs.Bool("seed", false, "seed after reset+migrate")
	force := fs.Bool("force", false, "allow reset on non-local database URLs")
	yes := fs.Bool("yes", false, "confirm destructive reset")
	dryRun := fs.Bool("dry-run", false, "print planned actions without executing")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(c.Err, "invalid db:reset arguments: %v\n", err)
		fmt.Fprintln(c.Err, "usage: ship db:reset [--seed] [--force] [--yes] [--dry-run]")
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintln(c.Err, "usage: ship db:reset [--seed] [--force] [--yes] [--dry-run]")
		return 1
	}

	dbURL, err := c.resolveDBURL()
	if err != nil {
		fmt.Fprintf(c.Err, "failed to resolve database URL: %v\n", err)
		return 1
	}
	local := isLocalDBURL(dbURL)
	if isProductionEnv() && !(*force && *yes) {
		fmt.Fprintln(c.Err, "refusing to reset in production without both --force and --yes")
		return 1
	}
	if !local && !*force {
		fmt.Fprintln(c.Err, "refusing to reset a non-local database without --force")
		return 1
	}
	if !*yes && !*dryRun {
		fmt.Fprintln(c.Err, "refusing destructive reset without --yes (or use --dry-run)")
		return 1
	}

	printDBPlan(c.Out, "reset", dbURL, local, []string{
		"atlas schema clean --auto-approve",
		"atlas migrate apply",
	}, *seed, *dryRun)
	if *dryRun {
		return 0
	}

	// Clean drops all objects in the target schema/database. Use with care.
	if code := c.runAtlasCmd("schema", "clean", "--url", dbURL, "--auto-approve"); code != 0 {
		return code
	}
	if code := c.runAtlasCmd("migrate", "apply", "--dir", atlasDir, "--url", dbURL); code != 0 {
		return code
	}
	if *seed {
		return c.runCmd("go", "run", "./cmd/seed/main.go")
	}
	return 0
}

func (c CLI) runDBDrop(args []string) int {
	fs := flag.NewFlagSet("db:drop", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	force := fs.Bool("force", false, "allow drop on non-local database URLs")
	yes := fs.Bool("yes", false, "confirm destructive drop")
	dryRun := fs.Bool("dry-run", false, "print planned actions without executing")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(c.Err, "invalid db:drop arguments: %v\n", err)
		fmt.Fprintln(c.Err, "usage: ship db:drop [--force] [--yes] [--dry-run]")
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintln(c.Err, "usage: ship db:drop [--force] [--yes] [--dry-run]")
		return 1
	}

	dbURL, err := c.resolveDBURL()
	if err != nil {
		fmt.Fprintf(c.Err, "failed to resolve database URL: %v\n", err)
		return 1
	}
	local := isLocalDBURL(dbURL)
	if isProductionEnv() && !(*force && *yes) {
		fmt.Fprintln(c.Err, "refusing to drop in production without both --force and --yes")
		return 1
	}
	if !local && !*force {
		fmt.Fprintln(c.Err, "refusing to drop a non-local database without --force")
		return 1
	}
	if !*yes && !*dryRun {
		fmt.Fprintln(c.Err, "refusing destructive drop without --yes (or use --dry-run)")
		return 1
	}
	printDBPlan(c.Out, "drop", dbURL, local, []string{
		"atlas schema clean --auto-approve",
	}, false, *dryRun)
	if *dryRun {
		return 0
	}
	return c.runAtlasCmd("schema", "clean", "--url", dbURL, "--auto-approve")
}

func (c CLI) runDBCreate(args []string) int {
	fs := flag.NewFlagSet("db:create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dryRun := fs.Bool("dry-run", false, "print planned actions without executing")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(c.Err, "invalid db:create arguments: %v\n", err)
		fmt.Fprintln(c.Err, "usage: ship db:create [--dry-run]")
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintln(c.Err, "usage: ship db:create [--dry-run]")
		return 1
	}

	dbURL, err := c.resolveDBURL()
	if err != nil {
		fmt.Fprintf(c.Err, "failed to resolve database URL: %v\n", err)
		return 1
	}
	local := isLocalDBURL(dbURL)
	printDBPlan(c.Out, "create", dbURL, local, []string{
		"ensure target database is reachable/exists",
	}, false, *dryRun)
	if *dryRun {
		return 0
	}

	if code := c.runAtlasCmd("schema", "inspect", "--url", dbURL); code != 0 {
		fmt.Fprintln(c.Err, "database is not reachable or does not exist; create it with your DB provider and retry")
		return code
	}
	return 0
}

func isLocalDBURL(dbURL string) bool {
	if strings.HasPrefix(dbURL, "sqlite://") {
		return true
	}
	u, err := url.Parse(dbURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return false
	}
	for _, allowed := range localDBHosts() {
		if host == allowed {
			return true
		}
	}
	return false
}

func localDBHosts() []string {
	raw := strings.TrimSpace(os.Getenv("SHIP_LOCAL_DB_HOSTS"))
	if raw == "" {
		return []string{"localhost", "127.0.0.1", "::1", "db", "postgres", "mysql"}
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		v := strings.ToLower(strings.TrimSpace(part))
		if v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return []string{"localhost", "127.0.0.1", "::1"}
	}
	return out
}

func isProductionEnv() bool {
	env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	return env == "production" || env == "prod"
}

func printDBPlan(w io.Writer, action, dbURL string, local bool, steps []string, seed, dryRun bool) {
	fmt.Fprintf(w, "DB %s plan:\n", action)
	fmt.Fprintf(w, "- url: %s\n", dbURL)
	fmt.Fprintf(w, "- local: %t\n", local)
	for _, step := range steps {
		fmt.Fprintf(w, "- step: %s\n", step)
	}
	if seed {
		fmt.Fprintln(w, "- step: go run ./cmd/seed/main.go")
	}
	if dryRun {
		fmt.Fprintln(w, "- mode: dry-run (no commands executed)")
	}
}

func (c CLI) runInfra(args []string) int {
	if len(args) == 0 {
		printInfraHelp(c.Err)
		return 1
	}

	switch args[0] {
	case "up":
		if len(args) != 1 {
			fmt.Fprintln(c.Err, "usage: ship infra:up")
			return 1
		}
		return c.runInfraUp()
	case "down":
		if len(args) != 1 {
			fmt.Fprintln(c.Err, "usage: ship infra:down")
			return 1
		}
		resolver := c.ResolveCompose
		if resolver == nil {
			resolver = resolveComposeCommand
		}
		compose, err := resolver()
		if err != nil {
			fmt.Fprintf(c.Err, "failed to resolve docker compose: %v\n", err)
			return 1
		}
		name := compose[0]
		baseArgs := compose[1:]
		return c.runCmd(name, append(baseArgs, "down")...)
	case "help", "-h", "--help":
		printInfraHelp(c.Out)
		return 0
	default:
		fmt.Fprintf(c.Err, "unknown infra command: %s\n\n", args[0])
		printInfraHelp(c.Err)
		return 1
	}
}

func (c CLI) runInfraUp() int {
	resolver := c.ResolveCompose
	if resolver == nil {
		resolver = resolveComposeCommand
	}
	compose, err := resolver()
	if err != nil {
		fmt.Fprintf(c.Err, "failed to resolve docker compose: %v\n", err)
		return 1
	}
	name := compose[0]
	baseArgs := compose[1:]

	if code := c.runCmd(name, append(baseArgs, "up", "-d", "cache")...); code != 0 {
		return code
	}
	if code := c.runCmd(name, append(baseArgs, "up", "-d", "mailpit")...); code != 0 {
		// Mailpit should not block local development when SMTP port 1025 is already occupied.
		fmt.Fprintln(c.Err, "warning: could not start mailpit; continuing with cache only")
	}
	return 0
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

func (c CLI) runDBRollback(args []string) int {
	amount := "1"
	if len(args) > 1 {
		fmt.Fprintln(c.Err, "usage: ship db:rollback [amount]")
		return 1
	}
	if len(args) == 1 {
		if _, err := strconv.Atoi(args[0]); err != nil {
			fmt.Fprintf(c.Err, "invalid rollback amount %q: must be an integer\n", args[0])
			return 1
		}
		amount = args[0]
	}

	dbURL, err := c.resolveDBURL()
	if err != nil {
		fmt.Fprintf(c.Err, "failed to resolve database URL: %v\n", err)
		return 1
	}

	return c.runAtlasCmd("migrate", "down", "--dir", atlasDir, "--url", dbURL, amount)
}

func (c CLI) runDBMake(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(c.Err, "usage: ship db:make <migration_name>")
		return 1
	}
	name := strings.TrimSpace(args[0])
	if name == "" {
		fmt.Fprintln(c.Err, "usage: ship db:make <migration_name>")
		return 1
	}
	return c.runAtlasCmd(
		"migrate",
		"diff",
		name,
		"--dir",
		atlasDir,
		"--to",
		"ent://apps/db/schema",
		"--dev-url",
		"sqlite://file?mode=memory&_fk=1",
	)
}

func (c CLI) runAtlasCmd(args ...string) int {
	// For mocked runners in tests, keep behavior deterministic.
	if !isExecRunnerFn(c.getRunner()) {
		return c.runCmd("atlas", args...)
	}
	if _, err := atlasLookPathFn("atlas"); err == nil {
		return c.runCmd("atlas", args...)
	}

	if atlasPath, err := atlasInstallFn(c.Out, c.Err); err == nil {
		fmt.Fprintf(c.Out, "atlas not found in PATH; installed local pinned atlas at %s\n", atlasPath)
		return c.runCmd(atlasPath, args...)
	} else {
		fmt.Fprintf(c.Err, "atlas auto-install failed, falling back to go run: %v\n", err)
	}

	// Final fallback when Atlas is not installed and auto-install failed.
	fmt.Fprintf(c.Out, "atlas not found in PATH; running via go module %s\n", atlasGoRunRef)
	goArgs := append([]string{"run", atlasGoRunRef}, args...)
	return c.runCmd("go", goArgs...)
}

func installAtlasBinary(out io.Writer, errOut io.Writer) (string, error) {
	toolsDir := filepath.Join(".cache", "tools", "bin")
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		return "", fmt.Errorf("create tools dir: %w", err)
	}

	absToolsDir, err := filepath.Abs(toolsDir)
	if err != nil {
		return "", fmt.Errorf("resolve tools dir: %w", err)
	}

	cmd := exec.Command("go", "install", atlasGoRunRef)
	cmd.Stdout = out
	cmd.Stderr = errOut
	cmd.Stdin = os.Stdin
	cmd.Env = append(os.Environ(), "GOBIN="+absToolsDir)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("go install atlas: %w", err)
	}

	atlasBinary := filepath.Join(absToolsDir, "atlas")
	if _, err := os.Stat(atlasBinary); err != nil {
		return "", fmt.Errorf("atlas binary missing after install at %s: %w", atlasBinary, err)
	}

	return atlasBinary, nil
}

func (c CLI) runCmd(name string, args ...string) int {
	code, err := c.getRunner().Run(name, args...)
	if err != nil {
		fmt.Fprintf(c.Err, "failed to run command %q: %v\n", append([]string{name}, args...), err)
		return 1
	}
	return code
}

func (c CLI) getRunner() CmdRunner {
	if c.Runner == nil {
		return ExecRunner{}
	}
	return c.Runner
}

func printRootHelp(w io.Writer) {
	fmt.Fprintln(w, "ship - GoShip CLI")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  ship new <app> [--module <module-path>] [--dry-run] [--force]")
	fmt.Fprintln(w, "  ship dev [worker|all] [--worker|--all]")
	fmt.Fprintln(w, "  ship check")
	fmt.Fprintln(w, "  ship doctor")
	fmt.Fprintln(w, "  ship upgrade --to <version> [--dry-run]")
	fmt.Fprintln(w, "  ship test [--integration]")
	fmt.Fprintln(w, "  ship db:<create|make|migrate|status|reset|drop|rollback|seed>  (or ship db for help)")
	fmt.Fprintln(w, "  ship infra:<up|down>                  (or ship infra for help)")
	fmt.Fprintln(w, "  ship templ <generate>")
	fmt.Fprintln(w, "  ship make:<scaffold|controller|resource|model|module>  (or ship make for help)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  ship new demo")
	fmt.Fprintln(w, "  ship dev")
	fmt.Fprintln(w, "  ship check")
	fmt.Fprintln(w, "  ship doctor")
	fmt.Fprintln(w, "  ship dev worker")
	fmt.Fprintln(w, "  ship dev --all")
	fmt.Fprintln(w, "  ship test --integration")
	fmt.Fprintln(w, "  ship upgrade --to v0.27.1")
	fmt.Fprintln(w, "  ship db:create")
	fmt.Fprintln(w, "  ship db:make add_posts")
	fmt.Fprintln(w, "  ship db:migrate")
	fmt.Fprintln(w, "  ship db:status")
	fmt.Fprintln(w, "  ship db:reset [--seed] [--force] [--yes]")
	fmt.Fprintln(w, "  ship db:drop [--force] [--yes]")
	fmt.Fprintln(w, "  ship db:rollback 1")
	fmt.Fprintln(w, "  ship infra:up")
	fmt.Fprintln(w, "  ship templ generate --path app")
	fmt.Fprintln(w, "  ship make:resource contact")
	fmt.Fprintln(w, "  ship make:model Post title:string")
	fmt.Fprintln(w, "  ship make:module EmailSubscriptions")
}

func printDevHelp(w io.Writer) {
	fmt.Fprintln(w, "ship dev commands:")
	fmt.Fprintln(w, "  ship dev")
	fmt.Fprintln(w, "  ship dev worker")
	fmt.Fprintln(w, "  ship dev all")
	fmt.Fprintln(w, "  ship dev --worker")
	fmt.Fprintln(w, "  ship dev --all")
	fmt.Fprintln(w, "  (default runs web; use --all to run web + worker concurrently)")
}

func printDBHelp(w io.Writer) {
	fmt.Fprintln(w, "ship db commands:")
	fmt.Fprintln(w, "  ship db:create [--dry-run]")
	fmt.Fprintln(w, "  ship db:make <migration_name>")
	fmt.Fprintln(w, "  ship db:migrate")
	fmt.Fprintln(w, "  ship db:status")
	fmt.Fprintln(w, "  ship db:reset [--seed] [--force] [--yes] [--dry-run]")
	fmt.Fprintln(w, "  ship db:drop [--force] [--yes] [--dry-run]")
	fmt.Fprintln(w, "  ship db:rollback [amount]")
	fmt.Fprintln(w, "  ship db:seed")
}

func printInfraHelp(w io.Writer) {
	fmt.Fprintln(w, "ship infra commands:")
	fmt.Fprintln(w, "  ship infra:up")
	fmt.Fprintln(w, "  ship infra:down")
}

func printTestHelp(w io.Writer) {
	fmt.Fprintln(w, "ship test commands:")
	fmt.Fprintln(w, "  ship test")
	fmt.Fprintln(w, "  ship test --integration")
}

func printCheckHelp(w io.Writer) {
	fmt.Fprintln(w, "ship check commands:")
	fmt.Fprintln(w, "  ship check")
}

func printDoctorHelp(w io.Writer) {
	fmt.Fprintln(w, "ship doctor commands:")
	fmt.Fprintln(w, "  ship doctor")
	fmt.Fprintln(w, "  (validates canonical app structure and LLM/DX conventions)")
}

func printUpgradeHelp(w io.Writer) {
	fmt.Fprintln(w, "ship upgrade commands:")
	fmt.Fprintln(w, "  ship upgrade --to <version> [--dry-run]")
	fmt.Fprintln(w, "  (currently upgrades atlas pin only; no auto-latest)")
}

func printTemplHelp(w io.Writer) {
	fmt.Fprintln(w, "ship templ commands:")
	fmt.Fprintln(w, "  ship templ generate [--path <dir>] [--file <file.templ>]")
	fmt.Fprintln(w, "    (generated files are moved to a child gen/ directory per templ package)")
}

func printMakeHelp(w io.Writer) {
	fmt.Fprintln(w, "ship make commands:")
	fmt.Fprintln(w, "  ship make:scaffold <Name> [fields...] [--path apps/goship] [--views templ|none] [--auth public|auth] [--api] [--migrate] [--dry-run] [--force]")
	fmt.Fprintln(w, "  ship make:controller <Name|NameController> [--actions index,show,create,update,destroy] [--auth public|auth] [--wire]")
	fmt.Fprintln(w, "  ship make:resource <name> [--path apps/goship] [--auth public|auth] [--views templ|none] [--wire] [--dry-run]")
	fmt.Fprintln(w, "  ship make:model <Name> [fields...]")
	fmt.Fprintln(w, "  ship make:module <Name> [--path pkg/modules] [--module-base github.com/leomorpho/goship-modules] [--dry-run] [--force]")
}

func relocateTemplGenerated(rootPath string) error {
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(absRoot); errors.Is(err, os.ErrNotExist) {
		return nil
	}

	goModDir, modulePath, err := findGoModule(absRoot)
	if err != nil {
		return err
	}

	var generatedFiles []string
	err = filepath.WalkDir(absRoot, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if d.Name() == "gen" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), "_templ.go") {
			generatedFiles = append(generatedFiles, p)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(generatedFiles) == 0 {
		return nil
	}

	importMap := make(map[string]string)
	movedFiles := make([]string, 0, len(generatedFiles))
	for _, src := range generatedFiles {
		srcDir := filepath.Dir(src)
		relDir, err := filepath.Rel(goModDir, srcDir)
		if err != nil {
			return err
		}
		oldImport := path.Join(modulePath, filepath.ToSlash(relDir))
		newImport := path.Join(oldImport, "gen")
		importMap[oldImport] = newImport

		dstDir := filepath.Join(srcDir, "gen")
		if err := os.MkdirAll(dstDir, 0o755); err != nil {
			return err
		}
		dst := filepath.Join(dstDir, filepath.Base(src))
		_ = os.Remove(dst)
		if err := os.Rename(src, dst); err != nil {
			return err
		}
		movedFiles = append(movedFiles, dst)
	}

	for _, file := range movedFiles {
		b, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		content := string(b)
		for oldImport, newImport := range importMap {
			content = strings.ReplaceAll(content, `"`+oldImport+`"`, `"`+newImport+`"`)
		}
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			return err
		}
	}

	return nil
}

func findGoModule(start string) (string, string, error) {
	dir := start
	for {
		goMod := filepath.Join(dir, "go.mod")
		f, err := os.Open(goMod)
		if err == nil {
			defer f.Close()
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if strings.HasPrefix(line, "module ") {
					modulePath := strings.TrimSpace(strings.TrimPrefix(line, "module "))
					if modulePath == "" {
						return "", "", errors.New("empty module path in go.mod")
					}
					return dir, modulePath, nil
				}
			}
			if scanErr := scanner.Err(); scanErr != nil {
				return "", "", scanErr
			}
			return "", "", errors.New("module line not found in go.mod")
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", "", errors.New("go.mod not found from current path")
		}
		dir = parent
	}
}

func hasFile(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func hasMakefile() bool {
	wd, err := os.Getwd()
	if err != nil {
		return false
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "Makefile")); err == nil {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}

func findProjectRootWithCheckLists() (string, bool) {
	wd, err := os.Getwd()
	if err != nil {
		return "", false
	}
	dir := wd
	for {
		unitPath := filepath.Join(dir, "scripts", "test", "unit-packages.txt")
		compilePath := filepath.Join(dir, "scripts", "test", "compile-packages.txt")
		if hasFile(unitPath) && hasFile(compilePath) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func readPackageList(path string) ([]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(b), "\n")
	pkgs := make([]string, 0, len(lines))
	for _, line := range lines {
		s := strings.TrimSpace(line)
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		pkgs = append(pkgs, s)
	}
	return pkgs, nil
}

func withWorkingDir(dir string, fn func() error) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := os.Chdir(dir); err != nil {
		return err
	}
	defer func() { _ = os.Chdir(wd) }()
	return fn()
}

func resolveComposeCommand() ([]string, error) {
	return resolveComposeCommandWith(exec.LookPath, func() error {
		cmd := exec.Command("docker", "compose", "version")
		return cmd.Run()
	})
}

func resolveComposeCommandWith(lookPath func(string) (string, error), dockerComposeVersion func() error) ([]string, error) {
	if _, err := lookPath("docker-compose"); err == nil {
		return []string{"docker-compose"}, nil
	}
	if _, err := lookPath("docker"); err == nil {
		if err := dockerComposeVersion(); err == nil {
			return []string{"docker", "compose"}, nil
		}
	}
	return nil, errors.New("no docker compose command found (docker-compose or docker compose)")
}

func (c CLI) resolveDBURL() (string, error) {
	if c.ResolveDBURL != nil {
		return c.ResolveDBURL()
	}
	return resolveAtlasDBURL()
}

type atlasConfig struct {
	App struct {
		Environment string `yaml:"environment"`
	} `yaml:"app"`
	Database struct {
		DbMode            string `yaml:"dbMode"`
		Hostname          string `yaml:"hostname"`
		Port              uint16 `yaml:"port"`
		User              string `yaml:"user"`
		Password          string `yaml:"password"`
		DatabaseNameLocal string `yaml:"databaseNameLocal"`
		DatabaseNameProd  string `yaml:"databaseNameProd"`
		TestDatabase      string `yaml:"testDatabase"`
		SslMode           string `yaml:"sslMode"`
		SslCertPath       string `yaml:"sslCertPath"`
	} `yaml:"database"`
}

func resolveAtlasDBURL() (string, error) {
	if u := strings.TrimSpace(os.Getenv("DATABASE_URL")); u != "" {
		return u, nil
	}
	if u := strings.TrimSpace(os.Getenv("PAGODA_DATABASE_URL")); u != "" {
		return "", errors.New("PAGODA_DATABASE_URL is not supported; use DATABASE_URL")
	}

	cfg, err := loadAtlasConfig()
	if err != nil {
		return "", err
	}
	if strings.EqualFold(cfg.Database.DbMode, "embedded") {
		return "", errors.New("database mode is embedded; set DATABASE_URL or switch runtime profile to server-db for atlas migrations")
	}

	env := strings.TrimSpace(os.Getenv("APP_ENV"))
	if env == "" {
		env = strings.TrimSpace(cfg.App.Environment)
	}
	if env == "" {
		env = "local"
	}

	dbName := strings.TrimSpace(cfg.Database.DatabaseNameLocal)
	switch env {
	case "production":
		dbName = strings.TrimSpace(cfg.Database.DatabaseNameProd)
	case "test":
		if t := strings.TrimSpace(cfg.Database.TestDatabase); t != "" {
			dbName = t
		}
	}
	if dbName == "" {
		return "", errors.New("database name is empty in config; set DATABASE_URL or database.databaseNameLocal")
	}
	if strings.TrimSpace(cfg.Database.Hostname) == "" || cfg.Database.Port == 0 {
		return "", errors.New("database host/port missing in config; set DATABASE_URL or database hostname/port")
	}

	query := url.Values{}
	sslMode := strings.TrimSpace(cfg.Database.SslMode)
	if sslMode == "" {
		sslMode = "disable"
	}
	query.Set("sslmode", sslMode)
	if cert := strings.TrimSpace(cfg.Database.SslCertPath); cert != "" {
		query.Set("sslrootcert", cert)
	}

	u := &url.URL{
		Scheme:   "postgresql",
		Host:     net.JoinHostPort(cfg.Database.Hostname, strconv.Itoa(int(cfg.Database.Port))),
		Path:     "/" + dbName,
		RawQuery: query.Encode(),
	}
	if user := strings.TrimSpace(cfg.Database.User); user != "" {
		u.User = url.UserPassword(user, cfg.Database.Password)
	}
	return u.String(), nil
}

func loadAtlasConfig() (atlasConfig, error) {
	var cfg atlasConfig
	configDir, err := findConfigDir()
	if err != nil {
		return cfg, err
	}
	if err := unmarshalYAMLFile(filepath.Join(configDir, "application.yaml"), &cfg); err != nil {
		return cfg, err
	}

	env := strings.TrimSpace(os.Getenv("APP_ENV"))
	if env == "" {
		env = strings.TrimSpace(cfg.App.Environment)
	}
	if env == "" {
		env = "local"
	}
	envFile := filepath.Join(configDir, "environments", env+".yaml")
	if hasFile(envFile) {
		if err := unmarshalYAMLFile(envFile, &cfg); err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}

func findConfigDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for {
		cfgDir := filepath.Join(dir, "config")
		if hasFile(filepath.Join(cfgDir, "application.yaml")) {
			return cfgDir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("config/application.yaml not found; set DATABASE_URL")
		}
		dir = parent
	}
}

func unmarshalYAMLFile(path string, dst any) error {
	cfg, ok := dst.(*atlasConfig)
	if !ok {
		return errors.New("unsupported config type")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return parseAtlasConfigYAML(string(b), cfg)
}

func parseAtlasConfigYAML(content string, cfg *atlasConfig) error {
	section := ""
	lines := strings.Split(content, "\n")
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(raw, " ") && strings.HasSuffix(line, ":") {
			section = strings.TrimSuffix(line, ":")
			continue
		}
		if !strings.HasPrefix(raw, "  ") {
			continue
		}
		key, value, ok := strings.Cut(strings.TrimSpace(raw), ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = normalizeYAMLScalar(value)
		switch section {
		case "app":
			if key == "environment" {
				cfg.App.Environment = value
			}
		case "database":
			switch key {
			case "dbMode":
				cfg.Database.DbMode = value
			case "hostname":
				cfg.Database.Hostname = value
			case "port":
				if v, err := strconv.Atoi(value); err == nil && v > 0 && v <= 65535 {
					cfg.Database.Port = uint16(v)
				}
			case "user":
				cfg.Database.User = value
			case "password":
				cfg.Database.Password = value
			case "databaseNameLocal":
				cfg.Database.DatabaseNameLocal = value
			case "databaseNameProd":
				cfg.Database.DatabaseNameProd = value
			case "testDatabase":
				cfg.Database.TestDatabase = value
			case "sslMode":
				cfg.Database.SslMode = value
			case "sslCertPath":
				cfg.Database.SslCertPath = value
			}
		}
	}
	return nil
}

func normalizeYAMLScalar(v string) string {
	s := strings.TrimSpace(v)
	if idx := strings.Index(s, "#"); idx >= 0 {
		s = strings.TrimSpace(s[:idx])
	}
	s = strings.Trim(s, `"`)
	s = strings.Trim(s, `'`)
	return s
}

type prefixedWriter struct {
	out    io.Writer
	prefix string
	mu     sync.Mutex
}

func newPrefixedWriter(out io.Writer, name string) io.Writer {
	return &prefixedWriter{
		out:    out,
		prefix: "[" + name + "] ",
	}
}

func (w *prefixedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	text := string(p)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		// Preserve trailing newline behavior while still prefixing all complete lines.
		if line == "" && i == len(lines)-1 {
			continue
		}
		if _, err := io.WriteString(w.out, w.prefix+line+"\n"); err != nil {
			return 0, err
		}
	}
	return len(p), nil
}
