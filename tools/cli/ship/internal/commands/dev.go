package commands

import (
	"flag"
	"fmt"
	"io"
)

type DevDeps struct {
	Out       io.Writer
	Err       io.Writer
	RunCmd    func(name string, args ...string) int
	RunDevAll func() int
}

func RunDev(args []string, d DevDeps) int {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			PrintDevHelp(d.Out)
			return 0
		}
	}

	mode := "all"
	if len(args) > 0 {
		switch args[0] {
		case "web":
			mode = "web"
			args = args[1:]
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
	web := fs.Bool("web", false, "run web-only dev mode")
	worker := fs.Bool("worker", false, "run worker-only dev mode")
	all := fs.Bool("all", false, "run full dev mode (default)")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(d.Err, "invalid dev arguments: %v\n", err)
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(d.Err, "unexpected dev arguments: %v\n", fs.Args())
		return 1
	}
	if (*web && *worker) || (*web && *all) || (*worker && *all) {
		fmt.Fprintln(d.Err, "cannot set more than one of --web, --worker, --all")
		return 1
	}
	if *web {
		mode = "web"
	}
	if *worker {
		mode = "worker"
	}
	if *all {
		mode = "all"
	}

	switch mode {
	case "web":
		return d.RunCmd("go", "run", "./cmd/web")
	case "worker":
		return d.RunCmd("go", "run", "./cmd/worker")
	case "all":
		if d.RunDevAll != nil {
			return d.RunDevAll()
		}
		fmt.Fprintln(d.Err, "dev all runner is not configured")
		return 1
	default:
		fmt.Fprintf(d.Err, "unknown dev mode: %s\n", mode)
		return 1
	}
}

func PrintDevHelp(w io.Writer) {
	fmt.Fprintln(w, "ship dev commands:")
	fmt.Fprintln(w, "  ship dev          run full dev mode (web + worker + js + templ)")
	fmt.Fprintln(w, "  ship dev web      run web-only dev mode")
	fmt.Fprintln(w, "  ship dev worker   run worker-only dev mode")
	fmt.Fprintln(w, "  ship dev --web")
	fmt.Fprintln(w, "  ship dev --worker")
	fmt.Fprintln(w, "  (default multiplexes processes via overmind/goreman and Procfile.dev)")
}
