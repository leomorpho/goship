package storagerepo

import (
	"context"
	"io"
	"time"

	"github.com/mikestefanello/pagoda/ent"
	"github.com/mikestefanello/pagoda/pkg/domain"
	"github.com/mikestefanello/pagoda/pkg/repos/pubsub"
	"github.com/stretchr/testify/mock"
)

// MockStorageClient is a mock implementation of the StorageClientInterface.
type MockStorageClient struct {
	mock.Mock
}

// NewMockStorageClient creates a new instance of MockStorageClient.
func NewMockStorageClient() *MockStorageClient {
	return &MockStorageClient{}
}

func (msc *MockStorageClient) CreateBucket(bucketName string, location string) error {
	// Implement mock logic, for example, return nil for successful execution.
	return nil
}

func (msc *MockStorageClient) UploadFile(bucket Bucket, objectName string, fileStream io.Reader) (*int, error) {
	// Implement mock logic.
	// You can return a mock file ID and nil error to simulate a successful upload.
	args := msc.Called(bucket, objectName, fileStream)
	return args.Get(0).(*int), args.Error(1)
}

func (msc *MockStorageClient) DeleteFile(bucket Bucket, objectName string) error {
	return nil
}

func (msc *MockStorageClient) GetPresignedURL(bucket Bucket, objectName string, expiry time.Duration) (string, error) {
	// Implement mock logic.
	// Return a mock URL and nil error.
	return "https://mockurl.com/" + objectName, nil
}

func (msc *MockStorageClient) GetImageObjectFromFile(file *ent.Image) (*domain.Photo, error) {
	return nil, nil
}

func (msc *MockStorageClient) GetImageObjectsFromFiles(files []*ent.Image) ([]domain.Photo, error) {
	return nil, nil
}

// MockNotifierRepo is a mock of NotifierRepo interface
type MockNotifierRepo struct {
	mock.Mock
}

func NewMockNotifierRepo() *MockNotifierRepo {
	return &MockNotifierRepo{}
}

func (m *MockNotifierRepo) CreateNotification(ctx context.Context, notification domain.Notification) error {
	args := m.Called(ctx, notification)
	return args.Error(0)
}

func (m *MockNotifierRepo) GetNotifications(ctx context.Context, profileID int, onlyUnread bool) ([]*domain.Notification, error) {
	args := m.Called(ctx, profileID, onlyUnread)
	return args.Get(0).([]*domain.Notification), args.Error(1)
}

func (m *MockNotifierRepo) MarkNotificationRead(ctx context.Context, notificationID int, profileID *int) error {
	args := m.Called(ctx, notificationID)
	return args.Error(0)
}
func (m *MockNotifierRepo) MarkNotificationUnread(ctx context.Context, notificationID int, profileID *int) error {
	args := m.Called(ctx, notificationID)
	return args.Error(0)
}

func (m *MockNotifierRepo) SSESubscribe(ctx context.Context, topic string, handler pubsub.MessageHandler) error {
	args := m.Called(ctx, topic, handler)
	return args.Error(0)
}

func (m *MockNotifierRepo) SSEUnsubscribe(ctx context.Context, topic string) error {
	args := m.Called(ctx, topic)
	return args.Error(0)
}
