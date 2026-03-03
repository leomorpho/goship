package ship

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	atlasDir = "file://ent/migrate/migrations"
	atlasURL = "postgres://admin:admin@localhost:5432/app?search_path=public&sslmode=disable"
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
	Out    io.Writer
	Err    io.Writer
	Runner CmdRunner
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

	switch args[0] {
	case "help", "-h", "--help":
		printRootHelp(c.Out)
		return 0
	case "dev", "shipdev":
		return c.runDev(args[1:])
	case "test":
		return c.runTest(args[1:])
	case "db":
		return c.runDB(args[1:])
	case "templ":
		return c.runTempl(args[1:])
	case "generate":
		return c.runGenerate(args[1:])
	default:
		fmt.Fprintf(c.Err, "unknown command: %s\n\n", args[0])
		printRootHelp(c.Err)
		return 1
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
		return c.runCmd("make", "dev")
	case "worker":
		return c.runCmd("make", "dev-worker")
	case "all":
		return c.runCmd("make", "dev-full")
	default:
		fmt.Fprintf(c.Err, "unknown dev mode: %s\n", mode)
		return 1
	}
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
		return c.runCmd("make", "test-integration")
	}
	return c.runCmd("make", "test")
}

func (c CLI) runDB(args []string) int {
	if len(args) == 0 {
		printDBHelp(c.Err)
		return 1
	}

	switch args[0] {
	case "create":
		if len(args) != 1 {
			fmt.Fprintln(c.Err, "usage: ship db create")
			return 1
		}
		// In current local setup, creating DB is equivalent to bringing infra up.
		return c.runCmd("make", "up")
	case "migrate":
		if len(args) != 1 {
			fmt.Fprintln(c.Err, "usage: ship db migrate")
			return 1
		}
		return c.runCmd("make", "migrate")
	case "rollback":
		return c.runDBRollback(args[1:])
	case "seed":
		if len(args) != 1 {
			fmt.Fprintln(c.Err, "usage: ship db seed")
			return 1
		}
		return c.runCmd("make", "seed")
	case "help", "-h", "--help":
		printDBHelp(c.Out)
		return 0
	default:
		fmt.Fprintf(c.Err, "unknown db command: %s\n\n", args[0])
		printDBHelp(c.Err)
		return 1
	}
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

func (c CLI) runGenerate(args []string) int {
	if len(args) == 0 {
		printGenerateHelp(c.Err)
		return 1
	}

	switch args[0] {
	case "resource":
		return c.runGenerateResource(args[1:])
	case "help", "-h", "--help":
		printGenerateHelp(c.Out)
		return 0
	default:
		fmt.Fprintf(c.Err, "unknown generate command: %s\n\n", args[0])
		printGenerateHelp(c.Err)
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
		fmt.Fprintln(c.Err, "usage: ship db rollback [amount]")
		return 1
	}
	if len(args) == 1 {
		if _, err := strconv.Atoi(args[0]); err != nil {
			fmt.Fprintf(c.Err, "invalid rollback amount %q: must be an integer\n", args[0])
			return 1
		}
		amount = args[0]
	}

	return c.runCmd("atlas", "migrate", "down", "--dir", atlasDir, "--url", atlasURL, amount)
}

func (c CLI) runCmd(name string, args ...string) int {
	code, err := c.Runner.Run(name, args...)
	if err != nil {
		fmt.Fprintf(c.Err, "failed to run command %q: %v\n", append([]string{name}, args...), err)
		return 1
	}
	return code
}

func printRootHelp(w io.Writer) {
	fmt.Fprintln(w, "ship - GoShip CLI")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  ship dev [worker|all] [--worker|--all]")
	fmt.Fprintln(w, "  ship test [--integration]")
	fmt.Fprintln(w, "  ship db <create|migrate|rollback|seed>")
	fmt.Fprintln(w, "  ship templ <generate>")
	fmt.Fprintln(w, "  ship generate <resource>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  ship dev")
	fmt.Fprintln(w, "  ship dev worker")
	fmt.Fprintln(w, "  ship dev --all")
	fmt.Fprintln(w, "  ship test --integration")
	fmt.Fprintln(w, "  ship db migrate")
	fmt.Fprintln(w, "  ship db rollback 1")
	fmt.Fprintln(w, "  ship templ generate --path app")
	fmt.Fprintln(w, "  ship generate resource contact")
}

func printDevHelp(w io.Writer) {
	fmt.Fprintln(w, "ship dev commands:")
	fmt.Fprintln(w, "  ship dev")
	fmt.Fprintln(w, "  ship dev worker")
	fmt.Fprintln(w, "  ship dev all")
	fmt.Fprintln(w, "  ship dev --worker")
	fmt.Fprintln(w, "  ship dev --all")
}

func printDBHelp(w io.Writer) {
	fmt.Fprintln(w, "ship db commands:")
	fmt.Fprintln(w, "  ship db create")
	fmt.Fprintln(w, "  ship db migrate")
	fmt.Fprintln(w, "  ship db rollback [amount]")
	fmt.Fprintln(w, "  ship db seed")
}

func printTestHelp(w io.Writer) {
	fmt.Fprintln(w, "ship test commands:")
	fmt.Fprintln(w, "  ship test")
	fmt.Fprintln(w, "  ship test --integration")
}

func printTemplHelp(w io.Writer) {
	fmt.Fprintln(w, "ship templ commands:")
	fmt.Fprintln(w, "  ship templ generate [--path <dir>] [--file <file.templ>]")
	fmt.Fprintln(w, "    (generated files are moved to a child gen/ directory per templ package)")
}

func printGenerateHelp(w io.Writer) {
	fmt.Fprintln(w, "ship generate commands:")
	fmt.Fprintln(w, "  ship generate resource <name> [--path app/goship] [--auth public|auth] [--views templ|none] [--wire] [--dry-run]")
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
