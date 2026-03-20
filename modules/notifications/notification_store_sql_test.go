package notifications

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"

	"github.com/leomorpho/goship/framework/domain"
)

func TestSQLNotificationStore_Lifecycle(t *testing.T) {
	store := openNotificationStoreTestDB(t)
	ctx := context.Background()

	n := domain.Notification{
		Type:                      domain.NotificationTypePlatformUpdate,
		Title:                     "Welcome",
		Text:                      "Hello world",
		ProfileID:                 42,
		ProfileIDWhoCausedNotif:   42,
		ResourceIDTiedToNotif:     1001,
		ReadInNotificationsCenter: true,
	}
	created, err := store.CreateNotification(ctx, n)
	require.NoError(t, err)
	require.NotZero(t, created.ID)

	list, err := store.GetNotificationsByProfileID(ctx, 42, false, nil, nil)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "Welcome", list[0].Title)

	exists, err := store.HasNotificationForResourceAndPerson(
		ctx, domain.NotificationTypePlatformUpdate, &n.ProfileIDWhoCausedNotif, &n.ResourceIDTiedToNotif, 24*time.Hour,
	)
	require.NoError(t, err)
	require.True(t, exists)

	require.NoError(t, store.MarkNotificationAsRead(ctx, created.ID, &created.ProfileID))
	afterRead, err := store.GetNotificationsByProfileID(ctx, 42, false, nil, nil)
	require.NoError(t, err)
	require.Len(t, afterRead, 1)
	require.True(t, afterRead[0].Read)

	require.NoError(t, store.MarkNotificationAsUnread(ctx, created.ID, &created.ProfileID))
	afterUnread, err := store.GetNotificationsByProfileID(ctx, 42, true, nil, nil)
	require.NoError(t, err)
	require.Len(t, afterUnread, 1)
	require.False(t, afterUnread[0].Read)

	require.NoError(t, store.MarkAllNotificationAsRead(ctx, 42))
	afterAllRead, err := store.GetNotificationsByProfileID(ctx, 42, true, nil, nil)
	require.NoError(t, err)
	require.Len(t, afterAllRead, 0)

	require.NoError(t, store.DeleteNotification(ctx, created.ID, &created.ProfileID))
	final, err := store.GetNotificationsByProfileID(ctx, 42, false, nil, nil)
	require.NoError(t, err)
	require.Len(t, final, 0)
}

func TestSQLNotificationStore_DeleteOnReadType(t *testing.T) {
	store := openNotificationStoreTestDB(t)
	ctx := context.Background()

	n := domain.Notification{
		Type:                      domain.NotificationTypeDailyConversationReminder,
		Title:                     "Daily",
		Text:                      "Question",
		ProfileID:                 9,
		ProfileIDWhoCausedNotif:   9,
		ResourceIDTiedToNotif:     55,
		ReadInNotificationsCenter: true,
	}
	created, err := store.CreateNotification(ctx, n)
	require.NoError(t, err)
	require.NoError(t, store.MarkNotificationAsRead(ctx, created.ID, &created.ProfileID))

	list, err := store.GetNotificationsByProfileID(ctx, 9, false, nil, nil)
	require.NoError(t, err)
	require.Len(t, list, 0)
}

func openNotificationStoreTestDB(t *testing.T) *SQLNotificationStore {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	store, err := NewSQLNotificationStoreWithSchema(db, "sqlite3")
	require.NoError(t, err)
	return store
}
