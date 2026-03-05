package gen

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

// QueryRower is the minimal query contract used by generated query helpers.
type QueryRower interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// CountUnseenNotificationsByProfile returns the number of unread notifications
// for a profile. This is the first query path that will be replaced by Bob codegen
// output once db:generate is wired into app runtime usage.
func CountUnseenNotificationsByProfile(
	ctx context.Context,
	db QueryRower,
	dialect string,
	profileID int,
) (int, error) {
	if db == nil {
		return 0, errors.New("query runner is nil")
	}

	query, args := countUnseenNotificationsByProfileQuery(strings.ToLower(strings.TrimSpace(dialect)), profileID)
	var count int
	if err := db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
}

func countUnseenNotificationsByProfileQuery(dialect string, profileID int) (string, []any) {
	switch dialect {
	case "postgres", "postgresql", "pgx":
		return "SELECT COUNT(*) FROM notifications WHERE profile_notifications = $1 AND read = $2", []any{profileID, false}
	default:
		return "SELECT COUNT(*) FROM notifications WHERE profile_notifications = ? AND read = ?", []any{profileID, false}
	}
}
