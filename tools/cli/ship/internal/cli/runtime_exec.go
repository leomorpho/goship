package ship

import (
	"errors"
	"fmt"
	"os/exec"

	rt "github.com/leomorpho/goship/v2/tools/cli/ship/internal/runtime"
)

func (c CLI) runGooseCmd(args ...string) int {
	if !isExecRunnerFn(c.getRunner()) {
		return c.runCmd("goose", args...)
	}
	if _, err := gooseLookPathFn("goose"); err == nil {
		return c.runCmd("goose", args...)
	}
	fmt.Fprintf(c.Out, "goose not found in PATH; running via go module %s\n", gooseGoRunRef)
	goArgs := append([]string{"run", gooseGoRunRef}, args...)
	return c.runCmd("go", goArgs...)
}

func (c CLI) runCmd(name string, args ...string) int {
	return rt.RunCommand(c.getRunner(), c.Err, name, args...)
}

func (c CLI) runCmdCapture(name string, args ...string) (int, string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), string(out), nil
		}
		return 1, string(out), err
	}
	return 0, string(out), nil
}

func (c CLI) getRunner() CmdRunner {
	if c.Runner == nil {
		return rt.ExecRunner{}
	}
	return c.Runner
}

func (c CLI) runDevAll() int {
	return rt.RunDevAll(c.Out, c.Err)
}
