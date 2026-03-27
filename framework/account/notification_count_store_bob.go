package profiles

import (
	"context"
	"database/sql"
	"strings"

	dbgen "github.com/leomorpho/goship/db/gen"
)

// BobNotificationCountStore uses db/gen query wrappers (Bob migration path).
type BobNotificationCountStore struct {
	db      *sql.DB
	dialect string
}

func NewBobNotificationCountStore(db *sql.DB, dialect string) *BobNotificationCountStore {
	return &BobNotificationCountStore{
		db:      db,
		dialect: strings.ToLower(strings.TrimSpace(dialect)),
	}
}

func (s *BobNotificationCountStore) CountUnseenNotifications(ctx context.Context, profileID int) (int, error) {
	if s == nil || s.db == nil {
		return 0, ErrNotificationCountStoreNotConfigured
	}
	return dbgen.CountUnseenNotificationsByProfile(ctx, s.db, s.dialect, profileID)
}
