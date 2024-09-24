package notifierrepo_test

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/mikestefanello/pagoda/ent/notificationpermission"
	"github.com/mikestefanello/pagoda/pkg/domain"
	"github.com/mikestefanello/pagoda/pkg/repos/notifierrepo"
	"github.com/mikestefanello/pagoda/pkg/repos/profilerepo"
	storagerepo "github.com/mikestefanello/pagoda/pkg/repos/storage"
	"github.com/mikestefanello/pagoda/pkg/repos/subscriptions"
	"github.com/mikestefanello/pagoda/pkg/tests"
	"github.com/stretchr/testify/assert"
)

func TestFcmHasPermissionsLeftAndTokenIsRegistered(t *testing.T) {
	client, ctx := tests.CreateTestContainerPostgresEntClient(t)
	defer client.Close()

	// Create user and profile.
	user1 := tests.CreateUser(ctx, client, "User", "user1@example.com", "password", true)
	subscriptionsRepo := subscriptions.NewSubscriptionsRepo(client, 10, 10)
	profileRepo := profilerepo.NewProfileRepo(client, storagerepo.NewMockStorageClient(), subscriptionsRepo)

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

	fcmPushNotificationsRepo, err := notifierrepo.NewFcmPushNotificationsRepo(client, nil)
	assert.Nil(t, err)

	err = fcmPushNotificationsRepo.AddPushSubscription(ctx, profile1.ID, notifierrepo.FcmSubscription{
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
