package migrate

import (
	"strings"
	"testing"
)

func TestLoadInitNotificationsUpSQL(t *testing.T) {
	t.Parallel()

	sql, err := LoadInitNotificationsUpSQL()
	if err != nil {
		t.Fatalf("LoadInitNotificationsUpSQL returned error: %v", err)
	}
	if !strings.Contains(sql, "CREATE TABLE IF NOT EXISTS notifications") {
		t.Fatalf("expected notifications table DDL in up sql")
	}
	if strings.Contains(sql, "-- +goose Down") {
		t.Fatalf("up sql should not contain goose down section")
	}
}
