package profiles

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// SQLNotificationCountStore provides a DB/sql-backed implementation that does not
// depend on Ent query builders. This is the bridge path toward Bob-generated stores.
type SQLNotificationCountStore struct {
	db      *sql.DB
	dialect string
}

func NewSQLNotificationCountStore(db *sql.DB, dialect string) *SQLNotificationCountStore {
	return &SQLNotificationCountStore{
		db:      db,
		dialect: strings.ToLower(strings.TrimSpace(dialect)),
	}
}

func (s *SQLNotificationCountStore) CountUnseenNotifications(ctx context.Context, profileID int) (int, error) {
	if s == nil || s.db == nil {
		return 0, ErrNotificationCountStoreNotConfigured
	}

	query, args := s.countQuery(profileID)
	var count int
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
}

func (s *SQLNotificationCountStore) countQuery(profileID int) (string, []any) {
	switch s.dialect {
	case "postgres", "postgresql", "pgx":
		return "SELECT COUNT(*) FROM notifications WHERE profile_notifications = $1 AND read = $2", []any{profileID, false}
	default:
		return "SELECT COUNT(*) FROM notifications WHERE profile_notifications = ? AND read = ?", []any{profileID, false}
	}
}

func (s *SQLNotificationCountStore) String() string {
	return fmt.Sprintf("SQLNotificationCountStore(dialect=%s)", s.dialect)
}
