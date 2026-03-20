package auditlog

import (
	"context"
	"testing"
)

type memoryStore struct {
	inserted []Log
	listed   []Log
}

func (m *memoryStore) Insert(_ context.Context, entry Log) error {
	m.inserted = append(m.inserted, entry)
	return nil
}

func (m *memoryStore) List(_ context.Context, _ ListFilters) ([]Log, error) {
	return m.listed, nil
}

func TestServiceRecordUsesContextMetadata(t *testing.T) {
	store := &memoryStore{}
	service := NewService(store)
	userID := int64(42)

	err := service.Record(
		WithRequestMetadata(context.Background(), &userID, "203.0.113.8", "GoShipTest/1.0"),
		"user.login",
		"user",
		"42",
		map[string]any{"source": "password"},
	)
	if err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	if len(store.inserted) != 1 {
		t.Fatalf("inserted logs = %d, want 1", len(store.inserted))
	}
	entry := store.inserted[0]
	if entry.UserID == nil || *entry.UserID != 42 {
		t.Fatalf("UserID = %v, want 42", entry.UserID)
	}
	if entry.IPAddress != "203.0.113.8" {
		t.Fatalf("IPAddress = %q, want 203.0.113.8", entry.IPAddress)
	}
	if entry.UserAgent != "GoShipTest/1.0" {
		t.Fatalf("UserAgent = %q, want GoShipTest/1.0", entry.UserAgent)
	}
	if entry.Changes != `{"source":"password"}` {
		t.Fatalf("Changes = %q", entry.Changes)
	}
}
