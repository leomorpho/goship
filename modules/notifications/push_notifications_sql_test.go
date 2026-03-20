package notifications

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"

	"github.com/leomorpho/goship/framework/domain"
)

func TestPwaPushService_SQLStore_Lifecycle(t *testing.T) {
	db := openPushStoresTestDB(t)
	permissionService := NewSQLNotificationPermissionService(db, "sqlite3")
	svc := NewSQLPwaPushService(db, "sqlite3", permissionService, "pub", "priv", "test@example.com")

	ctx := context.Background()
	err := svc.AddPushSubscription(ctx, 55, Subscription{
		Endpoint: "https://example.invalid/sub-1",
		P256dh:   "p256",
		Auth:     "auth",
	})
	require.NoError(t, err)

	endpoints, err := svc.GetPushSubscriptionEndpoints(ctx, 55)
	require.NoError(t, err)
	require.Len(t, endpoints, 1)
	require.Equal(t, "https://example.invalid/sub-1", endpoints[0])

	ok, err := svc.HasPermissionsLeftAndEndpointIsRegistered(ctx, 55, "https://example.invalid/sub-1")
	require.NoError(t, err)
	require.False(t, ok)

	err = permissionService.CreatePermission(ctx, 55, domain.NotificationPermissionDailyReminder, &domain.NotificationPlatformPush)
	require.NoError(t, err)

	ok, err = svc.HasPermissionsLeftAndEndpointIsRegistered(ctx, 55, "https://example.invalid/sub-1")
	require.NoError(t, err)
	require.True(t, ok)

	err = svc.DeletePushSubscriptionByEndpoint(ctx, 55, "https://example.invalid/sub-1")
	require.NoError(t, err)
	endpoints, err = svc.GetPushSubscriptionEndpoints(ctx, 55)
	require.NoError(t, err)
	require.Len(t, endpoints, 0)
}

func TestFcmPushService_SQLStore_Lifecycle(t *testing.T) {
	db := openPushStoresTestDB(t)
	permissionService := NewSQLNotificationPermissionService(db, "sqlite3")
	svc, err := NewSQLFcmPushService(db, "sqlite3", permissionService, nil)
	require.NoError(t, err)

	ctx := context.Background()
	err = svc.AddPushSubscription(ctx, 77, FcmSubscription{Token: "token-1"})
	require.NoError(t, err)

	ok, err := svc.HasPermissionsLeftAndTokenIsRegistered(ctx, 77, "token-1")
	require.NoError(t, err)
	require.False(t, ok)

	err = permissionService.CreatePermission(ctx, 77, domain.NotificationPermissionDailyReminder, &domain.NotificationPlatformFCMPush)
	require.NoError(t, err)

	ok, err = svc.HasPermissionsLeftAndTokenIsRegistered(ctx, 77, "token-1")
	require.NoError(t, err)
	require.True(t, ok)

	err = svc.DeletePushSubscriptionByToken(ctx, 77, "token-1")
	require.NoError(t, err)
	ok, err = svc.HasPermissionsLeftAndTokenIsRegistered(ctx, 77, "token-1")
	require.NoError(t, err)
	require.False(t, ok)
}

func openPushStoresTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = NewSQLNotificationStoreWithSchema(db, "sqlite3")
	require.NoError(t, err)
	return db
}
