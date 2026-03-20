package auditlog

import (
	"context"
	"testing"

	"github.com/leomorpho/goship/framework/events"
	eventtypes "github.com/leomorpho/goship/framework/events/types"
)

func TestSubscribeRecordsUserLoggedInEvents(t *testing.T) {
	store := &memoryStore{}
	service := NewService(store)
	bus := events.NewBus()

	Subscribe(bus, service)

	if err := bus.Publish(context.Background(), eventtypes.UserLoggedIn{UserID: 7, IP: "198.51.100.10"}); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	if len(store.inserted) != 1 {
		t.Fatalf("inserted logs = %d, want 1", len(store.inserted))
	}
	entry := store.inserted[0]
	if entry.Action != "user.login" {
		t.Fatalf("Action = %q, want user.login", entry.Action)
	}
	if entry.ResourceType != "user" || entry.ResourceID != "7" {
		t.Fatalf("resource = %s/%s, want user/7", entry.ResourceType, entry.ResourceID)
	}
	if entry.IPAddress != "198.51.100.10" {
		t.Fatalf("IPAddress = %q, want 198.51.100.10", entry.IPAddress)
	}
}
