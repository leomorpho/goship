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

// Convert PlatformsList to mapsets for comparison
func convertToMap(pl []domain.NotificationPermissionPlatform) map[string]bool {
	platformMap := make(map[string]bool)
	for _, p := range pl {
		platformMap[p.Platform] = p.Granted
	}
	return platformMap
}
func TestGetPermissions(t *testing.T) {
	client, ctx := tests.CreateTestContainerPostgresEntClient(t)
	defer client.Close()

	// Create user and profile.
	user1 := tests.CreateUser(ctx, client, "User", "user1@example.com", "password", true)
	subscriptionsRepo := subscriptions.NewSubscriptionsRepo(client, 10, 10)
	profileRepo := profilerepo.NewProfileRepo(client, storagerepo.NewMockStorageClient(), subscriptionsRepo)
	notifSendPermissionRepo := notifierrepo.NewNotificationSendPermissionRepo(client)

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

	// Test getting permissions
	permissions, err := notifSendPermissionRepo.GetPermissions(ctx, profile1.ID)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(permissions))

	notifMap := make(map[domain.NotificationPermissionType]domain.NotificationPermission)
	notifMap[domain.NotificationPermissionDailyReminder] = domain.NotificationPermission{
		Title:      "Daily conversation",
		Subtitle:   "A reminder to not miss today's question, sent at most once a day.",
		Permission: domain.NotificationPermissionDailyReminder.Value,
		PlatformsList: []domain.NotificationPermissionPlatform{
			{
				Platform: domain.NotificationPlatformPush.Value, Granted: true,
			},
			{
				Platform: domain.NotificationPlatformFCMPush.Value, Granted: false,
			},
			{
				Platform: domain.NotificationPlatformEmail.Value, Granted: false,
			},
			{
				Platform: domain.NotificationPlatformSMS.Value, Granted: false,
			},
		},
	}
	notifMap[domain.NotificationPermissionNewFriendActivity] = domain.NotificationPermission{
		Title:      "Partner activity",
		Subtitle:   "Answers you missed, sent at most once a day.",
		Permission: domain.NotificationPermissionNewFriendActivity.Value,
		PlatformsList: []domain.NotificationPermissionPlatform{
			{
				Platform: domain.NotificationPlatformPush.Value, Granted: false,
			},
			{
				Platform: domain.NotificationPlatformFCMPush.Value, Granted: false,
			},
			{
				Platform: domain.NotificationPlatformEmail.Value, Granted: false,
			},
			{
				Platform: domain.NotificationPlatformSMS.Value, Granted: false,
			},
		},
	}
	assert.Equal(t, notifMap[domain.NotificationPermissionDailyReminder].Title, permissions[domain.NotificationPermissionDailyReminder].Title)
	assert.Equal(t, notifMap[domain.NotificationPermissionDailyReminder].Subtitle, permissions[domain.NotificationPermissionDailyReminder].Subtitle)
	assert.Equal(t, notifMap[domain.NotificationPermissionDailyReminder].Permission, permissions[domain.NotificationPermissionDailyReminder].Permission)
	assert.Equal(t, 4, len(notifMap[domain.NotificationPermissionDailyReminder].PlatformsList))

	assert.Equal(t, notifMap[domain.NotificationPermissionNewFriendActivity].Title, permissions[domain.NotificationPermissionNewFriendActivity].Title)
	assert.Equal(t, notifMap[domain.NotificationPermissionNewFriendActivity].Subtitle, permissions[domain.NotificationPermissionNewFriendActivity].Subtitle)
	assert.Equal(t, notifMap[domain.NotificationPermissionNewFriendActivity].Permission, permissions[domain.NotificationPermissionNewFriendActivity].Permission)
	assert.Equal(t, 4, len(notifMap[domain.NotificationPermissionNewFriendActivity].PlatformsList))

	expectedMap := convertToMap(notifMap[domain.NotificationPermissionDailyReminder].PlatformsList)
	actualMap := convertToMap(permissions[domain.NotificationPermissionDailyReminder].PlatformsList)
	assert.Equal(t, expectedMap, actualMap)
}

func TestCreatePermission(t *testing.T) {
	client, ctx := tests.CreateTestContainerPostgresEntClient(t)
	defer client.Close()

	// Create user and profile.
	user1 := tests.CreateUser(ctx, client, "User", "user1@example.com", "password", true)
	subscriptionsRepo := subscriptions.NewSubscriptionsRepo(client, 10, 10)
	profileRepo := profilerepo.NewProfileRepo(client, storagerepo.NewMockStorageClient(), subscriptionsRepo)

	notifSendPermissionRepo := notifierrepo.NewNotificationSendPermissionRepo(client)

	profile1, err := profileRepo.CreateProfile(
		ctx, user1, "bio",
		time.Now().AddDate(-25, 0, 0), nil, nil,
	)
	assert.Nil(t, err)

	// Test creating a permission
	err = notifSendPermissionRepo.CreatePermission(
		ctx, profile1.ID, domain.NotificationPermissionDailyReminder, &domain.NotificationPlatformPush)
	assert.Nil(t, err)

	// Validate the permission was created
	permissions, err := notifSendPermissionRepo.GetPermissions(ctx, profile1.ID)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(permissions))

	notifMap := make(map[domain.NotificationPermissionType]domain.NotificationPermission)
	notifMap[domain.NotificationPermissionDailyReminder] = domain.NotificationPermission{
		PlatformsList: []domain.NotificationPermissionPlatform{
			{
				Platform: domain.NotificationPlatformPush.Value, Granted: true,
			},
			{
				Platform: domain.NotificationPlatformFCMPush.Value, Granted: false,
			},
			{
				Platform: domain.NotificationPlatformEmail.Value, Granted: false,
			},
			{
				Platform: domain.NotificationPlatformSMS.Value, Granted: false,
			},
		},
	}
	notifMap[domain.NotificationPermissionNewFriendActivity] = domain.NotificationPermission{
		PlatformsList: []domain.NotificationPermissionPlatform{
			{
				Platform: domain.NotificationPlatformPush.Value, Granted: false,
			},
			{
				Platform: domain.NotificationPlatformFCMPush.Value, Granted: false,
			},
			{
				Platform: domain.NotificationPlatformEmail.Value, Granted: false,
			},
			{
				Platform: domain.NotificationPlatformSMS.Value, Granted: false,
			},
		},
	}

	expectedMap := convertToMap(notifMap[domain.NotificationPermissionDailyReminder].PlatformsList)
	actualMap := convertToMap(permissions[domain.NotificationPermissionDailyReminder].PlatformsList)
	assert.Equal(t, expectedMap, actualMap)
}

func TestDeletePermission(t *testing.T) {
	client, ctx := tests.CreateTestContainerPostgresEntClient(t)
	defer client.Close()

	// Create user and profile.
	user1 := tests.CreateUser(ctx, client, "User", "user1@example.com", "password", true)
	subscriptionsRepo := subscriptions.NewSubscriptionsRepo(client, 10, 10)
	profileRepo := profilerepo.NewProfileRepo(client, storagerepo.NewMockStorageClient(), subscriptionsRepo)

	notifSendPermissionRepo := notifierrepo.NewNotificationSendPermissionRepo(client)

	profile1, err := profileRepo.CreateProfile(
		ctx, user1, "bio",
		time.Now().AddDate(-25, 0, 0), nil, nil,
	)
	assert.Nil(t, err)

	// Set permissions
	uuidToken, err := uuid.NewV7(uuid.MicrosecondPrecision)
	assert.NoError(t, err)
	uuidTokenStr := uuidToken.String()

	_, err = client.NotificationPermission.Create().
		SetProfileID(profile1.ID).
		SetPermission(notificationpermission.Permission(domain.NotificationPermissionDailyReminder.Value)).
		SetPlatform(notificationpermission.Platform(domain.NotificationPlatformPush.Value)).
		SetToken(uuidTokenStr).
		Save(ctx)
	assert.Nil(t, err)

	// Validate the permission was created
	permissions, err := notifSendPermissionRepo.GetPermissions(ctx, profile1.ID)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(permissions))

	notifMap := make(map[domain.NotificationPermissionType]domain.NotificationPermission)
	notifMap[domain.NotificationPermissionDailyReminder] = domain.NotificationPermission{
		PlatformsList: []domain.NotificationPermissionPlatform{
			{
				Platform: domain.NotificationPlatformPush.Value, Granted: true,
			},
			{
				Platform: domain.NotificationPlatformFCMPush.Value, Granted: false,
			},

			{
				Platform: domain.NotificationPlatformEmail.Value, Granted: false,
			},
			{
				Platform: domain.NotificationPlatformSMS.Value, Granted: false,
			},
		},
	}
	notifMap[domain.NotificationPermissionNewFriendActivity] = domain.NotificationPermission{
		PlatformsList: []domain.NotificationPermissionPlatform{
			{
				Platform: domain.NotificationPlatformPush.Value, Granted: false,
			},
			{
				Platform: domain.NotificationPlatformFCMPush.Value, Granted: false,
			},

			{
				Platform: domain.NotificationPlatformEmail.Value, Granted: false,
			},
			{
				Platform: domain.NotificationPlatformSMS.Value, Granted: false,
			},
		},
	}

	expectedMap := convertToMap(notifMap[domain.NotificationPermissionDailyReminder].PlatformsList)
	actualMap := convertToMap(permissions[domain.NotificationPermissionDailyReminder].PlatformsList)
	assert.Equal(t, expectedMap, actualMap)

	// Test deleting the permission
	err = notifSendPermissionRepo.DeletePermission(
		ctx, profile1.ID, domain.NotificationPermissionDailyReminder, &domain.NotificationPlatformPush, &uuidTokenStr)
	assert.Nil(t, err)

	// Validate the permission was deleted
	permissions, err = notifSendPermissionRepo.GetPermissions(ctx, profile1.ID)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(permissions))

	notifMap = make(map[domain.NotificationPermissionType]domain.NotificationPermission)
	notifMap[domain.NotificationPermissionDailyReminder] = domain.NotificationPermission{
		PlatformsList: []domain.NotificationPermissionPlatform{
			{
				Platform: domain.NotificationPlatformPush.Value, Granted: false,
			},
			{
				Platform: domain.NotificationPlatformFCMPush.Value, Granted: false,
			},
			{
				Platform: domain.NotificationPlatformEmail.Value, Granted: false,
			},
			{
				Platform: domain.NotificationPlatformSMS.Value, Granted: false,
			},
		},
	}
	notifMap[domain.NotificationPermissionNewFriendActivity] = domain.NotificationPermission{
		PlatformsList: []domain.NotificationPermissionPlatform{
			{
				Platform: domain.NotificationPlatformPush.Value, Granted: false,
			},
			{
				Platform: domain.NotificationPlatformFCMPush.Value, Granted: false,
			},
			{
				Platform: domain.NotificationPlatformEmail.Value, Granted: false,
			},
			{
				Platform: domain.NotificationPlatformSMS.Value, Granted: false,
			},
		},
	}

	expectedMap = convertToMap(notifMap[domain.NotificationPermissionDailyReminder].PlatformsList)
	actualMap = convertToMap(permissions[domain.NotificationPermissionDailyReminder].PlatformsList)
	assert.Equal(t, expectedMap, actualMap)
}
