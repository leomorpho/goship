package commands

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"time"
)

type DevDeps struct {
	Out                io.Writer
	Err                io.Writer
	RunCmd             func(name string, args ...string) int
	RunDevAll          func() int
	ResolveDefaultMode func() (string, error)
	ResolveWebURL      func() (string, error)
	IsInteractive      func() bool
	PromptOpenURL      func(url string) (bool, error)
	OpenBrowser        func(url string) error
}

func RunDev(args []string, d DevDeps) int {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			PrintDevHelp(d.Out)
			return 0
		}
	}

	mode := "web"
	explicitMode := false

	fs := flag.NewFlagSet("dev", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	web := fs.Bool("web", false, "run web-only dev mode")
	worker := fs.Bool("worker", false, "run worker-only dev mode")
	all := fs.Bool("all", false, "run full dev mode")
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
		explicitMode = true
	}
	if *worker {
		mode = "worker"
		explicitMode = true
	}
	if *all {
		mode = "all"
		explicitMode = true
	}

	if !explicitMode && d.ResolveDefaultMode != nil {
		if resolved, err := d.ResolveDefaultMode(); err == nil {
			switch resolved {
			case "web", "worker", "all":
				mode = resolved
			}
		}
	}

	var maybeOpenWhenReady func(done <-chan struct{})
	if mode == "web" || mode == "all" {
		maybeOpenWhenReady = setupDevURLOpen(d)
	}

	switch mode {
	case "web":
		done := make(chan struct{})
		if maybeOpenWhenReady != nil {
			maybeOpenWhenReady(done)
		}
		code := d.RunCmd("air", "-c", ".air.toml")
		close(done)
		return code
	case "worker":
		return d.RunCmd("go", "run", "./cmd/worker")
	case "all":
		if d.RunDevAll != nil {
			done := make(chan struct{})
			if maybeOpenWhenReady != nil {
				maybeOpenWhenReady(done)
			}
			code := d.RunDevAll()
			close(done)
			return code
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
	fmt.Fprintln(w, "  ship dev          Run auto dev mode (default web mode; full mode when jobs backend is asynq)")
	fmt.Fprintln(w, "  ship dev --web    Run explicit web-only dev mode")
	fmt.Fprintln(w, "  ship dev --worker Flag form of worker-only mode")
	fmt.Fprintln(w, "  ship dev --all    Flag form of full mode")
	fmt.Fprintln(w, "  note: full mode multiplexes processes via overmind/goreman and Procfile.dev")
}

func setupDevURLOpen(d DevDeps) func(done <-chan struct{}) {
	if d.ResolveWebURL == nil {
		return nil
	}
	url, err := d.ResolveWebURL()
	if err != nil || url == "" {
		return nil
	}
	fmt.Fprintf(d.Out, "Dev URL: %s\n", url)

	if d.IsInteractive == nil || !d.IsInteractive() {
		return nil
	}
	if d.PromptOpenURL == nil || d.OpenBrowser == nil {
		return nil
	}

	open, err := d.PromptOpenURL(url)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to read browser prompt: %v\n", err)
		return nil
	}
	if !open {
		return nil
	}

	return func(done <-chan struct{}) {
		go waitForURLAndOpen(done, d, url)
	}
}

func waitForURLAndOpen(done <-chan struct{}, d DevDeps, url string) {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	deadline := time.Now().Add(30 * time.Second)

	for time.Now().Before(deadline) {
		select {
		case <-done:
			return
		default:
		}

		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 500 {
				if err := d.OpenBrowser(url); err != nil {
					fmt.Fprintf(d.Err, "failed to open browser: %v\n", err)
				}
				return
			}
		}

		select {
		case <-done:
			return
		case <-time.After(250 * time.Millisecond):
		}
	}
}
