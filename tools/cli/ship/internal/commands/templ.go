package commands

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
)

type TemplDeps struct {
	Out               io.Writer
	Err               io.Writer
	RunCmd            func(name string, args ...string) int
	RelocateGenerated func(rootPath string) error
}

func RunTempl(args []string, d TemplDeps) int {
	if len(args) == 0 {
		PrintTemplHelp(d.Err)
		return 1
	}

	switch args[0] {
	case "generate":
		return runTemplGenerate(args[1:], d)
	case "help", "-h", "--help":
		PrintTemplHelp(d.Out)
		return 0
	default:
		fmt.Fprintf(d.Err, "unknown templ command: %s\n\n", args[0])
		PrintTemplHelp(d.Err)
		return 1
	}
}

func runTemplGenerate(args []string, d TemplDeps) int {
	fs := flag.NewFlagSet("templ generate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", ".", "path to generate templ files from")
	file := fs.String("file", "", "single .templ file to generate")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(d.Err, "invalid templ generate arguments: %v\n", err)
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(d.Err, "unexpected templ generate arguments: %v\n", fs.Args())
		return 1
	}

	cmdArgs := []string{"generate"}
	if *file != "" {
		cmdArgs = append(cmdArgs, "-f", *file)
	} else {
		cmdArgs = append(cmdArgs, "-path", *path)
	}

	if code := d.RunCmd("templ", cmdArgs...); code != 0 {
		return code
	}

	rootPath := *path
	if *file != "" {
		rootPath = filepath.Dir(*file)
	}
	if err := d.RelocateGenerated(rootPath); err != nil {
		fmt.Fprintf(d.Err, "failed to move generated templ files into gen directories: %v\n", err)
		return 1
	}

	return 0
}
