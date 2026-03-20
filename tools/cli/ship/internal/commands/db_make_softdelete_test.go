package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDBMakeSoftDelete_GeneratesMigrationFile(t *testing.T) {
	dir := t.TempDir()
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	code := RunDB([]string{"make", "add_soft_delete_to_orders", "--soft-delete", "--table", "orders"}, DBDeps{
		Out:      out,
		Err:      errOut,
		GooseDir: dir,
		RunGoose: func(args ...string) int {
			t.Fatalf("RunGoose should not be called, got %v", args)
			return 1
		},
	})
	if code != 0 {
		t.Fatalf("code = %d, want 0 (stderr=%q)", code, errOut.String())
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 migration file, got %d", len(entries))
	}

	content, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "ALTER TABLE orders ADD COLUMN deleted_at DATETIME;") {
		t.Fatalf("migration missing deleted_at column:\n%s", text)
	}
	if !strings.Contains(text, "CREATE INDEX idx_orders_deleted_at ON orders(deleted_at);") {
		t.Fatalf("migration missing deleted_at index:\n%s", text)
	}
}

func TestRunDBMakeSoftDelete_RequiresTable(t *testing.T) {
	errOut := &bytes.Buffer{}
	code := RunDB([]string{"make", "add_soft_delete_to_orders", "--soft-delete"}, DBDeps{
		Out:      &bytes.Buffer{},
		Err:      errOut,
		GooseDir: t.TempDir(),
		RunGoose: func(args ...string) int { return 0 },
	})
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "requires --table") {
		t.Fatalf("stderr = %q, want missing table error", errOut.String())
	}
}
