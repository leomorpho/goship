package profiles

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	dbqueries "github.com/leomorpho/goship/db/queries"
	"strings"
)

var (
	countUnseenNotificationsByProfileIDPostgres = mustQuery("count_unseen_notifications_by_profile_id_postgres")
	countUnseenNotificationsByProfileIDSQLite   = mustQuery("count_unseen_notifications_by_profile_id_sqlite")
)

// SQLNotificationCountStore provides a DB/sql-backed implementation that does not
// depend on generated query builders. This is the bridge path toward Bob-generated stores.
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
		return countUnseenNotificationsByProfileIDPostgres, []any{profileID, false}
	default:
		return countUnseenNotificationsByProfileIDSQLite, []any{profileID, false}
	}
}

func mustQuery(name string) string {
	query, err := dbqueries.Get(name)
	if err != nil {
		panic(err)
	}
	return query
}

func (s *SQLNotificationCountStore) String() string {
	return fmt.Sprintf("SQLNotificationCountStore(dialect=%s)", s.dialect)
}
