package notifierrepo_test

import (
	"testing"
	"time"

	"github.com/mikestefanello/pagoda/pkg/domain"
	"github.com/mikestefanello/pagoda/pkg/repos/notifierrepo"
	"github.com/mikestefanello/pagoda/pkg/repos/profilerepo"
	"github.com/mikestefanello/pagoda/pkg/tests"
	"github.com/stretchr/testify/assert"
)

func TestNotifications(t *testing.T) {
	client, ctx := tests.CreateTestContainerPostgresEntClient(t)
	defer client.Close()

	// Create users
	user1 := tests.CreateUser(ctx, client, "Jo Bandi", "jo@gmail.com", "password", true)
	user2 := tests.CreateUser(ctx, client, "Ba Babagaya", "ba@gmail.com", "password", true)

	// Create profiles
	notificationsRepo := notifierrepo.NewNotificationStorageRepo(client)
	profileRepo := profilerepo.NewProfileRepo(client, nil, nil)

	profile1Obj, err := profileRepo.CreateProfile(
		ctx, user1, "bio", time.Time{}, nil, nil,
	)
	assert.Nil(t, err)

	profile2Obj, err := profileRepo.CreateProfile(
		ctx, user2, "bio", time.Time{}, nil, nil,
	)
	assert.Nil(t, err)

	notif1 := domain.Notification{
		Type:      domain.NotificationTypeNewPrivateMessage,
		Text:      "You've got mail!",
		Link:      "https://chatbond.app",
		Read:      false,
		ProfileID: profile1Obj.ID,
	}

	notif2 := domain.Notification{
		Type:      domain.NotificationTypeNewPrivateMessage,
		Text:      "You've got another mail!",
		Link:      "https://chatbond.app",
		Read:      false,
		ProfileID: profile2Obj.ID,
	}

	newNotification1, err := notificationsRepo.CreateNotification(ctx, notif1)
	assert.Nil(t, err)
	assert.Equal(t, notif1.Type, newNotification1.Type)
	assert.Equal(t, notif1.Text, newNotification1.Text)
	assert.Equal(t, notif1.Link, newNotification1.Link)
	assert.Equal(t, notif1.Read, newNotification1.Read)
	assert.Equal(t, notif1.ProfileID, newNotification1.ProfileID)
	assert.NotNil(t, newNotification1.CreatedAt)
	assert.Equal(t, time.Time{}, newNotification1.ReadAt)

	newNotification2, err := notificationsRepo.CreateNotification(ctx, notif2)
	assert.Nil(t, err)
	assert.Equal(t, notif2.Type, newNotification2.Type)
	assert.Equal(t, notif2.Text, newNotification2.Text)
	assert.Equal(t, notif2.Link, newNotification2.Link)
	assert.Equal(t, notif2.Read, newNotification2.Read)
	assert.Equal(t, notif2.ProfileID, newNotification2.ProfileID)
	assert.NotNil(t, newNotification2.CreatedAt)
	assert.Equal(t, time.Time{}, newNotification2.ReadAt)

	notifs, err := notificationsRepo.GetNotificationsByProfileID(ctx, notif1.ProfileID, false, nil, nil)
	assert.Nil(t, err)
	assert.Len(t, notifs, 1)
	assert.Equal(t, newNotification1.ID, notifs[0].ID)

	err = notificationsRepo.MarkNotificationAsRead(ctx, newNotification1.ID, nil)
	assert.NoError(t, err)

	notifs, err = notificationsRepo.GetNotificationsByProfileID(ctx, notif1.ProfileID, false, nil, nil)
	assert.NoError(t, err)
	assert.Len(t, notifs, 1)
	assert.Equal(t, newNotification1.ID, notifs[0].ID)

	notifs, err = notificationsRepo.GetNotificationsByProfileID(ctx, notif1.ProfileID, true, nil, nil)
	assert.NoError(t, err)
	assert.Len(t, notifs, 0)

	notifs, err = notificationsRepo.GetNotificationsByProfileID(ctx, notif2.ProfileID, false, nil, nil)
	assert.NoError(t, err)
	assert.Len(t, notifs, 1)
	assert.Equal(t, newNotification2.ID, notifs[0].ID)

	entNotifs, err := client.Notification.Query().All(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(entNotifs))
}
