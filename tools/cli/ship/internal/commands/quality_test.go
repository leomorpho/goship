package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRunTest_UsesCuratedPackageListsFromCanonicalPath(t *testing.T) {
	root := t.TempDir()
	writeQualityFile(t, filepath.Join(root, "tools", "scripts", "test", "unit-packages.txt"), "./pkg/a\n./pkg/b\n")
	writeQualityFile(t, filepath.Join(root, "tools", "scripts", "test", "compile-packages.txt"), "./pkg/c\n")

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	calls := make([][]string, 0)
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := RunTest([]string{}, QualityDeps{
		Out: out,
		Err: errOut,
		RunCmd: func(name string, args ...string) int {
			call := append([]string{name}, args...)
			calls = append(calls, call)
			return 0
		},
	})
	if code != 0 {
		t.Fatalf("code = %d, want 0; stderr=%s", code, errOut.String())
	}

	wantCalls := [][]string{
		{"go", "test", "./pkg/a"},
		{"go", "test", "./pkg/b"},
		{"go", "test", "-run", "^$", "./pkg/c"},
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("calls = %#v, want %#v", calls, wantCalls)
	}
}

func TestRunTest_LegacyScriptsPathIsIgnoredAndFallsBackToGoTestAll(t *testing.T) {
	root := t.TempDir()
	writeQualityFile(t, filepath.Join(root, "scripts", "test", "unit-packages.txt"), "./pkg/legacy\n")
	writeQualityFile(t, filepath.Join(root, "scripts", "test", "compile-packages.txt"), "./pkg/legacy\n")

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	calls := make([][]string, 0)
	code := RunTest([]string{}, QualityDeps{
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
		RunCmd: func(name string, args ...string) int {
			call := append([]string{name}, args...)
			calls = append(calls, call)
			return 0
		},
	})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}

	wantCalls := [][]string{{"go", "test", "./..."}}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("calls = %#v, want %#v", calls, wantCalls)
	}
}

func writeQualityFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
