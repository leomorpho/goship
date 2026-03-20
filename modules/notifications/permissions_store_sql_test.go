package notifications

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"

	"github.com/leomorpho/goship/framework/domain"
)

func TestSQLNotificationPermissionService_Lifecycle(t *testing.T) {
	db := openNotificationPermissionTestDB(t)
	svc := NewSQLNotificationPermissionService(db, "sqlite3")
	ctx := context.Background()
	profileID := 101

	perms, err := svc.GetPermissions(ctx, profileID)
	require.NoError(t, err)
	require.False(t, hasPermissionForPlatform(perms, domain.NotificationPermissionDailyReminder, domain.NotificationPlatformPush))

	err = svc.CreatePermission(ctx, profileID, domain.NotificationPermissionDailyReminder, &domain.NotificationPlatformPush)
	require.NoError(t, err)

	perms, err = svc.GetPermissions(ctx, profileID)
	require.NoError(t, err)
	require.True(t, hasPermissionForPlatform(perms, domain.NotificationPermissionDailyReminder, domain.NotificationPlatformPush))
	require.False(t, hasPermissionForPlatform(perms, domain.NotificationPermissionDailyReminder, domain.NotificationPlatformEmail))

	hasAny, err := svc.HasPermissionsForPlatform(ctx, profileID, domain.NotificationPlatformPush)
	require.NoError(t, err)
	require.True(t, hasAny)

	err = svc.DeletePermission(ctx, profileID, domain.NotificationPermissionDailyReminder, &domain.NotificationPlatformPush, nil)
	require.NoError(t, err)

	perms, err = svc.GetPermissions(ctx, profileID)
	require.NoError(t, err)
	require.False(t, hasPermissionForPlatform(perms, domain.NotificationPermissionDailyReminder, domain.NotificationPlatformPush))

	hasAny, err = svc.HasPermissionsForPlatform(ctx, profileID, domain.NotificationPlatformPush)
	require.NoError(t, err)
	require.False(t, hasAny)
}

func TestSQLNotificationPermissionService_CreateForAllPlatforms(t *testing.T) {
	db := openNotificationPermissionTestDB(t)
	svc := NewSQLNotificationPermissionService(db, "sqlite3")
	ctx := context.Background()
	profileID := 202

	err := svc.CreatePermission(ctx, profileID, domain.NotificationPermissionNewFriendActivity, nil)
	require.NoError(t, err)

	perms, err := svc.GetPermissions(ctx, profileID)
	require.NoError(t, err)
	require.True(t, hasPermissionForPlatform(perms, domain.NotificationPermissionNewFriendActivity, domain.NotificationPlatformPush))
	require.True(t, hasPermissionForPlatform(perms, domain.NotificationPermissionNewFriendActivity, domain.NotificationPlatformFCMPush))
	require.True(t, hasPermissionForPlatform(perms, domain.NotificationPermissionNewFriendActivity, domain.NotificationPlatformEmail))
	require.True(t, hasPermissionForPlatform(perms, domain.NotificationPermissionNewFriendActivity, domain.NotificationPlatformSMS))
}

func openNotificationPermissionTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = NewSQLNotificationStoreWithSchema(db, "sqlite3")
	require.NoError(t, err)
	return db
}

func hasPermissionForPlatform(
	perms map[domain.NotificationPermissionType]domain.NotificationPermission,
	permission domain.NotificationPermissionType,
	platform domain.NotificationPlatform,
) bool {
	perm, ok := perms[permission]
	if !ok {
		return false
	}
	for _, p := range perm.PlatformsList {
		if p.Platform == platform.Value {
			return p.Granted
		}
	}
	return false
}
