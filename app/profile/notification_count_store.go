package profiles

import (
	"context"
	"errors"
)

var ErrNotificationCountStoreNotConfigured = errors.New("notification count store is not configured")

// NotificationCountStore is the data boundary for unseen notifications count lookup.
type NotificationCountStore interface {
	CountUnseenNotifications(ctx context.Context, profileID int) (int, error)
}
