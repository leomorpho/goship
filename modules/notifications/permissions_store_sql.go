package notifications

import (
	"context"
	"database/sql"
	dbqueries "github.com/leomorpho/goship-modules/notifications/db/queries"
	"strconv"
	"strings"
	"time"

	"github.com/leomorpho/goship/framework/domain"
)

type sqlNotificationPermissionStore struct {
	db         *sql.DB
	postgresql bool
}

func NewSQLNotificationPermissionService(db *sql.DB, dialect string) *NotificationPermissionService {
	return NewNotificationPermissionServiceWithStore(newSQLNotificationPermissionStore(db, dialect))
}

func newSQLNotificationPermissionStore(db *sql.DB, dialect string) *sqlNotificationPermissionStore {
	d := strings.ToLower(strings.TrimSpace(dialect))
	return &sqlNotificationPermissionStore{
		db:         db,
		postgresql: d == "postgres" || d == "postgresql" || d == "pgx",
	}
}

func (s *sqlNotificationPermissionStore) deleteAllPermissions(
	ctx context.Context, profileID int, platform *domain.NotificationPlatform,
) error {
	query, err := dbqueries.Get("delete_permissions_by_profile_base")
	if err != nil {
		return err
	}
	args := []any{profileID}
	if platform != nil {
		query += " AND platform = ?"
		args = append(args, platform.Value)
	}
	_, err = s.db.ExecContext(ctx, s.bind(query), args...)
	return err
}

func (s *sqlNotificationPermissionStore) listPermissionsByProfileID(
	ctx context.Context, profileID int,
) ([]notificationPermissionRecord, error) {
	query, err := dbqueries.Get("list_permissions_by_profile")
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, s.bind(query), profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]notificationPermissionRecord, 0)
	for rows.Next() {
		var rec notificationPermissionRecord
		if err := rows.Scan(&rec.Permission, &rec.Platform, &rec.Token); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (s *sqlNotificationPermissionStore) createPermission(
	ctx context.Context, profileID int, permission domain.NotificationPermissionType, platform domain.NotificationPlatform, token string,
) error {
	now := time.Now().UTC()
	query, err := dbqueries.Get("insert_or_upsert_permission")
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, s.bind(query), now, now, permission.Value, platform.Value, profileID, token)
	return err
}

func (s *sqlNotificationPermissionStore) deletePermission(
	ctx context.Context,
	profileID int,
	permission domain.NotificationPermissionType,
	platform *domain.NotificationPlatform,
	token *string,
) error {
	query, err := dbqueries.Get("delete_permission_base")
	if err != nil {
		return err
	}
	args := []any{profileID, permission.Value}
	if token != nil && strings.TrimSpace(*token) != "" {
		query += " AND token = ?"
		args = append(args, *token)
	}
	if platform != nil {
		query += " AND platform = ?"
		args = append(args, platform.Value)
	}

	res, err := s.db.ExecContext(ctx, s.bind(query), args...)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *sqlNotificationPermissionStore) countPermissionsForPlatform(
	ctx context.Context, profileID int, platform domain.NotificationPlatform,
) (int, error) {
	query, err := dbqueries.Get("count_permissions_for_platform")
	if err != nil {
		return 0, err
	}
	var count int
	err = s.db.QueryRowContext(ctx, s.bind(query), profileID, platform.Value).Scan(&count)
	return count, err
}

func (s *sqlNotificationPermissionStore) bind(query string) string {
	if !s.postgresql || strings.Count(query, "?") == 0 {
		return query
	}
	var b strings.Builder
	b.Grow(len(query) + 8)
	arg := 1
	for _, r := range query {
		if r == '?' {
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(arg))
			arg++
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
