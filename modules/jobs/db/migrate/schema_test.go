package migrate

import (
	"strings"
	"testing"
)

func TestLoadInitJobsUpSQL(t *testing.T) {
	t.Parallel()

	sql, err := LoadInitJobsUpSQL()
	if err != nil {
		t.Fatalf("LoadInitJobsUpSQL returned error: %v", err)
	}
	if !strings.Contains(sql, "CREATE TABLE IF NOT EXISTS goship_jobs") {
		t.Fatalf("expected jobs table DDL in up sql, got: %s", sql)
	}
	if strings.Contains(sql, "-- +goose Down") {
		t.Fatalf("up sql should not contain goose down section")
	}
}
