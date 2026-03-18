package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	policies "github.com/leomorpho/goship/tools/cli/ship/internal/policies"
)

type VerifyDeps struct {
	Out           io.Writer
	Err           io.Writer
	FindGoModule  func(start string) (string, string, error)
	RunStep       func(name string, args ...string) (int, string, error)
	LookPath      func(file string) (string, error)
	RelocateTempl func(rootPath string) error
	RunDoctor     func() (int, string, error)
}

type verifyJSONStep struct {
	Name     string `json:"name"`
	OK       bool   `json:"ok"`
	Output   string `json:"output"`
	Severity string `json:"severity,omitempty"`
}

type verifyJSONResult struct {
	OK    bool             `json:"ok"`
	Steps []verifyJSONStep `json:"steps"`
}

func RunVerify(args []string, d VerifyDeps) int {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			PrintVerifyHelp(d.Out)
			return 0
		}
	}

	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	skipTests := fs.Bool("skip-tests", false, "skip go test ./...")
	jsonOutput := fs.Bool("json", false, "output verify results as JSON")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(d.Err, "invalid verify arguments: %v\n", err)
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(d.Err, "unexpected verify arguments: %v\n", fs.Args())
		return 1
	}

	if d.FindGoModule == nil {
		fmt.Fprintln(d.Err, "verify requires FindGoModule dependency")
		return 1
	}

	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve working directory: %v\n", err)
		return 1
	}
	root, _, err := d.FindGoModule(wd)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to resolve project root (go.mod): %v\n", err)
		return 1
	}

	runStep := d.RunStep
	if runStep == nil {
		runStep = defaultVerifyRunStep
	}
	relocateTempl := d.RelocateTempl
	if relocateTempl == nil {
		relocateTempl = func(string) error { return nil }
	}
	lookPath := d.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	runDoctor := d.RunDoctor
	if runDoctor == nil {
		runDoctor = func() (int, string, error) {
			return runVerifyDoctorJSON(d.FindGoModule)
		}
	}

	results := make([]verifyJSONStep, 0, 5)
	var failed *verifyJSONStep

	appendStep := func(name string, ok bool, output string, severity string) {
		results = append(results, verifyJSONStep{
			Name:     name,
			OK:       ok,
			Output:   strings.TrimSpace(output),
			Severity: strings.TrimSpace(severity),
		})
		if !ok && failed == nil {
			failed = &results[len(results)-1]
		}
	}

	if err := withWorkingDir(root, func() error {
		code, output, runErr := runStep("templ", "generate", "-path", ".")
		if code == 0 && runErr == nil {
			if relocateErr := relocateTempl("."); relocateErr != nil {
				runErr = relocateErr
			}
		}
		appendStep("templ generate", code == 0 && runErr == nil, mergeVerifyOutput(output, runErr), "")
		if failed != nil {
			return nil
		}

		code, output, runErr = runStep("go", "build", "./...")
		appendStep("go build ./...", code == 0 && runErr == nil, mergeVerifyOutput(output, runErr), "")
		if failed != nil {
			return nil
		}

		code, output, runErr = runDoctor()
		appendStep("ship doctor --json", code == 0 && runErr == nil, mergeVerifyOutput(output, runErr), "")
		if failed != nil {
			return nil
		}

		if _, err := lookPath("nilaway"); err != nil {
			appendStep("nilaway ./...", true, "nilaway not installed; skipping", "warning")
		} else {
			code, output, runErr = runStep("nilaway", "./...")
			appendStep("nilaway ./...", code == 0 && runErr == nil, mergeVerifyOutput(output, runErr), "")
			if failed != nil {
				return nil
			}
		}

		if *skipTests {
			appendStep("go test ./...", true, "skipped via --skip-tests", "warning")
		} else {
			code, output, runErr = runStep("go", "test", "./...")
			appendStep("go test ./...", code == 0 && runErr == nil, mergeVerifyOutput(output, runErr), "")
		}
		if failed != nil {
			return nil
		}

		if exportabilityErr := checkStandaloneExportability("."); exportabilityErr != nil {
			appendStep("standalone exportability gate", false, exportabilityErr.Error(), "")
			return nil
		}
		appendStep("standalone exportability gate", true, "starter/runtime surfaces remain free of control-plane dependency drift", "")

		scaffoldSkips, scanErr := findScaffoldSkippedTests(".")
		if scanErr != nil {
			appendStep("scaffold skip checks", true, fmt.Sprintf("Warning: failed to scan scaffold skips: %v", scanErr), "warning")
			return nil
		}
		if len(scaffoldSkips) > 0 {
			appendStep(
				"scaffold skip checks",
				true,
				fmt.Sprintf("Warning: %d scaffolded tests are still skipped.\n%s", len(scaffoldSkips), strings.Join(scaffoldSkips, "\n")),
				"warning",
			)
		}
		return nil
	}); err != nil {
		fmt.Fprintf(d.Err, "verify failed: %v\n", err)
		return 1
	}

	if *jsonOutput {
		return writeVerifyJSON(d.Out, failed == nil, results)
	}

	if failed != nil {
		fmt.Fprintf(d.Err, "verify failed at %s\n", failed.Name)
		if failed.Output != "" {
			fmt.Fprintln(d.Err, failed.Output)
		}
		return 1
	}

	for _, step := range results {
		if step.Severity == "warning" || strings.Contains(step.Output, "skipping") || strings.Contains(step.Output, "skipped") {
			fmt.Fprintf(d.Out, "! %s: %s\n", step.Name, step.Output)
		}
	}
	fmt.Fprintln(d.Out, "✓ verify passed")
	return 0
}

func PrintVerifyHelp(w io.Writer) {
	fmt.Fprintln(w, "ship verify commands:")
	fmt.Fprintln(w, "  ship verify               Run full verification workflow")
	fmt.Fprintln(w, "  ship verify --skip-tests  Skip final test step")
	fmt.Fprintln(w, "  ship verify --json        Output verification result as JSON")
}

func writeVerifyJSON(w io.Writer, ok bool, steps []verifyJSONStep) int {
	payload := verifyJSONResult{OK: ok, Steps: steps}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		fmt.Fprintf(w, "{\"ok\":false,\"steps\":[{\"name\":\"verify\",\"ok\":false,\"output\":%q}]}\n", fmt.Sprintf("failed to encode verify JSON: %v", err))
		return 1
	}
	if ok {
		return 0
	}
	return 1
}

func mergeVerifyOutput(output string, err error) string {
	output = strings.TrimSpace(output)
	if err == nil {
		return output
	}
	if output == "" {
		return err.Error()
	}
	return output + "\n" + err.Error()
}

func defaultVerifyRunStep(name string, args ...string) (int, string, error) {
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

func runVerifyDoctorJSON(findGoModule func(start string) (string, string, error)) (int, string, error) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := policies.RunDoctor([]string{"--json"}, policies.DoctorDeps{
		Out:          out,
		Err:          errOut,
		FindGoModule: findGoModule,
	})
	return code, strings.TrimSpace(out.String() + errOut.String()), nil
}

var (
	scaffoldSkipPattern = regexp.MustCompile(`t\.Skip\(\s*"scaffold:`)
	testFuncPattern     = regexp.MustCompile(`^\s*func\s+(Test[^\s(]+)\s*\(`)
)

func findScaffoldSkippedTests(root string) ([]string, error) {
	results := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "vendor", "node_modules", ".cache":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)

		currentTest := ""
		for _, line := range strings.Split(string(content), "\n") {
			if m := testFuncPattern.FindStringSubmatch(line); len(m) == 2 {
				currentTest = m[1]
			}
			if !scaffoldSkipPattern.MatchString(line) {
				continue
			}
			testName := currentTest
			if strings.TrimSpace(testName) == "" {
				testName = "<unknown>"
			}
			results = append(results, rel+":"+testName)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

func checkStandaloneExportability(root string) error {
	scanRoots := []string{
		filepath.Join(root, "app"),
		filepath.Join(root, "cmd"),
		filepath.Join(root, "config"),
		filepath.Join(root, "framework"),
		filepath.Join(root, "modules"),
	}
	starterRoot := filepath.Join(root, "tools", "cli", "ship", "internal", "templates", "starter", "testdata", "scaffold")
	if info, err := os.Stat(starterRoot); err == nil && info.IsDir() {
		scanRoots = append(scanRoots, starterRoot)
	}

	forbidden := []string{
		"tools/private/control-plane",
		"control-plane dependency",
	}

	for _, scanRoot := range scanRoots {
		info, err := os.Stat(scanRoot)
		if err != nil || !info.IsDir() {
			continue
		}
		err = filepath.WalkDir(scanRoot, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			b, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			text := string(b)
			for _, token := range forbidden {
				if strings.Contains(strings.ToLower(text), strings.ToLower(token)) {
					return fmt.Errorf("control-plane dependency drift detected in %s via %q", path, token)
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}
