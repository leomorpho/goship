package profiles

import (
	"context"
	"errors"

	"github.com/leomorpho/goship/db/ent"
	"github.com/leomorpho/goship/db/ent/notification"
	"github.com/leomorpho/goship/db/ent/profile"
)

var ErrNotificationCountStoreNotConfigured = errors.New("notification count store is not configured")

// NotificationCountStore is the data boundary for unseen notifications count lookup.
// It isolates profile service read paths from direct ORM-specific query code.
type NotificationCountStore interface {
	CountUnseenNotifications(ctx context.Context, profileID int) (int, error)
}

type EntNotificationCountStore struct {
	orm *ent.Client
}

func NewEntNotificationCountStore(orm *ent.Client) *EntNotificationCountStore {
	return &EntNotificationCountStore{orm: orm}
}

func (s *EntNotificationCountStore) CountUnseenNotifications(ctx context.Context, profileID int) (int, error) {
	if s == nil || s.orm == nil {
		return 0, ErrNotificationCountStoreNotConfigured
	}
	return s.orm.Notification.Query().
		Where(
			notification.HasProfileWith(profile.IDEQ(profileID)),
			notification.ReadEQ(false),
		).Count(ctx)
}
