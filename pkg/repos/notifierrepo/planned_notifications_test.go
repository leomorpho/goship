package notifierrepo_test

import (
	"testing"
	"time"

	"database/sql"

	"github.com/gofrs/uuid"
	"github.com/mikestefanello/pagoda/ent"
	"github.com/mikestefanello/pagoda/ent/lastseenonline"
	"github.com/mikestefanello/pagoda/ent/notification"
	"github.com/mikestefanello/pagoda/ent/notificationpermission"
	"github.com/mikestefanello/pagoda/ent/notificationtime"
	"github.com/mikestefanello/pagoda/ent/profile"
	"github.com/mikestefanello/pagoda/ent/user"
	"github.com/mikestefanello/pagoda/pkg/domain"
	"github.com/mikestefanello/pagoda/pkg/repos/notifierrepo"
	"github.com/mikestefanello/pagoda/pkg/repos/profilerepo"
	storagerepo "github.com/mikestefanello/pagoda/pkg/repos/storage"
	"github.com/mikestefanello/pagoda/pkg/repos/subscriptions"
	"github.com/mikestefanello/pagoda/pkg/tests"
	"github.com/stretchr/testify/assert"

	"github.com/jackc/pgx/stdlib"
)

func init() {
	// Register "pgx" as "postgres" explicitly for database/sql
	sql.Register("postgres", stdlib.GetDefaultDriver())
}

func TestUpsertNotificationTime(t *testing.T) {
	client, ctx := tests.CreateTestContainerPostgresEntClient(t)
	defer client.Close()

	// Create users and profiles.
	user1 := tests.CreateUser(ctx, client, "User", "user1@example.com", "password", true)
	subscriptionsRepo := subscriptions.NewSubscriptionsRepo(client, 10, 10)
	profileRepo := profilerepo.NewProfileRepo(client, storagerepo.NewMockStorageClient(), subscriptionsRepo)

	plannedNotifsRepo := notifierrepo.NewPlannedNotificationsRepo(client, subscriptionsRepo)
	profile1, err := profileRepo.CreateProfile(
		ctx, user1, "bio",
		time.Now().AddDate(-25, 0, 0), nil, nil,
	)
	assert.Nil(t, err)

	// Insert last seen online times
	midnight := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location())
	offset1 := 60
	offset2 := 120
	offset3 := 180
	offset4 := 185
	offset5 := 190
	lastSeenTimes := []time.Time{
		midnight.Add(time.Duration(offset1) * time.Minute), // 1:00 AM
		midnight.Add(time.Duration(offset2) * time.Minute), // 2:00 AM
		midnight.Add(time.Duration(offset3) * time.Minute), // 3:00 AM
		midnight.Add(time.Duration(offset4) * time.Minute), // 3:00 AM
		midnight.Add(time.Duration(offset5) * time.Minute), // 3:00 AM
	}
	for _, ts := range lastSeenTimes {
		_, err := client.LastSeenOnline.Create().
			SetSeenAt(ts).
			SetUserID(user1.ID).
			Save(ctx)
		assert.Nil(t, err)
	}

	// Test upserting notification time
	minutes, err := plannedNotifsRepo.UpsertNotificationTime(ctx, profile1.ID, domain.NotificationTypeDailyConversationReminder)
	assert.Nil(t, err)
	expectedMinutes := 180
	assert.LessOrEqual(t, expectedMinutes-2, minutes)
	assert.GreaterOrEqual(t, expectedMinutes+2, minutes)

	// Fetch the updated notification time
	notificationTime, err := client.NotificationTime.
		Query().
		Where(notificationtime.HasProfileWith(profile.IDEQ(profile1.ID))).
		Where(notificationtime.TypeEQ(notificationtime.Type(domain.NotificationTypeDailyConversationReminder.Value))).
		Only(ctx)
	assert.Nil(t, err)

	// Validate the result
	assert.LessOrEqual(t, expectedMinutes-2, notificationTime.SendMinute)
	assert.GreaterOrEqual(t, expectedMinutes+2, notificationTime.SendMinute)

}

func TestCreateNotificationTimeObjects(t *testing.T) {
	client, ctx := tests.CreateTestContainerPostgresEntClient(t)
	defer client.Close()

	// Create users and profiles.
	user1 := tests.CreateUser(ctx, client, "User", "user1@example.com", "password", true)
	user2 := tests.CreateUser(ctx, client, "User", "user2@example.com", "password", true)
	subscriptionsRepo := subscriptions.NewSubscriptionsRepo(client, 10, 10)
	profileRepo := profilerepo.NewProfileRepo(client, storagerepo.NewMockStorageClient(), subscriptionsRepo)

	plannedNotifsRepo := notifierrepo.NewPlannedNotificationsRepo(client, subscriptionsRepo)

	profile1, err := profileRepo.CreateProfile(
		ctx, user1, "bio1",
		time.Now().AddDate(-25, 0, 0), nil, nil,
	)
	assert.Nil(t, err)
	profile2, err := profileRepo.CreateProfile(
		ctx, user2, "bio2",
		time.Now().AddDate(-25, 0, 0), nil, nil,
	)
	assert.Nil(t, err)

	// Have 1 user with no last seen objects, and the other with them
	// Insert last seen online times
	midnight := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location())
	offset1 := 60
	offset2 := 120
	offset3 := 180
	offset4 := 185
	offset5 := 190

	lastSeenTimes := []time.Time{
		midnight.Add(time.Duration(offset1) * time.Minute), // 1:00 AM
		midnight.Add(time.Duration(offset2) * time.Minute), // 2:00 AM
		midnight.Add(time.Duration(offset3) * time.Minute), // 3:00 AM
		midnight.Add(time.Duration(offset4) * time.Minute), // 3:00 AM
		midnight.Add(time.Duration(offset5) * time.Minute), // 3:00 AM
	}
	for _, ts := range lastSeenTimes {
		_, err := client.LastSeenOnline.Create().
			SetSeenAt(ts).
			SetUserID(user1.ID).
			Save(ctx)
		assert.Nil(t, err)
	}

	// Set permissions
	uuidToken, err := uuid.NewV7(uuid.MicrosecondPrecision)
	assert.Nil(t, err)

	_, err = client.NotificationPermission.Create().
		SetProfileID(profile1.ID).
		SetPermission(notificationpermission.Permission(domain.NotificationPermissionDailyReminder.Value)).
		SetPlatform(notificationpermission.Platform(domain.NotificationPlatformPush.Value)).
		SetToken(uuidToken.String()).
		Save(ctx)
	assert.Nil(t, err)

	uuidToken, err = uuid.NewV7(uuid.MicrosecondPrecision)
	assert.Nil(t, err)

	_, err = client.NotificationPermission.Create().
		SetProfileID(profile2.ID).
		SetPermission(notificationpermission.Permission(domain.NotificationPermissionDailyReminder.Value)).
		SetPlatform(notificationpermission.Platform(domain.NotificationPlatformPush.Value)).
		SetToken(uuidToken.String()).
		Save(ctx)
	assert.Nil(t, err)

	// Test creating/updating notification time objects
	err = plannedNotifsRepo.CreateNotificationTimeObjects(
		ctx, domain.NotificationTypeDailyConversationReminder,
		domain.NotificationPermissionDailyReminder,
	)
	assert.Nil(t, err)

	// Fetch and validate notification times
	profiles := []int{profile1.ID, profile2.ID}
	for _, profileID := range profiles {
		notificationTime, err := client.NotificationTime.
			Query().
			Where(notificationtime.HasProfileWith(profile.IDEQ(profileID))).
			Where(notificationtime.TypeEQ(notificationtime.Type(domain.NotificationTypeDailyConversationReminder.Value))).
			Only(ctx)

		if profileID == profile1.ID {
			assert.Nil(t, err)
			assert.NotNil(t, notificationTime)
		}
		if profileID == profile2.ID {
			assert.True(t, ent.IsNotFound(err))
			assert.Nil(t, notificationTime)
		}

	}
}

func TestDeleteStaleLastSeenObjects(t *testing.T) {
	client, ctx := tests.CreateTestContainerPostgresEntClient(t)
	defer client.Close()

	// Create user and profiles.
	user1 := tests.CreateUser(ctx, client, "User", "user1@example.com", "password", true)
	subscriptionsRepo := subscriptions.NewSubscriptionsRepo(client, 10, 10)
	profileRepo := profilerepo.NewProfileRepo(client, storagerepo.NewMockStorageClient(), subscriptionsRepo)

	plannedNotifsRepo := notifierrepo.NewPlannedNotificationsRepo(client, subscriptionsRepo)
	_, err := profileRepo.CreateProfile(
		ctx, user1, "bio",
		time.Now().AddDate(-25, 0, 0), nil, nil,
	)
	assert.Nil(t, err)

	// Insert last seen online times, some of which are stale
	currentTime := time.Now()
	staleTime := currentTime.AddDate(0, 0, -31)
	validTime := currentTime.AddDate(0, 0, -15)
	lastSeenTimes := []time.Time{
		staleTime,   // Should be deleted
		validTime,   // Should not be deleted
		currentTime, // Should not be deleted
	}
	for _, ts := range lastSeenTimes {
		_, err := client.LastSeenOnline.Create().
			SetSeenAt(ts).
			SetUserID(user1.ID).
			Save(ctx)
		assert.Nil(t, err)
	}

	// Call deleteStaleLastSeenObjects
	plannedNotifsRepo.DeleteStaleLastSeenObjects(ctx)

	// Fetch remaining last seen online times
	lastSeenOnlineRecords, err := client.LastSeenOnline.Query().Where(lastseenonline.HasUserWith(user.IDEQ(user1.ID))).All(ctx)
	assert.Nil(t, err)

	// Validate that only the valid records remain
	for _, record := range lastSeenOnlineRecords {
		assert.True(t, record.SeenAt.After(currentTime.AddDate(0, 0, -30)), "Expected last seen online record to be within the last 30 days")
	}
}

func TestProfileIDsCanGetPlannedNotificationNow(t *testing.T) {
	client, ctx := tests.CreateTestContainerPostgresEntClient(t)
	defer client.Close()

	// Step 1: Create users and profiles.
	user1 := tests.CreateUser(ctx, client, "User", "user1@example.com", "password", true)
	user2 := tests.CreateUser(ctx, client, "User", "user2@example.com", "password", true)
	user3 := tests.CreateUser(ctx, client, "User", "user3@example.com", "password", true)
	user4 := tests.CreateUser(ctx, client, "User", "user4@example.com", "password", true)
	subscriptionsRepo := subscriptions.NewSubscriptionsRepo(client, 10, 10)
	profileRepo := profilerepo.NewProfileRepo(client, storagerepo.NewMockStorageClient(), subscriptionsRepo)

	plannedNotifsRepo := notifierrepo.NewPlannedNotificationsRepo(client, subscriptionsRepo)

	// Create profiles with different notification times.
	profile1, err := profileRepo.CreateProfile(
		ctx, user1, "bio1",
		time.Now().AddDate(-25, 0, 0), nil, nil,
	)
	assert.Nil(t, err)
	profile2, err := profileRepo.CreateProfile(
		ctx, user2, "bio2",
		time.Now().AddDate(-25, 0, 0), nil, nil,
	)
	assert.Nil(t, err)
	profile3, err := profileRepo.CreateProfile(
		ctx, user3, "bio3",
		time.Now().AddDate(-25, 0, 0), nil, nil,
	)
	assert.Nil(t, err)
	profile4, err := profileRepo.CreateProfile(
		ctx, user4, "bio4",
		time.Now().AddDate(-25, 0, 0), nil, nil,
	)
	assert.Nil(t, err)

	// Step 2: Set up notification times.
	// Profiles have different notification times set in minutes from midnight.
	sendMinute1 := 60  // 1:00 AM
	sendMinute2 := 120 // 2:00 AM
	sendMinute3 := 180 // 3:00 AM
	sendMinute4 := 240 // 4:00 AM
	_, err = client.NotificationTime.Create().
		SetProfileID(profile1.ID).
		SetType(notificationtime.Type(domain.NotificationTypeDailyConversationReminder.Value)).
		SetSendMinute(sendMinute1).
		Save(ctx)
	assert.Nil(t, err)
	_, err = client.NotificationTime.Create().
		SetProfileID(profile2.ID).
		SetType(notificationtime.Type(domain.NotificationTypeDailyConversationReminder.Value)).
		SetSendMinute(sendMinute2).
		Save(ctx)
	assert.Nil(t, err)
	_, err = client.NotificationTime.Create().
		SetProfileID(profile3.ID).
		SetType(notificationtime.Type(domain.NotificationTypeDailyConversationReminder.Value)).
		SetSendMinute(sendMinute3).
		Save(ctx)
	assert.Nil(t, err)
	_, err = client.NotificationTime.Create().
		SetProfileID(profile4.ID).
		SetType(notificationtime.Type(domain.NotificationTypeDailyConversationReminder.Value)).
		SetSendMinute(sendMinute4).
		Save(ctx)
	assert.Nil(t, err)

	// Step 3: Set up existing notifications to avoid double notification.
	// Profiles 2 and 3 have already received notifications at 2:00 AM and 3:00 AM respectively.
	now := time.Now().UTC()
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	_, err = client.Notification.Create().
		SetProfileID(profile2.ID).
		SetText("You've got mail!").
		SetType(notification.Type(domain.NotificationTypeDailyConversationReminder.Value)).
		SetCreatedAt(midnight.Add(2 * time.Hour)). // 2:00 AM
		Save(ctx)
	assert.Nil(t, err)
	_, err = client.Notification.Create().
		SetProfileID(profile3.ID).
		SetText("You've got mail!").
		SetType(notification.Type(domain.NotificationTypeDailyConversationReminder.Value)).
		SetCreatedAt(midnight.Add(3 * time.Hour)). // 3:00 AM
		Save(ctx)
	assert.Nil(t, err)

	// Step 4: Define test cases with different timestamps.
	// Test Case 1: 1:30 AM timestamp
	timestamp1 := time.Date(now.Year(), now.Month(), now.Day(), 1, 30, 0, 0, time.UTC)
	// Only profile1 should be notified since its notification time is 1:00 AM and the timestamp is 1:30 AM.
	profiles, err := plannedNotifsRepo.ProfileIDsCanGetPlannedNotificationNow(
		ctx, timestamp1, domain.NotificationTypeDailyConversationReminder, nil)
	assert.Nil(t, err)
	expectedProfiles := []int{profile1.ID}
	assert.Equal(t, expectedProfiles, profiles)

	// Test Case 2: 5:30 AM timestamp
	timestamp2 := time.Date(now.Year(), now.Month(), now.Day(), 5, 30, 0, 0, time.UTC)
	// Profiles 1 and 4 should be notified as their notification times (1:00 AM and 4:00 AM) have passed and profiles 2 and 3 already received their notifications.
	profiles, err = plannedNotifsRepo.ProfileIDsCanGetPlannedNotificationNow(
		ctx, timestamp2, domain.NotificationTypeDailyConversationReminder, nil)
	assert.Nil(t, err)
	expectedProfiles = []int{profile1.ID, profile4.ID}
	assert.Equal(t, expectedProfiles, profiles)

	// Test Case 3: 4:00 AM timestamp
	timestamp3 := time.Date(now.Year(), now.Month(), now.Day(), 4, 0, 0, 0, time.UTC)
	// Profiles 1 and 4 should be notified since their notification times (1:00 AM and 4:00 AM) are less than or equal to the 4:00 AM timestamp.
	profiles, err = plannedNotifsRepo.ProfileIDsCanGetPlannedNotificationNow(
		ctx, timestamp3, domain.NotificationTypeDailyConversationReminder, nil)
	assert.Nil(t, err)
	expectedProfiles = []int{profile1.ID, profile4.ID}
	assert.Equal(t, expectedProfiles, profiles)

	// Test Case 4: 4:00 AM timestamp
	// Profiles 1 should be notified since their notification times (1:00 AM) are less than or equal to the 4:00 AM timestamp, and we only want to check for them.
	profileIdsWhoWeWantToCheck := []int{profile1.ID}
	profiles, err = plannedNotifsRepo.ProfileIDsCanGetPlannedNotificationNow(
		ctx, timestamp3, domain.NotificationTypeDailyConversationReminder, &profileIdsWhoWeWantToCheck)
	assert.Nil(t, err)
	expectedProfiles = []int{profile1.ID}
	assert.Equal(t, expectedProfiles, profiles)

	// Test Case 5: No profiles should be notified after 3:00 AM
	timestamp4 := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, time.UTC)
	// At 3:00 AM, profile 1 should be notified, but profiles 2 and 3 should not be notified again as they already received notifications at 2:00 AM and 3:00 AM respectively.
	profiles, err = plannedNotifsRepo.ProfileIDsCanGetPlannedNotificationNow(
		ctx, timestamp4, domain.NotificationTypeDailyConversationReminder, nil)
	assert.Nil(t, err)
	expectedProfiles = []int{profile1.ID}
	assert.Equal(t, expectedProfiles, profiles)
}
