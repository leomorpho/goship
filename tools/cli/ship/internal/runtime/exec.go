package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
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

func RunCommand(r CmdRunner, errOut io.Writer, name string, args ...string) int {
	code, err := r.Run(name, args...)
	if err != nil {
		fmt.Fprintf(errOut, "failed to run command %q: %v\n", append([]string{name}, args...), err)
		return 1
	}
	return code
}

type devProcessExit struct {
	name string
	code int
	err  error
}

func RunDevAll(out io.Writer, errOut io.Writer) int {
	// Check for overmind or goreman
	if path, err := exec.LookPath("overmind"); err == nil {
		fmt.Fprintf(out, "Starting dev session with overmind...\n")
		cmd := exec.Command(path, "start", "-f", "Procfile.dev")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				return exitErr.ExitCode()
			}
			fmt.Fprintf(errOut, "overmind failed: %v\n", err)
			return 1
		}
		return 0
	}

	if path, err := exec.LookPath("goreman"); err == nil {
		fmt.Fprintf(out, "Starting dev session with goreman...\n")
		cmd := exec.Command(path, "-f", "Procfile.dev", "start")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				return exitErr.ExitCode()
			}
			fmt.Fprintf(errOut, "goreman failed: %v\n", err)
			return 1
		}
		return 0
	}

	fmt.Fprintf(errOut, "Neither 'overmind' nor 'goreman' found in PATH.\n")
	fmt.Fprintf(errOut, "Install overmind: brew install overmind (macOS) or see https://github.com/DarthSim/overmind\n")
	fmt.Fprintf(errOut, "Falling back to internal process manager...\n\n")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	processes := []struct {
		name string
		bin  string
		args []string
	}{
		{name: "web", bin: devAllWebBinary(), args: devAllWebArgs()},
		{name: "worker", bin: "go", args: []string{"run", "./cmd/worker"}},
	}

	cmds := make([]*exec.Cmd, 0, len(processes))
	exitCh := make(chan devProcessExit, len(processes))

	for _, proc := range processes {
		command := exec.CommandContext(ctx, proc.bin, proc.args...)
		command.Stdout = newPrefixedWriter(out, proc.name)
		command.Stderr = newPrefixedWriter(errOut, proc.name)
		command.Stdin = os.Stdin
		if err := command.Start(); err != nil {
			stop()
			fmt.Fprintf(errOut, "failed to start %s: %v\n", proc.name, err)
			return 1
		}
		cmds = append(cmds, command)
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
		}(proc.name, command)
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
				fmt.Fprintf(errOut, "%s exited with code %d\n", exit.name, exit.code)
				stop()
				for _, command := range cmds {
					if command.Process != nil {
						_ = command.Process.Signal(syscall.SIGTERM)
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

func devAllWebBinary() string {
	if _, err := os.Stat(".air.toml"); err == nil {
		return "air"
	}
	return "go"
}

func devAllWebArgs() []string {
	if _, err := os.Stat(".air.toml"); err == nil {
		return []string{"-c", ".air.toml"}
	}
	return []string{"run", "./cmd/web"}
}

type prefixedWriter struct {
	out    io.Writer
	prefix string
	mu     sync.Mutex
}

func newPrefixedWriter(out io.Writer, name string) io.Writer {
	return &prefixedWriter{out: out, prefix: "[" + name + "] "}
}

func (w *prefixedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	text := string(p)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line == "" && i == len(lines)-1 {
			continue
		}
		if _, err := io.WriteString(w.out, w.prefix+line+"\n"); err != nil {
			return 0, err
		}
	}
	return len(p), nil
}
