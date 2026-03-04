package ship

import (
	"fmt"

	rt "github.com/leomorpho/goship/tools/cli/ship/internal/runtime"
)

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

func (c CLI) runCmd(name string, args ...string) int {
	return rt.RunCommand(c.getRunner(), c.Err, name, args...)
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
