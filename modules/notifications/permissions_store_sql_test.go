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
	require.False(t, hasPermissionForPlatform(perms, PermissionDailyReminder, PlatformPWAPush))

	err = svc.CreatePermission(ctx, profileID, PermissionDailyReminder, &PlatformPWAPush)
	require.NoError(t, err)

	perms, err = svc.GetPermissions(ctx, profileID)
	require.NoError(t, err)
	require.True(t, hasPermissionForPlatform(perms, PermissionDailyReminder, PlatformPWAPush))
	require.False(t, hasPermissionForPlatform(perms, PermissionDailyReminder, PlatformEmail))

	hasAny, err := svc.HasPermissionsForPlatform(ctx, profileID, PlatformPWAPush)
	require.NoError(t, err)
	require.True(t, hasAny)

	err = svc.DeletePermission(ctx, profileID, PermissionDailyReminder, &PlatformPWAPush, nil)
	require.NoError(t, err)

	perms, err = svc.GetPermissions(ctx, profileID)
	require.NoError(t, err)
	require.False(t, hasPermissionForPlatform(perms, PermissionDailyReminder, PlatformPWAPush))

	hasAny, err = svc.HasPermissionsForPlatform(ctx, profileID, PlatformPWAPush)
	require.NoError(t, err)
	require.False(t, hasAny)
}

func TestSQLNotificationPermissionService_CreateForAllPlatforms(t *testing.T) {
	db := openNotificationPermissionTestDB(t)
	svc := NewSQLNotificationPermissionService(db, "sqlite3")
	ctx := context.Background()
	profileID := 202

	err := svc.CreatePermission(ctx, profileID, PermissionNewFriendActivity, nil)
	require.NoError(t, err)

	perms, err := svc.GetPermissions(ctx, profileID)
	require.NoError(t, err)
	require.True(t, hasPermissionForPlatform(perms, PermissionNewFriendActivity, PlatformPWAPush))
	require.True(t, hasPermissionForPlatform(perms, PermissionNewFriendActivity, PlatformFCMPush))
	require.True(t, hasPermissionForPlatform(perms, PermissionNewFriendActivity, PlatformEmail))
	require.True(t, hasPermissionForPlatform(perms, PermissionNewFriendActivity, PlatformSMS))
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
	perms map[PermissionType]domain.NotificationPermission,
	permission PermissionType,
	platform Platform,
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
