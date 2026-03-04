//go:build integration

package notifications_test

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	"github.com/leomorpho/goship/app/notifications"
	"github.com/leomorpho/goship/app/profiles"
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
	subscriptionsRepo := paidsubscriptions.NewSubscriptionsRepo(client, 10, 10)
	profileRepo := profiles.NewProfileRepo(client, storagerepo.NewMockStorageClient(), subscriptionsRepo)

	profile1, err := profileRepo.CreateProfile(
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

	fcmPushNotificationsRepo, err := notifications.NewFcmPushNotificationsRepo(client, nil)
	assert.Nil(t, err)

	err = fcmPushNotificationsRepo.AddPushSubscription(ctx, profile1.ID, notifications.FcmSubscription{
		Token: "12345",
	})
	assert.Nil(t, err)

	hasPermissionsLeft, err := fcmPushNotificationsRepo.HasPermissionsLeftAndTokenIsRegistered(ctx, profile1.ID, "12345")
	assert.Nil(t, err)
	// TODO: the below is False even though it would normally be True, because
	// I did not explicitly add a permission, and fcmPushNotificationsRepo is breaking
	// walls of responsability by putting its hands in permissions. Bad design. To rework.
	assert.False(t, hasPermissionsLeft)
}
