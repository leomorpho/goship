package commands

import (
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type DBDeps struct {
	Out          io.Writer
	Err          io.Writer
	ResolveDBURL func() (string, error)
	RunAtlas     func(args ...string) int
	RunCmd       func(name string, args ...string) int
	AtlasDir     string
	EntSchemaDir string
}

func RunDB(args []string, d DBDeps) int {
	if len(args) == 0 {
		PrintDBHelp(d.Err)
		return 1
	}

	switch args[0] {
	case "create":
		return runCreate(args[1:], d)
	case "make":
		return runMake(args[1:], d)
	case "migrate":
		if len(args) != 1 {
			fmt.Fprintln(d.Err, "usage: ship db:migrate")
			return 1
		}
		dbURL, err := d.ResolveDBURL()
		if err != nil {
			fmt.Fprintf(d.Err, "failed to resolve database URL: %v\n", err)
			return 1
		}
		return d.RunAtlas("migrate", "apply", "--dir", d.AtlasDir, "--url", dbURL)
	case "status":
		return runStatus(args[1:], d)
	case "reset":
		return runReset(args[1:], d)
	case "drop":
		return runDrop(args[1:], d)
	case "rollback":
		return runRollback(args[1:], d)
	case "seed":
		if len(args) != 1 {
			fmt.Fprintln(d.Err, "usage: ship db:seed")
			return 1
		}
		return d.RunCmd("go", "run", "./cmd/seed/main.go")
	case "help", "-h", "--help":
		PrintDBHelp(d.Out)
		return 0
	default:
		fmt.Fprintf(d.Err, "unknown db command: %s\n\n", args[0])
		PrintDBHelp(d.Err)
		return 1
	}
}

func runStatus(args []string, d DBDeps) int {
	if len(args) != 0 {
		fmt.Fprintln(d.Err, "usage: ship db:status")
		return 1
	}

	dbURL, err := d.ResolveDBURL()
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve database URL: %v\n", err)
		return 1
	}

	return d.RunAtlas("migrate", "status", "--dir", d.AtlasDir, "--url", dbURL)
}

func runReset(args []string, d DBDeps) int {
	fs := flag.NewFlagSet("db:reset", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	seed := fs.Bool("seed", false, "seed after reset+migrate")
	force := fs.Bool("force", false, "allow reset on non-local database URLs")
	yes := fs.Bool("yes", false, "confirm destructive reset")
	dryRun := fs.Bool("dry-run", false, "print planned actions without executing")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(d.Err, "invalid db:reset arguments: %v\n", err)
		fmt.Fprintln(d.Err, "usage: ship db:reset [--seed] [--force] [--yes] [--dry-run]")
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintln(d.Err, "usage: ship db:reset [--seed] [--force] [--yes] [--dry-run]")
		return 1
	}

	dbURL, err := d.ResolveDBURL()
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve database URL: %v\n", err)
		return 1
	}
	local := IsLocalDBURL(dbURL)
	if isProductionEnv() && !(*force && *yes) {
		fmt.Fprintln(d.Err, "refusing to reset in production without both --force and --yes")
		return 1
	}
	if !local && !*force {
		fmt.Fprintln(d.Err, "refusing to reset a non-local database without --force")
		return 1
	}
	if !*yes && !*dryRun {
		fmt.Fprintln(d.Err, "refusing destructive reset without --yes (or use --dry-run)")
		return 1
	}

	printPlan(d.Out, "reset", dbURL, local, []string{
		"atlas schema clean --auto-approve",
		"atlas migrate apply",
	}, *seed, *dryRun)
	if *dryRun {
		return 0
	}

	if code := d.RunAtlas("schema", "clean", "--url", dbURL, "--auto-approve"); code != 0 {
		return code
	}
	if code := d.RunAtlas("migrate", "apply", "--dir", d.AtlasDir, "--url", dbURL); code != 0 {
		return code
	}
	if *seed {
		return d.RunCmd("go", "run", "./cmd/seed/main.go")
	}
	return 0
}

func runDrop(args []string, d DBDeps) int {
	fs := flag.NewFlagSet("db:drop", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	force := fs.Bool("force", false, "allow drop on non-local database URLs")
	yes := fs.Bool("yes", false, "confirm destructive drop")
	dryRun := fs.Bool("dry-run", false, "print planned actions without executing")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(d.Err, "invalid db:drop arguments: %v\n", err)
		fmt.Fprintln(d.Err, "usage: ship db:drop [--force] [--yes] [--dry-run]")
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintln(d.Err, "usage: ship db:drop [--force] [--yes] [--dry-run]")
		return 1
	}

	dbURL, err := d.ResolveDBURL()
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve database URL: %v\n", err)
		return 1
	}
	local := IsLocalDBURL(dbURL)
	if isProductionEnv() && !(*force && *yes) {
		fmt.Fprintln(d.Err, "refusing to drop in production without both --force and --yes")
		return 1
	}
	if !local && !*force {
		fmt.Fprintln(d.Err, "refusing to drop a non-local database without --force")
		return 1
	}
	if !*yes && !*dryRun {
		fmt.Fprintln(d.Err, "refusing destructive drop without --yes (or use --dry-run)")
		return 1
	}
	printPlan(d.Out, "drop", dbURL, local, []string{"atlas schema clean --auto-approve"}, false, *dryRun)
	if *dryRun {
		return 0
	}
	return d.RunAtlas("schema", "clean", "--url", dbURL, "--auto-approve")
}

func runCreate(args []string, d DBDeps) int {
	fs := flag.NewFlagSet("db:create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dryRun := fs.Bool("dry-run", false, "print planned actions without executing")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(d.Err, "invalid db:create arguments: %v\n", err)
		fmt.Fprintln(d.Err, "usage: ship db:create [--dry-run]")
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintln(d.Err, "usage: ship db:create [--dry-run]")
		return 1
	}

	dbURL, err := d.ResolveDBURL()
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve database URL: %v\n", err)
		return 1
	}
	local := IsLocalDBURL(dbURL)
	printPlan(d.Out, "create", dbURL, local, []string{"ensure target database is reachable/exists"}, false, *dryRun)
	if *dryRun {
		return 0
	}

	if code := d.RunAtlas("schema", "inspect", "--url", dbURL); code != 0 {
		fmt.Fprintln(d.Err, "database is not reachable or does not exist; create it with your DB provider and retry")
		return code
	}
	return 0
}

func runRollback(args []string, d DBDeps) int {
	amount := "1"
	if len(args) > 1 {
		fmt.Fprintln(d.Err, "usage: ship db:rollback [amount]")
		return 1
	}
	if len(args) == 1 {
		if _, err := strconv.Atoi(args[0]); err != nil {
			fmt.Fprintf(d.Err, "invalid rollback amount %q: must be an integer\n", args[0])
			return 1
		}
		amount = args[0]
	}

	dbURL, err := d.ResolveDBURL()
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve database URL: %v\n", err)
		return 1
	}

	return d.RunAtlas("migrate", "down", "--dir", d.AtlasDir, "--url", dbURL, amount)
}

func runMake(args []string, d DBDeps) int {
	if len(args) != 1 {
		fmt.Fprintln(d.Err, "usage: ship db:make <migration_name>")
		return 1
	}
	name := strings.TrimSpace(args[0])
	if name == "" {
		fmt.Fprintln(d.Err, "usage: ship db:make <migration_name>")
		return 1
	}
	return d.RunAtlas(
		"migrate",
		"diff",
		name,
		"--dir",
		d.AtlasDir,
		"--to",
		"ent://"+d.EntSchemaDir,
		"--dev-url",
		"sqlite://file?mode=memory&_fk=1",
	)
}

func IsLocalDBURL(dbURL string) bool {
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

func printPlan(w io.Writer, action, dbURL string, local bool, steps []string, seed, dryRun bool) {
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

func PrintDBHelp(w io.Writer) {
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
