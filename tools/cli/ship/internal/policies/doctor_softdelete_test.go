package policies

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunDoctorChecks_SoftDeleteQueryFilter(t *testing.T) {
	t.Run("warns when soft-delete table query omits deleted_at filter", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)

		migration := `CREATE TABLE orders (
id INTEGER PRIMARY KEY,
deleted_at DATETIME
);`
		if err := os.WriteFile(filepath.Join(root, "db", "migrate", "migrations", "20260316000100_orders.sql"), []byte(migration), 0o644); err != nil {
			t.Fatal(err)
		}

		query := `-- name: orders_list :many
SELECT id FROM orders ORDER BY id DESC;`
		if err := os.WriteFile(filepath.Join(root, "db", "queries", "orders.sql"), []byte(query), 0o644); err != nil {
			t.Fatal(err)
		}

		tables := discoverSoftDeleteTables(root)
		if _, ok := tables["orders"]; !ok {
			t.Fatalf("expected orders table to be detected, got %#v", tables)
		}

		issues := RunDoctorChecks(root)
		mustContainIssueCode(t, issues, "DX028")
	})

	t.Run("does not warn when query filters deleted_at IS NULL", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)

		migration := `ALTER TABLE orders ADD COLUMN deleted_at DATETIME;`
		if err := os.WriteFile(filepath.Join(root, "db", "migrate", "migrations", "20260316000101_orders.sql"), []byte(migration), 0o644); err != nil {
			t.Fatal(err)
		}

		query := `-- name: orders_list_active :many
SELECT id FROM orders WHERE deleted_at IS NULL ORDER BY id DESC;`
		if err := os.WriteFile(filepath.Join(root, "db", "queries", "orders.sql"), []byte(query), 0o644); err != nil {
			t.Fatal(err)
		}

		issues := RunDoctorChecks(root)
		for _, issue := range issues {
			if issue.Code == "DX028" {
				t.Fatalf("unexpected DX028 issue: %+v", issue)
			}
		}
	})

	t.Run("does not warn when query explicitly requests deleted rows", func(t *testing.T) {
		root := t.TempDir()
		writeDoctorFixture(t, root)

		migration := `ALTER TABLE orders ADD COLUMN deleted_at DATETIME;`
		if err := os.WriteFile(filepath.Join(root, "db", "migrate", "migrations", "20260316000102_orders.sql"), []byte(migration), 0o644); err != nil {
			t.Fatal(err)
		}

		query := `-- name: orders_list_deleted :many
SELECT id FROM orders WHERE deleted_at IS NOT NULL ORDER BY id DESC;`
		if err := os.WriteFile(filepath.Join(root, "db", "queries", "orders.sql"), []byte(query), 0o644); err != nil {
			t.Fatal(err)
		}

		issues := RunDoctorChecks(root)
		for _, issue := range issues {
			if issue.Code == "DX028" {
				t.Fatalf("unexpected DX028 issue: %+v", issue)
			}
		}
	})
}
