//go:build integration

package notifications_test

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	"github.com/leomorpho/goship-modules/notifications"
	profilesvc "github.com/leomorpho/goship/app/profile"
	"github.com/leomorpho/goship/db/ent/notificationpermission"
	"github.com/leomorpho/goship/framework/domain"
	storagerepo "github.com/leomorpho/goship/framework/repos/storage"
	"github.com/leomorpho/goship/framework/tests"
	"github.com/stretchr/testify/assert"
)

func TestFcmHasPermissionsLeftAndTokenIsRegistered(t *testing.T) {
	client, ctx := tests.CreateTestContainerPostgresEntClient(t)
	defer client.Close()

	// Create user and profile.
	user1 := tests.CreateUser(ctx, client, "User", "user1@example.com", "password", true)
	subscriptionsService := paidsubscriptions.NewSubscriptionsRepo(client, 10, 10)
	profileService := profilesvc.NewProfileService(client, storagerepo.NewMockStorageClient(), subscriptionsService)

	profile1, err := profileService.CreateProfile(
		ctx, user1, "bio",
		time.Now().AddDate(-25, 0, 0), nil, nil,
	)
	assert.Nil(t, err)

	// Set permissions
	uuidToken, err := uuid.NewV7(uuid.MicrosecondPrecision)
	assert.NoError(t, err)
	_, err = client.NotificationPermission.Create().
		SetProfileID(profile1.ID).
		SetPermission(notificationpermission.Permission(domain.NotificationPermissionDailyReminder.Value)).
		SetPlatform(notificationpermission.Platform(domain.NotificationPlatformPush.Value)).
		SetToken(uuidToken.String()).
		Save(ctx)
	assert.Nil(t, err)

	fcmPushService, err := notifications.NewFcmPushService(client, nil)
	assert.Nil(t, err)

	err = fcmPushService.AddPushSubscription(ctx, profile1.ID, notifications.FcmSubscription{
		Token: "12345",
	})
	assert.Nil(t, err)

	hasPermissionsLeft, err := fcmPushService.HasPermissionsLeftAndTokenIsRegistered(ctx, profile1.ID, "12345")
	assert.Nil(t, err)
	// TODO: the below is False even though it would normally be True, because
	// I did not explicitly add a permission, and fcmPushService is breaking
	// walls of responsability by putting its hands in permissions. Bad design. To rework.
	assert.False(t, hasPermissionsLeft)
}
