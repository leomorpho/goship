package notifications

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

type noopSub struct{}

func (noopSub) Close() error { return nil }

type noopPubSub struct{}

func (noopPubSub) Publish(context.Context, string, []byte) error { return nil }
func (noopPubSub) Subscribe(context.Context, string, MessageHandler) (PubSubSubscription, error) {
	return noopSub{}, nil
}
func (noopPubSub) Close() error { return nil }

func TestModuleNew_SQLMode_DoesNotRequireORM(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	services, err := New(RuntimeDeps{
		DB:                                  db,
		DBDialect:                           "sqlite3",
		PubSub:                              noopPubSub{},
		SMSRegion:                           "us-east-1",
		SMSValidationCodeExpirationMinutes:  10,
		GetNumNotificationsForProfileByIDFn: func(context.Context, int) (int, error) { return 0, nil },
	})
	if err != nil {
		t.Fatalf("New(sql mode) error: %v", err)
	}
	if services == nil || services.Notifier == nil || services.Permission == nil || services.PwaPush == nil || services.FcmPush == nil || services.SMSSender == nil || services.PlannedNotificationsService == nil {
		t.Fatalf("New(sql mode) returned incomplete services: %#v", services)
	}
}

func TestModuleNew_RequiresDB(t *testing.T) {
	t.Parallel()

	_, err := New(RuntimeDeps{
		DB:                                  nil,
		DBDialect:                           "",
		PubSub:                              noopPubSub{},
		SMSRegion:                           "us-east-1",
		SMSValidationCodeExpirationMinutes:  10,
		GetNumNotificationsForProfileByIDFn: func(context.Context, int) (int, error) { return 0, nil },
	})
	if err == nil {
		t.Fatal("expected error when DB is nil")
	}
}
