package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/leomorpho/goship/config"
)

func TestDBImportContract_DefinesImportAndVerificationHooks_RedSpec(t *testing.T) {
	t.Run("promotion report points at import hooks", func(t *testing.T) {
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
}
