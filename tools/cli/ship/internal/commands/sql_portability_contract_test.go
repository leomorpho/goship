package commands

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSQLPortabilityContract_ReportsMissingDialectCompanion_RedSpec(t *testing.T) {
	root := t.TempDir()
	repoRoot := repoRootFromCommandsTest(t)

	queriesDir := filepath.Join(root, "framework", "repos", "storage", "queries")
	if err := os.MkdirAll(queriesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(queriesDir, "storage.sql"), []byte(`-- name: delete_file_storage_by_object_key_sqlite
DELETE FROM file_storages
WHERE object_key = ?;
`), 0o644); err != nil {
		t.Fatal(err)
	}

	out := runSQLPortabilityScript(t, repoRoot, root)

	for _, token := range []string{
		"branch handling required",
		"query=delete_file_storage_by_object_key_sqlite",
		"file=framework/repos/storage/queries/storage.sql",
	} {
		if !strings.Contains(out, token) {
			t.Fatalf("portability output missing %q:\n%s", token, out)
		}
	}
}

func TestSQLPortabilityContract_RequiresExplicitBranchHandlingForPostgresOnlyMigration_RedSpec(t *testing.T) {
	root := t.TempDir()
	repoRoot := repoRootFromCommandsTest(t)

	queriesDir := filepath.Join(root, "db", "queries")
	if err := os.MkdirAll(queriesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(queriesDir, "migrations.sql"), []byte(`-- name: create_database_postgres
CREATE DATABASE 
`), 0o644); err != nil {
		t.Fatal(err)
	}

	out := runSQLPortabilityScript(t, repoRoot, root)

	for _, token := range []string{
		"branch handling required",
		"query=create_database_postgres",
		"file=db/queries/migrations.sql",
	} {
		if !strings.Contains(out, token) {
			t.Fatalf("portability output missing %q:\n%s", token, out)
		}
	}
}

func runSQLPortabilityScript(t *testing.T, repoRoot, root string) string {
	t.Helper()

	script := filepath.Join(repoRoot, "tools", "scripts", "check-sql-portability.sh")
	cmd := exec.Command("bash", script)
	cmd.Env = append(os.Environ(),
		"ROOT_DIR="+root,
		"SQL_PORTABILITY_SKIP_CONFIG=1",
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err == nil {
		t.Fatal("expected sql portability script to fail for a red-spec fixture")
	}
	return out.String()
}
