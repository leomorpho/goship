//go:build integration

package notifications_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/leomorpho/goship-modules/notifications"
	"github.com/leomorpho/goship/framework/core"
	"github.com/leomorpho/goship/framework/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockPubSubClient struct {
	mock.Mock
}

func (m *MockPubSubClient) Publish(ctx context.Context, topic string, payload []byte) error {
	args := m.Called(ctx, topic, payload)
	return args.Error(0)
}

func (m *MockPubSubClient) Subscribe(ctx context.Context, topic string, handler core.MessageHandler) (core.Subscription, error) {
	args := m.Called(ctx, topic, handler)
	return args.Get(0).(core.Subscription), args.Error(1)
}

func (m *MockPubSubClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockPubSubClient) DeleteNotification(ctx context.Context, notificationID int) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type mockSubscription struct {
	mock.Mock
}

func (s *mockSubscription) Close() error {
	args := s.Called()
	return args.Error(0)
}

type MockNotificationStore struct {
	mock.Mock
}

func (m *MockNotificationStore) CreateNotification(
	ctx context.Context, n domain.Notification,
) (*domain.Notification, error) {
	args := m.Called(ctx, n)
	return args.Get(0).(*domain.Notification), args.Error(1)
}

func (m *MockNotificationStore) GetNotificationsByProfileID(
	ctx context.Context, profileID int, onlyUnread bool, beforeTimestamp *time.Time, pageSize *int,
) ([]*domain.Notification, error) {
	args := m.Called(ctx, profileID, onlyUnread)
	return args.Get(0).([]*domain.Notification), args.Error(1)
}

func (m *MockNotificationStore) MarkNotificationAsRead(
	ctx context.Context, notificationID int, profileID *int,
) error {
	args := m.Called(ctx, notificationID)
	return args.Error(0)
}

func (m *MockNotificationStore) MarkAllNotificationAsRead(
	ctx context.Context, profileID int,
) error {
	args := m.Called(ctx, profileID)
	return args.Error(0)
}

func (m *MockNotificationStore) MarkNotificationAsUnread(
	ctx context.Context, notificationID int, profileID *int,
) error {
	args := m.Called(ctx, notificationID)
	return args.Error(0)
}

func (m *MockNotificationStore) DeleteNotification(
	ctx context.Context, notificationID int, profileID *int,
) error {
	args := m.Called(ctx, notificationID)
	return args.Error(0)
}

func (m *MockNotificationStore) HasNotificationForResourceAndPerson(
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
			mockNotificationStore := new(MockNotificationStore)

			// Setting expectations based on the test case
			if tc.expectCreate {
				mockNotificationStore.
					On("CreateNotification", ctx, notification).
					Return(&notification, nil)
			}
			mockPubSubClient.
				On("Publish", ctx, fmt.Sprint(notification.ProfileID), mock.Anything).
				Return(nil)

			// Create notifier repo
			notifierService := notifications.NewNotifierService(mockPubSubClient, mockNotificationStore, nil, nil, nil)

			// Test CreateNotification
			err := notifierService.PublishNotification(ctx, notification, tc.storeInDB, false)
			assert.NoError(t, err)

			if tc.expectCreate {
				mockNotificationStore.AssertExpectations(t)
			}
			mockPubSubClient.AssertExpectations(t)
		})
	}
}

func TestGetNotifications(t *testing.T) {
	ctx := context.Background()
	mockPubSubClient := new(MockPubSubClient)
	mockNotificationStore := new(MockNotificationStore)
	profileID := 1
	notificationList := []*domain.Notification{{ID: 1, ProfileID: profileID, Type: domain.NotificationTypeNewPrivateMessage, Text: "Test Notification"}}

	// Set expectations
	mockNotificationStore.On("GetNotificationsByProfileID", ctx, profileID, false).Return(notificationList, nil)

	// Create notifier repo
	notifierService := notifications.NewNotifierService(mockPubSubClient, mockNotificationStore, nil, nil, nil)

	// Test GetNotifications
	result, err := notifierService.GetNotifications(ctx, profileID, false, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, notificationList, result)

	mockNotificationStore.AssertExpectations(t)
}

func TestMarkNotificationRead(t *testing.T) {
	ctx := context.Background()
	mockPubSubClient := new(MockPubSubClient)
	mockNotificationStore := new(MockNotificationStore)
	notificationID := 1

	// Set expectations
	mockNotificationStore.On("MarkNotificationAsRead", ctx, notificationID).Return(nil)

	// Create notifier repo
	notifierService := notifications.NewNotifierService(mockPubSubClient, mockNotificationStore, nil, nil, nil)

	// Test MarkNotificationRead
	err := notifierService.MarkNotificationRead(ctx, notificationID, nil)
	assert.NoError(t, err)

	mockNotificationStore.AssertExpectations(t)
}

func TestSubscribe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure the context gets cancelled to clean up resources

	mockPubSubClient := new(MockPubSubClient)
	mockNotificationStore := new(MockNotificationStore)
	topic := "someTopic"
	mockSub := new(mockSubscription)

	evt := notifications.SSEEvent{Type: "TestEvent", Data: "TestData"}
	payload, err := json.Marshal(evt)
	assert.NoError(t, err)
	mockSub.On("Close").Return(nil).Maybe()
	mockPubSubClient.
		On("Subscribe", mock.Anything, topic, mock.Anything).
		Run(func(args mock.Arguments) {
			handler := args.Get(2).(core.MessageHandler)
			go func() {
				_ = handler(ctx, topic, payload)
			}()
		}).
		Return(core.Subscription(mockSub), nil)

	// Create notifier repo
	notifierService := notifications.NewNotifierService(mockPubSubClient, mockNotificationStore, nil, nil, nil)

	// Test SSESubscribe
	receivedCh, err := notifierService.SSESubscribe(ctx, topic)
	assert.NoError(t, err)
	assert.NotNil(t, receivedCh)

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

	mockPubSubClient.AssertExpectations(t)
	mockSub.AssertExpectations(t)
}
