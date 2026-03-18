package commands

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type QualityDeps struct {
	Out     io.Writer
	Err     io.Writer
	RunCmd  func(name string, args ...string) int
	HasFile func(path string) bool
}

func RunTest(args []string, d QualityDeps) int {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			PrintTestHelp(d.Out)
			return 0
		}
	}

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	integration := fs.Bool("integration", false, "run integration tests")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(d.Err, "invalid test arguments: %v\n", err)
		return 1
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(d.Err, "unexpected test arguments: %v\n", fs.Args())
		return 1
	}

	if *integration {
		return d.RunCmd("go", "test", "-tags=integration", "./...")
	}
	if err := runDefaultTestSuite(d); err != nil {
		fmt.Fprintf(d.Err, "ship test failed: %v\n", err)
		return 1
	}
	return 0
}

func runDefaultTestSuite(d QualityDeps) error {
	if d.HasFile == nil {
		d.HasFile = fileExists
	}
	root, hasLists := findProjectRootWithCheckLists(d.HasFile)
	if hasLists {
		return withWorkingDir(root, func() error {
			unitPkgs, err := readPackageList(filepath.Join("scripts", "test", "unit-packages.txt"))
			if err != nil {
				return err
			}
			for _, pkg := range unitPkgs {
				if code := d.RunCmd("go", "test", pkg); code != 0 {
					return fmt.Errorf("go test %s failed with exit code %d", pkg, code)
				}
			}

			compilePkgs, err := readPackageList(filepath.Join("scripts", "test", "compile-packages.txt"))
			if err != nil {
				return err
			}
			for _, pkg := range compilePkgs {
				if code := d.RunCmd("go", "test", "-run", "^$", pkg); code != 0 {
					return fmt.Errorf("compile check for %s failed with exit code %d", pkg, code)
				}
			}
			return nil
		})
	}
	if code := d.RunCmd("go", "test", "./..."); code != 0 {
		return fmt.Errorf("go test ./... failed with exit code %d", code)
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func findProjectRootWithCheckLists(hasFile func(path string) bool) (string, bool) {
	wd, err := os.Getwd()
	if err != nil {
		return "", false
	}
	dir := wd
	for {
		unitPath := filepath.Join(dir, "scripts", "test", "unit-packages.txt")
		compilePath := filepath.Join(dir, "scripts", "test", "compile-packages.txt")
		if hasFile(unitPath) && hasFile(compilePath) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func readPackageList(path string) ([]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(b), "\n")
	pkgs := make([]string, 0, len(lines))
	for _, line := range lines {
		s := strings.TrimSpace(line)
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		pkgs = append(pkgs, s)
	}
	return pkgs, nil
}

func withWorkingDir(dir string, fn func() error) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := os.Chdir(dir); err != nil {
		return err
	}
	defer func() { _ = os.Chdir(wd) }()
	return fn()
}

func PrintTestHelp(w io.Writer) {
	fmt.Fprintln(w, "ship test commands:")
	fmt.Fprintln(w, "  ship test                 Run canonical fast unit/compile suite")
	fmt.Fprintln(w, "  ship test --integration   Include integration-tagged tests")
}
