package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leomorpho/goship/config"
)

func TestDBImportContract_DefinesImportAndVerificationHooks_RedSpec(t *testing.T) {
	t.Run("promotion report points at import hooks", func(t *testing.T) {
		root := t.TempDir()
		envPath := filepath.Join(root, ".env")
		if err := os.WriteFile(envPath, []byte("PAGODA_RUNTIME_PROFILE=single-node\n"), 0o644); err != nil {
			t.Fatalf("write env: %v", err)
		}
		prevWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("getwd: %v", err)
		}
		if err := os.Chdir(root); err != nil {
			t.Fatalf("chdir %s: %v", root, err)
		}
		t.Cleanup(func() { _ = os.Chdir(prevWD) })

		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}

		code := RunDB([]string{"promote"}, DBDeps{
			Out: out,
			Err: errOut,
			LoadConfig: func() (config.Config, error) {
				cfg := config.Config{}
				cfg.Database.DbMode = config.DBModeEmbedded
				cfg.Database.Driver = config.DBDriverSQLite
				return cfg, nil
			},
		})
		if code != 0 {
			t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
		}

		for _, token := range []string{
			"next: ship db:import --json",
			"next: ship db:verify-import --json",
		} {
			if !strings.Contains(out.String(), token) {
				t.Fatalf("stdout missing %q:\n%s", token, out.String())
			}
		}
	})

	t.Run("db help exposes import hooks", func(t *testing.T) {
		out := captureHelp(t, PrintDBHelp)

		for _, token := range []string{
			"  ship db:import [--json]",
			"  ship db:verify-import [--json]",
		} {
			if !strings.Contains(out, token) {
				t.Fatalf("db help missing %q:\n%s", token, out)
			}
		}
	})

	t.Run("db import reports an import plan", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}

		code := RunDB([]string{"import", "--json"}, DBDeps{
			Out: out,
			Err: errOut,
			LoadConfig: func() (config.Config, error) {
				cfg := config.Config{}
				cfg.Database.DbMode = config.DBModeEmbedded
				cfg.Database.Driver = config.DBDriverSQLite
				return cfg, nil
			},
		})
		if code != 0 {
			t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
		}

		var payload struct {
			Database struct {
				PromotionPath string `json:"promotion_path"`
			} `json:"database"`
			Steps             []string `json:"steps"`
			SuggestedCommands []string `json:"suggested_commands"`
		}
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("decode json: %v\n%s", err, out.String())
		}
		for _, token := range []string{
			"load export manifest and validate version, driver, and checksums",
			"import exported data into Postgres through framework-supported import hooks",
			"ship db:verify-import --json",
		} {
			if !containsString(payload.Steps, token) && !containsString(payload.SuggestedCommands, token) {
				t.Fatalf("import plan missing %q:\n%s", token, out.String())
			}
		}
	})

	t.Run("db import text output highlights manifest validation path", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}

		code := RunDB([]string{"import"}, DBDeps{
			Out: out,
			Err: errOut,
			LoadConfig: func() (config.Config, error) {
				cfg := config.Config{}
				cfg.Database.DbMode = config.DBModeEmbedded
				cfg.Database.Driver = config.DBDriverSQLite
				return cfg, nil
			},
		})
		if code != 0 {
			t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
		}

		for _, token := range []string{
			"DB import plan:",
			"- step: load export manifest and validate version, driver, and checksums",
			"- step: import exported data into Postgres through framework-supported import hooks",
			"- next: ship db:verify-import --json",
			"- note: planning only; db:import does not mutate files or import data yet",
		} {
			if !strings.Contains(out.String(), token) {
				t.Fatalf("stdout missing %q:\n%s", token, out.String())
			}
		}
	})

	t.Run("db verify-import reports post-import checks", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}

		code := RunDB([]string{"verify-import", "--json"}, DBDeps{
			Out: out,
			Err: errOut,
			LoadConfig: func() (config.Config, error) {
				cfg := config.Config{}
				cfg.Database.DbMode = config.DBModeEmbedded
				cfg.Database.Driver = config.DBDriverSQLite
				return cfg, nil
			},
		})
		if code != 0 {
			t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
		}

		var payload struct {
			PostImportChecks []string `json:"post_import_checks"`
		}
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("decode json: %v\n%s", err, out.String())
		}
		for _, token := range []string{
			"manifest.validated",
			"row.counts.checked",
			"migration.baseline.checked",
			"key.integrity.checked",
		} {
			if !containsString(payload.PostImportChecks, token) {
				t.Fatalf("verify-import checks missing %q:\n%s", token, out.String())
			}
		}
	})

	t.Run("db verify-import text output highlights post-import checks", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}

		code := RunDB([]string{"verify-import"}, DBDeps{
			Out: out,
			Err: errOut,
			LoadConfig: func() (config.Config, error) {
				cfg := config.Config{}
				cfg.Database.DbMode = config.DBModeEmbedded
				cfg.Database.Driver = config.DBDriverSQLite
				return cfg, nil
			},
		})
		if code != 0 {
			t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
		}

		for _, token := range []string{
			"DB verify-import plan:",
			"- check: manifest.validated",
			"- check: row.counts.checked",
			"- check: migration.baseline.checked",
			"- check: key.integrity.checked",
			"- note: planning only; db:verify-import does not mutate files or databases yet",
		} {
			if !strings.Contains(out.String(), token) {
				t.Fatalf("stdout missing %q:\n%s", token, out.String())
			}
		}
	})
}
