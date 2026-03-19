package commands

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/backup"
)

func TestDBExportContract_DefinesManifestChecksumHook_RedSpec(t *testing.T) {
	t.Run("promotion report points at export hook", func(t *testing.T) {
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
			"next: ship db:export --json",
			"next: ship db:import --json",
		} {
			if !strings.Contains(out.String(), token) {
				t.Fatalf("stdout missing %q:\n%s", token, out.String())
			}
		}
	})

	t.Run("db help exposes export hook", func(t *testing.T) {
		out := captureHelp(t, PrintDBHelp)

		if !strings.Contains(out, "  ship db:export [--json]") {
			t.Fatalf("db help missing export hook:\n%s", out)
		}
	})

	t.Run("db export emits checksum manifest", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "main.db")
		dbBytes := []byte("sqlite-export-content")
		if err := os.WriteFile(dbPath, dbBytes, 0o644); err != nil {
			t.Fatalf("write sqlite file: %v", err)
		}

		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunDB([]string{"export", "--json"}, DBDeps{
			Out: out,
			Err: errOut,
			LoadConfig: func() (config.Config, error) {
				cfg := config.Config{}
				cfg.Backup.SchemaVersion = "20260318_001"
				cfg.Backup.SQLitePath = dbPath
				cfg.Database.Path = dbPath
				cfg.Database.DbMode = config.DBModeEmbedded
				cfg.Database.Driver = config.DBDriverSQLite
				return cfg, nil
			},
		})
		if code != 0 {
			t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
		}

		var payload struct {
			Manifest          backup.Manifest `json:"manifest"`
			SuggestedCommands []string        `json:"suggested_commands"`
		}
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("decode json: %v\n%s", err, out.String())
		}

		sum := sha256.Sum256(dbBytes)
		if payload.Manifest.Version != backup.ManifestVersionV1 {
			t.Fatalf("version = %q", payload.Manifest.Version)
		}
		if payload.Manifest.Database.SchemaVersion != "20260318_001" {
			t.Fatalf("schema_version = %q", payload.Manifest.Database.SchemaVersion)
		}
		if payload.Manifest.Database.SourcePath != dbPath {
			t.Fatalf("source_path = %q", payload.Manifest.Database.SourcePath)
		}
		if payload.Manifest.Artifact.ChecksumSHA256 != hex.EncodeToString(sum[:]) {
			t.Fatalf("checksum = %q", payload.Manifest.Artifact.ChecksumSHA256)
		}
		if !containsString(payload.SuggestedCommands, "ship db:import --json") {
			t.Fatalf("expected export to suggest db:import:\n%s", out.String())
		}
	})

	t.Run("db export text output highlights checksum contract", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "main.db")
		dbBytes := []byte("sqlite-export-content")
		if err := os.WriteFile(dbPath, dbBytes, 0o644); err != nil {
			t.Fatalf("write sqlite file: %v", err)
		}

		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		code := RunDB([]string{"export"}, DBDeps{
			Out: out,
			Err: errOut,
			LoadConfig: func() (config.Config, error) {
				cfg := config.Config{}
				cfg.Backup.SchemaVersion = "20260318_001"
				cfg.Backup.SQLitePath = dbPath
				cfg.Database.Path = dbPath
				cfg.Database.DbMode = config.DBModeEmbedded
				cfg.Database.Driver = config.DBDriverSQLite
				return cfg, nil
			},
		})
		if code != 0 {
			t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
		}

		sum := sha256.Sum256(dbBytes)
		for _, token := range []string{
			"DB export manifest:",
			"- version: backup-manifest-v1",
			"- schema_version: 20260318_001",
			"- source_path: " + dbPath,
			"- checksum_sha256: " + hex.EncodeToString(sum[:]),
			"- next: ship db:import --json",
			"- note: planning only; db:export reports manifest checksums and does not mutate runtime state yet",
		} {
			if !strings.Contains(out.String(), token) {
				t.Fatalf("stdout missing %q:\n%s", token, out.String())
			}
		}
	})
}
