package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCLIGoldenContractSuite_RedSpec(t *testing.T) {
	packageDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	t.Run("routes table and json", func(t *testing.T) {
		root := t.TempDir()
		writeDescribeFixture(t, root)

		prevWD := chdirCommandsFixture(t, root)
		t.Cleanup(func() { _ = os.Chdir(prevWD) })

		tableOut := &bytes.Buffer{}
		if code := RunRoutes([]string{}, RoutesDeps{Out: tableOut, Err: &bytes.Buffer{}, FindGoModule: findDescribeGoModule}); code != 0 {
			t.Fatalf("routes table exit code = %d", code)
		}
		assertCLIGoldenSnapshot(t, packageDir, "routes_table.golden", tableOut.String())

		jsonOut := &bytes.Buffer{}
		if code := RunRoutes([]string{"--json"}, RoutesDeps{Out: jsonOut, Err: &bytes.Buffer{}, FindGoModule: findDescribeGoModule}); code != 0 {
			t.Fatalf("routes json exit code = %d", code)
		}
		assertCLIJSONGolden(t, packageDir, "routes_json.golden", jsonOut.Bytes())
	})

	t.Run("describe pretty", func(t *testing.T) {
		root := t.TempDir()
		writeDescribeFixture(t, root)

		prevWD := chdirCommandsFixture(t, root)
		t.Cleanup(func() { _ = os.Chdir(prevWD) })

		out := &bytes.Buffer{}
		if code := RunDescribe([]string{"--pretty"}, DescribeDeps{Out: out, Err: &bytes.Buffer{}, FindGoModule: findDescribeGoModule}); code != 0 {
			t.Fatalf("describe exit code = %d", code)
		}
		assertCLIJSONGolden(t, packageDir, "describe_pretty.golden", out.Bytes())
	})

	t.Run("verify human and json", func(t *testing.T) {
		root := t.TempDir()
		writeVerifyGoMod(t, root)

		prevWD := chdirCommandsFixture(t, root)
		t.Cleanup(func() { _ = os.Chdir(prevWD) })

		humanOut := &bytes.Buffer{}
		if code := RunVerify([]string{"--skip-tests"}, VerifyDeps{
			Out:          humanOut,
			Err:          &bytes.Buffer{},
			FindGoModule: findVerifyGoModule,
			RelocateTempl: func(string) error {
				return nil
			},
			RunStep: func(string, ...string) (int, string, error) {
				return 0, "ok", nil
			},
			LookPath: func(file string) (string, error) {
				return "/usr/bin/" + file, nil
			},
			RunDoctor: func() (int, string, error) {
				return 0, `{"ok":true,"issues":[]}`, nil
			},
		}); code != 0 {
			t.Fatalf("verify human exit code = %d", code)
		}
		assertCLIGoldenSnapshot(t, packageDir, "verify_human.golden", humanOut.String())

		jsonOut := &bytes.Buffer{}
		if code := RunVerify([]string{"--json"}, VerifyDeps{
			Out:          jsonOut,
			Err:          &bytes.Buffer{},
			FindGoModule: findVerifyGoModule,
			RelocateTempl: func(string) error {
				return nil
			},
			RunStep: func(name string, args ...string) (int, string, error) {
				return 0, name + " ok", nil
			},
			LookPath: func(file string) (string, error) {
				return "/usr/bin/" + file, nil
			},
			RunDoctor: func() (int, string, error) {
				return 0, `{"ok":true,"issues":[]}`, nil
			},
		}); code != 0 {
			t.Fatalf("verify json exit code = %d", code)
		}
		assertCLIJSONGolden(t, packageDir, "verify_json.golden", jsonOut.Bytes())
	})
}

func chdirCommandsFixture(t *testing.T, root string) string {
	t.Helper()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir %s: %v", root, err)
	}
	return prevWD
}

func assertCLIGoldenSnapshot(t *testing.T, packageDir, name, got string) {
	t.Helper()

	path := filepath.Join(packageDir, "testdata", name)
	if os.Getenv("UPDATE_CLI_GOLDENS") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("write snapshot %s: %v", path, err)
		}
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read snapshot %s: %v", path, err)
	}
	if string(want) != got {
		t.Fatalf("cli golden drift for %s", path)
	}
}

func assertCLIJSONGolden(t *testing.T, packageDir, name string, payload []byte) {
	t.Helper()

	var pretty bytes.Buffer
	if err := json.Indent(&pretty, payload, "", "  "); err != nil {
		t.Fatalf("indent json: %v", err)
	}
	pretty.WriteByte('\n')
	assertCLIGoldenSnapshot(t, packageDir, name, pretty.String())
}
