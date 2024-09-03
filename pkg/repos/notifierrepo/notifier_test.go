package notifierrepo_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mikestefanello/pagoda/pkg/domain"
	"github.com/mikestefanello/pagoda/pkg/repos/notifierrepo"
	"github.com/mikestefanello/pagoda/pkg/repos/pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockPubSubClient struct {
	mock.Mock
}

func (m *MockPubSubClient) Publish(ctx context.Context, topic string, event pubsub.SSEEvent) error {
	args := m.Called(ctx, topic, event)
	return args.Error(0)
}

func (m *MockPubSubClient) SSESubscribe(ctx context.Context, topic string) (<-chan pubsub.SSEEvent, error) {
	args := m.Called(ctx, topic)
	return args.Get(0).(<-chan pubsub.SSEEvent), args.Error(1)
}

func (m *MockPubSubClient) DeleteNotification(ctx context.Context, notificationID int) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type MockNotificationStorageRepo struct {
	mock.Mock
}

func (m *MockNotificationStorageRepo) CreateNotification(
	ctx context.Context, n domain.Notification,
) (*domain.Notification, error) {
	args := m.Called(ctx, n)
	return args.Get(0).(*domain.Notification), args.Error(1)
}

func (m *MockNotificationStorageRepo) GetNotificationsByProfileID(
	ctx context.Context, profileID int, onlyUnread bool, beforeTimestamp *time.Time, pageSize *int,
) ([]*domain.Notification, error) {
	args := m.Called(ctx, profileID, onlyUnread)
	return args.Get(0).([]*domain.Notification), args.Error(1)
}

func (m *MockNotificationStorageRepo) MarkNotificationAsRead(
	ctx context.Context, notificationID int, profileID *int,
) error {
	args := m.Called(ctx, notificationID)
	return args.Error(0)
}

func (m *MockNotificationStorageRepo) MarkAllNotificationAsRead(
	ctx context.Context, profileID int,
) error {
	args := m.Called(ctx, profileID)
	return args.Error(0)
}

func (m *MockNotificationStorageRepo) MarkNotificationAsUnread(
	ctx context.Context, notificationID int, profileID *int,
) error {
	args := m.Called(ctx, notificationID)
	return args.Error(0)
}

func (m *MockNotificationStorageRepo) DeleteNotification(
	ctx context.Context, notificationID int, profileID *int,
) error {
	args := m.Called(ctx, notificationID)
	return args.Error(0)
}

func (m *MockNotificationStorageRepo) HasNotificationForResourceAndPerson(
	ctx context.Context, notifType domain.NotificationType, profileIDWhoCausedNotif, resourceID *int, maxAge time.Duration,
) (bool, error) {
	args := m.Called(ctx, notifType, profileIDWhoCausedNotif, resourceID, maxAge)
	return args.Get(0).(bool), args.Error(1)
}

func TestCreateNotification(t *testing.T) {
	ctx := context.Background()
	notification := domain.Notification{ProfileID: 1, Type: domain.NotificationTypeNewPrivateMessage, Text: "Test notification"}

	testCases := []struct {
		name         string
		storeInDB    bool
		expectCreate bool // Whether we expect the CreateNotification method to be called on the repo
	}{
		{"StoreInDB", true, true},
		{"DoNotStoreInDB", false, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockPubSubClient := new(MockPubSubClient)
			mockNotificationStorageRepo := new(MockNotificationStorageRepo)

			// Setting expectations based on the test case
			if tc.expectCreate {
				mockNotificationStorageRepo.
					On("CreateNotification", ctx, notification).
					Return(&notification, nil)
			}
			mockPubSubClient.
				On("Publish", ctx, fmt.Sprint(notification.ProfileID), mock.Anything).
				Return(nil)

			// Create notifier repo
			notifierRepo := notifierrepo.NewNotifierRepo(mockPubSubClient, mockNotificationStorageRepo, nil, nil, nil)

			// Test CreateNotification
			err := notifierRepo.PublishNotification(ctx, notification, tc.storeInDB, false)
			assert.NoError(t, err)

			if tc.expectCreate {
				mockNotificationStorageRepo.AssertExpectations(t)
			}
			mockPubSubClient.AssertExpectations(t)
		})
	}
}

func TestGetNotifications(t *testing.T) {
	ctx := context.Background()
	mockPubSubClient := new(MockPubSubClient)
	mockNotificationStorageRepo := new(MockNotificationStorageRepo)
	profileID := 1
	notifications := []*domain.Notification{{ID: 1, ProfileID: profileID, Type: domain.NotificationTypeNewPrivateMessage, Text: "Test Notification"}}

	// Set expectations
	mockNotificationStorageRepo.On("GetNotificationsByProfileID", ctx, profileID, false).Return(notifications, nil)

	// Create notifier repo
	notifierRepo := notifierrepo.NewNotifierRepo(mockPubSubClient, mockNotificationStorageRepo, nil, nil, nil)

	// Test GetNotifications
	result, err := notifierRepo.GetNotifications(ctx, profileID, false, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, notifications, result)

	mockNotificationStorageRepo.AssertExpectations(t)
}

func TestMarkNotificationRead(t *testing.T) {
	ctx := context.Background()
	mockPubSubClient := new(MockPubSubClient)
	mockNotificationStorageRepo := new(MockNotificationStorageRepo)
	notificationID := 1

	// Set expectations
	mockNotificationStorageRepo.On("MarkNotificationAsRead", ctx, notificationID).Return(nil)

	// Create notifier repo
	notifierRepo := notifierrepo.NewNotifierRepo(mockPubSubClient, mockNotificationStorageRepo, nil, nil, nil)

	// Test MarkNotificationRead
	err := notifierRepo.MarkNotificationRead(ctx, notificationID, nil)
	assert.NoError(t, err)

	mockNotificationStorageRepo.AssertExpectations(t)
}

func TestSubscribe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure the context gets cancelled to clean up resources

	mockPubSubClient := new(MockPubSubClient)
	mockNotificationStorageRepo := new(MockNotificationStorageRepo)
	topic := "someTopic"

	// Create a channel to return from the mocked SSESubscribe
	eventCh := make(chan pubsub.SSEEvent)
	// Mock SSESubscribe to return the channel
	mockPubSubClient.On("SSESubscribe", ctx, topic).Return((<-chan pubsub.SSEEvent)(eventCh), nil)

	// Create notifier repo
	notifierRepo := notifierrepo.NewNotifierRepo(mockPubSubClient, mockNotificationStorageRepo, nil, nil, nil)

	// Test SSESubscribe
	receivedCh, err := notifierRepo.SSESubscribe(ctx, topic)
	assert.NoError(t, err)
	assert.NotNil(t, receivedCh)

	// Simulate receiving an event and closing the channel
	go func() {
		eventCh <- pubsub.SSEEvent{Type: "TestEvent", Data: "TestData"}
		close(eventCh)
	}()

	// Listen on the received channel and test for the expected event
	select {
	case event, ok := <-receivedCh:
		if !ok {
			t.Fatal("Expected channel to be open and receive an event, but it was closed")
		}
		assert.Equal(t, "TestEvent", event.Type)
		assert.Equal(t, "TestData", event.Data)
	case <-time.After(time.Second * 1):
		t.Fatal("Timed out waiting for an event")
	}

	// Ensure the channel eventually gets closed
	_, ok := <-receivedCh
	assert.False(t, ok, "Expected the channel to be closed")

	mockPubSubClient.AssertExpectations(t)
}
