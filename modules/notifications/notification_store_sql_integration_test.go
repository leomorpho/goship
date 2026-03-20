//go:build integration

package notifications

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	dbmigrate "github.com/leomorpho/goship-modules/notifications/db/migrate"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"

	"github.com/leomorpho/goship/framework/domain"
)

func TestSQLNotificationStore_WithModuleMigration_EndToEnd(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "notifications-integration.db")
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	applyNotificationsModuleMigration(t, db)

	ctx := context.Background()
	store := NewSQLNotificationStore(db, "sqlite3")

	n := domain.Notification{
		Type:                      domain.NotificationTypePlatformUpdate,
		Title:                     "Platform update",
		Text:                      "New feature",
		ProfileID:                 77,
		ProfileIDWhoCausedNotif:   77,
		ResourceIDTiedToNotif:     1234,
		ReadInNotificationsCenter: true,
	}
	created, err := store.CreateNotification(ctx, n)
	require.NoError(t, err)
	require.NotZero(t, created.ID)

	list, err := store.GetNotificationsByProfileID(ctx, 77, false, nil, nil)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, domain.NotificationTypePlatformUpdate, list[0].Type)

	ok, err := store.HasNotificationForResourceAndPerson(
		ctx,
		domain.NotificationTypePlatformUpdate,
		&n.ProfileIDWhoCausedNotif,
		&n.ResourceIDTiedToNotif,
		time.Hour,
	)
	require.NoError(t, err)
	require.True(t, ok)

	require.NoError(t, store.MarkNotificationAsRead(ctx, created.ID, &created.ProfileID))
	readOnly, err := store.GetNotificationsByProfileID(ctx, 77, true, nil, nil)
	require.NoError(t, err)
	require.Len(t, readOnly, 0)

	require.NoError(t, store.MarkNotificationAsUnread(ctx, created.ID, &created.ProfileID))
	unreadOnly, err := store.GetNotificationsByProfileID(ctx, 77, true, nil, nil)
	require.NoError(t, err)
	require.Len(t, unreadOnly, 1)

	require.NoError(t, store.MarkAllNotificationAsRead(ctx, 77))
	unreadOnly, err = store.GetNotificationsByProfileID(ctx, 77, true, nil, nil)
	require.NoError(t, err)
	require.Len(t, unreadOnly, 0)

	require.NoError(t, store.DeleteNotification(ctx, created.ID, &created.ProfileID))
	list, err = store.GetNotificationsByProfileID(ctx, 77, false, nil, nil)
	require.NoError(t, err)
	require.Len(t, list, 0)
}

func applyNotificationsModuleMigration(t *testing.T, db *sql.DB) {
	t.Helper()

	upSQL, err := dbmigrate.LoadInitNotificationsUpSQL()
	require.NoError(t, err)

	upSQL = sqliteCompatibleNotificationsDDL(upSQL)
	_, err = db.Exec(upSQL)
	require.NoError(t, err)
}

func sqliteCompatibleNotificationsDDL(v string) string {
	replacements := []struct {
		old string
		new string
	}{
		{"BIGSERIAL PRIMARY KEY", "INTEGER PRIMARY KEY AUTOINCREMENT"},
		{"TIMESTAMPTZ", "DATETIME"},
	}
	out := v
	for _, replacement := range replacements {
		out = strings.ReplaceAll(out, replacement.old, replacement.new)
	}
	return out
}
