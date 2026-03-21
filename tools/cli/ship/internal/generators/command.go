package generators

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	frameworkbootstrap "github.com/leomorpho/goship/framework/bootstrap"
)

type MakeCommandOptions struct {
	Name string
}

type MakeCommandDeps struct {
	Out io.Writer
	Err io.Writer
	Cwd string
}

func RunMakeCommand(args []string, d MakeCommandDeps) int {
	opts, err := ParseMakeCommandArgs(args)
	if err != nil {
		fmt.Fprintf(d.Err, "invalid make:command arguments: %v\n", err)
		return 1
	}

	cwd := d.Cwd
	if strings.TrimSpace(cwd) == "" {
		var wdErr error
		cwd, wdErr = os.Getwd()
		if wdErr != nil {
			fmt.Fprintf(d.Err, "failed to resolve working directory: %v\n", wdErr)
			return 1
		}
	}

	tokens := splitWords(opts.Name)
	if len(tokens) == 0 {
		fmt.Fprintln(d.Err, "invalid make:command arguments: usage: ship make:command <Name>")
		return 1
	}
	pascal := toPascalFromParts(tokens)
	snake := strings.Join(tokens, "_")
	commandName := strings.Join(tokens, ":")

	commandPath := filepath.Join(cwd, "app", "commands", snake+".go")
	if _, err := os.Stat(commandPath); err == nil {
		fmt.Fprintf(d.Err, "refusing to overwrite existing command file: %s\n", commandPath)
		return 1
	}
	commandContent := renderCommandFile(pascal, commandName)
	if err := os.MkdirAll(filepath.Dir(commandPath), 0o755); err != nil {
		fmt.Fprintf(d.Err, "failed to create command directory: %v\n", err)
		return 1
	}
	if err := os.WriteFile(commandPath, []byte(commandContent), 0o644); err != nil {
		fmt.Fprintf(d.Err, "failed to write command file: %v\n", err)
		return 1
	}

	mainPath := filepath.Join(cwd, "cmd", "cli", "main.go")
	mainContent, err := os.ReadFile(mainPath)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to read command runner %s: %v\n", mainPath, err)
		return 1
	}
	snippet := fmt.Sprintf("\tif err := registry.Register(&appcommands.%sCommand{Container: container}); err != nil {\n\t\tlog.Fatal(err)\n\t}\n", pascal)
	updated, changed, err := insertBetweenMarkers(
		string(mainContent),
		"// ship:commands:start",
		"// ship:commands:end",
		snippet,
	)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to wire command registration: %v\n", err)
		return 1
	}
	if changed {
		if err := os.WriteFile(mainPath, []byte(updated), 0o644); err != nil {
			fmt.Fprintf(d.Err, "failed to update command runner: %v\n", err)
			return 1
		}
	}

	writeGeneratorReport(d.Out, "command", false, []string{commandPath}, []string{mainPath}, nil, nil)
	return 0
}

func ParseMakeCommandArgs(args []string) (MakeCommandOptions, error) {
	opts := MakeCommandOptions{}
	if len(args) == 0 {
		return opts, errors.New("usage: ship make:command <Name>")
	}
	opts.Name = strings.TrimSpace(args[0])
	if opts.Name == "" || strings.HasPrefix(opts.Name, "-") {
		return opts, errors.New("usage: ship make:command <Name>")
	}
	for i := 1; i < len(args); i++ {
		if strings.HasPrefix(args[i], "-") {
			return opts, fmt.Errorf("unknown option: %s", args[i])
		}
		return opts, fmt.Errorf("unexpected argument: %s", args[i])
	}
	return opts, nil
}

func renderCommandFile(pascal, commandName string) string {
	return fmt.Sprintf(`package commands

import (
	"context"
	"fmt"

	frameworkbootstrap "github.com/leomorpho/goship/framework/bootstrap"
)

type %sCommand struct {
	Container *frameworkbootstrap.Container
}

func (c *%sCommand) Name() string { return %q }

func (c *%sCommand) Description() string {
	return "TODO: describe this command."
}

func (c *%sCommand) Run(_ context.Context, args []string) error {
	fmt.Printf("%s executed with %%d arg(s)\n", len(args))
	return nil
}
	`, pascal, pascal, commandName, pascal, pascal, commandName)
}
