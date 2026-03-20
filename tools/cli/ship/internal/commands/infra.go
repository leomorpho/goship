package commands

import (
	"fmt"
	"io"
)

type InfraDeps struct {
	Out            io.Writer
	Err            io.Writer
	ResolveCompose func() ([]string, error)
	RunCmd         func(name string, args ...string) int
}

func RunInfra(args []string, d InfraDeps) int {
	if len(args) == 0 {
		PrintInfraHelp(d.Err)
		return 1
	}

	switch args[0] {
	case "up":
		if len(args) != 1 {
			fmt.Fprintln(d.Err, "usage: ship infra:up")
			return 1
		}
		return runInfraUp(d)
	case "down":
		if len(args) != 1 {
			fmt.Fprintln(d.Err, "usage: ship infra:down")
			return 1
		}
		compose, err := d.ResolveCompose()
		if err != nil {
			fmt.Fprintf(d.Err, "failed to resolve docker compose: %v\n", err)
			return 1
		}
		return d.RunCmd(compose[0], append(compose[1:], "down")...)
	case "help", "-h", "--help":
		PrintInfraHelp(d.Out)
		return 0
	default:
		fmt.Fprintf(d.Err, "unknown infra command: %s\n\n", args[0])
		PrintInfraHelp(d.Err)
		return 1
	}
}

func runInfraUp(d InfraDeps) int {
	compose, err := d.ResolveCompose()
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve docker compose: %v\n", err)
		return 1
	}

	if code := d.RunCmd(compose[0], append(compose[1:], "up", "-d", "cache")...); code != 0 {
		return code
	}
	if code := d.RunCmd(compose[0], append(compose[1:], "up", "-d", "mailpit")...); code != 0 {
		fmt.Fprintln(d.Err, "warning: could not start mailpit; continuing with cache only")
	}
	return 0
}

func PrintInfraHelp(w io.Writer) {
	fmt.Fprintln(w, "ship infra commands:")
	fmt.Fprintln(w, "  ship infra:up")
	fmt.Fprintln(w, "  ship infra:down")
}
