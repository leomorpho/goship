package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunUpgrade_JSONReadinessReport_RedSpec(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	cliPath := filepath.Join(root, "tools", "cli", "ship", "internal", "cli")
	if err := os.MkdirAll(cliPath, 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/demo\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(go.mod) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(cliPath, "cli.go"), []byte("package cli\n\nconst gooseGoRunRef = \"github.com/pressly/goose/v3/cmd/goose@v3.26.0\"\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(cli.go) error = %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("os.Chdir(root) error = %v", err)
	}

	var out bytes.Buffer
	code := RunUpgrade([]string{"--to", "v3.27.0", "--json"}, UpgradeDeps{
		Out:          &out,
		Err:          &out,
		FindGoModule: func(string) (string, string, error) { return root, "", nil },
	})
	if code != 0 {
		t.Fatalf("RunUpgrade() exit code = %d\n%s", code, out.String())
	}
	assertContains(t, "upgrade json", out.String(), "\"schema_version\":\"upgrade-readiness-v1\"")
	assertContains(t, "upgrade json", out.String(), "\"target_version\":\"v3.27.0\"")
}

func TestRunUpgrade_RejectsUnsupportedContractVersion_RedSpec(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	code := RunUpgrade([]string{"--to", "v3.27.0", "--contract-version", "upgrade-readiness-v999"}, UpgradeDeps{
		Out:          &out,
		Err:          &out,
		FindGoModule: func(string) (string, string, error) { return ".", "", nil },
	})
	if code == 0 {
		t.Fatalf("RunUpgrade() exit code = 0, want non-zero\n%s", out.String())
	}
	if !strings.Contains(out.String(), "unsupported upgrade contract version") {
		t.Fatalf("unexpected output\n%s", out.String())
	}
}

func TestRunUpgradeApply_RewritesGoosePinFixture(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	cliPath := filepath.Join(root, "tools", "cli", "ship", "internal", "cli")
	if err := os.MkdirAll(cliPath, 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/demo\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(go.mod) error = %v", err)
	}
	before := "package cli\n\nconst gooseGoRunRef = \"github.com/pressly/goose/v3/cmd/goose@v3.26.0\"\n"
	cliFile := filepath.Join(cliPath, "cli.go")
	if err := os.WriteFile(cliFile, []byte(before), 0o644); err != nil {
		t.Fatalf("os.WriteFile(cli.go) error = %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("os.Chdir(root) error = %v", err)
	}

	var out bytes.Buffer
	code := RunUpgrade([]string{"apply", "--to", "v3.27.0"}, UpgradeDeps{
		Out:          &out,
		Err:          &out,
		FindGoModule: func(string) (string, string, error) { return root, "", nil },
	})
	if code != 0 {
		t.Fatalf("RunUpgrade() exit code = %d\n%s", code, out.String())
	}
	after, err := os.ReadFile(cliFile)
	if err != nil {
		t.Fatalf("os.ReadFile(cli.go) error = %v", err)
	}
	if !strings.Contains(string(after), "v3.27.0") {
		t.Fatalf("cli.go did not update\n%s", after)
	}
}
