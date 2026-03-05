package emailsubscriptions

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestSQLStore_SubscribeConfirmUnsubscribe(t *testing.T) {
	db := openTestDB(t)
	store := NewSQLStore(db, "sqlite3")
	ctx := context.Background()

	list := List("newsletter")
	require.NoError(t, store.CreateList(ctx, list))
	require.NoError(t, store.CreateList(ctx, list))

	sub, err := store.Subscribe(ctx, "alice@example.com", list, nil, nil)
	require.NoError(t, err)
	require.NotZero(t, sub.ID)
	require.NotEmpty(t, sub.ConfirmationCode)
	require.False(t, sub.Verified)

	require.NoError(t, store.Confirm(ctx, sub.ConfirmationCode))

	_, err = store.Subscribe(ctx, "alice@example.com", list, nil, nil)
	var alreadySubscribedErr *ErrAlreadySubscribed
	require.ErrorAs(t, err, &alreadySubscribedErr)

	require.NoError(t, store.Unsubscribe(ctx, "alice@example.com", "", list))

	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM email_subscriptions WHERE email = ?`, "alice@example.com").Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestSQLStore_ConfirmInvalidCode(t *testing.T) {
	db := openTestDB(t)
	store := NewSQLStore(db, "sqlite3")
	ctx := context.Background()

	err := store.Confirm(ctx, "missing-code")
	require.ErrorIs(t, err, ErrInvalidEmailConfirmationCode)
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`
		CREATE TABLE email_subscription_types (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			name TEXT NOT NULL UNIQUE,
			active BOOLEAN NOT NULL
		);
		CREATE TABLE email_subscriptions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			email TEXT NOT NULL UNIQUE,
			verified BOOLEAN NOT NULL DEFAULT 0,
			confirmation_code TEXT NOT NULL UNIQUE,
			latitude REAL NULL,
			longitude REAL NULL
		);
		CREATE TABLE email_subscription_subscriptions (
			email_subscription_id INTEGER NOT NULL,
			email_subscription_type_id INTEGER NOT NULL,
			PRIMARY KEY (email_subscription_id, email_subscription_type_id),
			FOREIGN KEY (email_subscription_id) REFERENCES email_subscriptions(id) ON DELETE CASCADE,
			FOREIGN KEY (email_subscription_type_id) REFERENCES email_subscription_types(id) ON DELETE CASCADE
		);
	`)
	require.NoError(t, err)

	return db
}
