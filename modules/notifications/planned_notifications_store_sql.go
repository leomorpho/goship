package notifications

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	"github.com/leomorpho/goship/framework/domain"
)

type sqlPlannedNotificationStore struct {
	db         *sql.DB
	postgresql bool
}

func NewSQLPlannedNotificationsService(
	db *sql.DB, dialect string, subscriptionRepo *paidsubscriptions.Service,
) *PlannedNotificationsService {
	return NewPlannedNotificationsServiceWithStore(
		newSQLPlannedNotificationStore(db, dialect),
		subscriptionRepo,
	)
}

func newSQLPlannedNotificationStore(db *sql.DB, dialect string) *sqlPlannedNotificationStore {
	d := strings.ToLower(strings.TrimSpace(dialect))
	return &sqlPlannedNotificationStore{
		db:         db,
		postgresql: d == "postgres" || d == "postgresql" || d == "pgx",
	}
}

func (s *sqlPlannedNotificationStore) listProfilesForPermission(
	ctx context.Context, permission domain.NotificationPermissionType, notifType domain.NotificationType,
) ([]plannedNotificationCandidate, error) {
	rows, err := s.db.QueryContext(ctx, s.bind(`
		SELECT np.profile_id, MAX(nt.updated_at) AS nt_updated_at
		FROM notification_permissions np
		LEFT JOIN notification_times nt
		  ON nt.profile_id = np.profile_id AND nt.type = ?
		WHERE np.permission = ?
		GROUP BY np.profile_id
	`), notifType.Value, permission.Value)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]plannedNotificationCandidate, 0)
	for rows.Next() {
		var (
			c         plannedNotificationCandidate
			updatedAt sql.NullTime
		)
		if err := rows.Scan(&c.ProfileID, &updatedAt); err != nil {
			return nil, err
		}
		if updatedAt.Valid {
			t := updatedAt.Time
			c.NotificationTimeUpdatedAt = &t
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *sqlPlannedNotificationStore) deleteStaleLastSeenBefore(
	ctx context.Context, deleteBeforeTime time.Time,
) error {
	_, err := s.db.ExecContext(ctx, s.bind(`
		DELETE FROM last_seen_onlines
		WHERE seen_at <= ?
	`), deleteBeforeTime)
	return err
}

func (s *sqlPlannedNotificationStore) listLastSeenForProfile(
	ctx context.Context, profileID int,
) ([]time.Time, error) {
	rows, err := s.db.QueryContext(ctx, s.bind(`
		SELECT lso.seen_at
		FROM last_seen_onlines lso
		JOIN users u ON lso.user_last_seen_at = u.id
		JOIN profiles p ON p.user_profile = u.id
		WHERE p.id = ?
	`), profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]time.Time, 0)
	for rows.Next() {
		var seenAt time.Time
		if err := rows.Scan(&seenAt); err != nil {
			return nil, err
		}
		out = append(out, seenAt)
	}
	return out, rows.Err()
}

func (s *sqlPlannedNotificationStore) upsertNotificationTime(
	ctx context.Context, profileID int, notificationType domain.NotificationType, sendMinute int,
) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, s.bind(`
		INSERT INTO notification_times (
			created_at, updated_at, type, send_minute, profile_id
		) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(profile_id, type) DO UPDATE SET
			send_minute = excluded.send_minute,
			updated_at = excluded.updated_at
	`), now, now, notificationType.Value, sendMinute, profileID)
	return err
}

func (s *sqlPlannedNotificationStore) listProfileIDsCanGetPlannedNotificationNow(
	ctx context.Context, notifType domain.NotificationType, prevMidnightTimestamp time.Time, timestampMinutesFromMidnight int, profileIDs *[]int,
) ([]int, error) {
	var args []any
	q := `
		SELECT nt.profile_id
		FROM notification_times nt
		WHERE nt.type = ?
		  AND nt.send_minute >= 0
		  AND nt.send_minute <= ?
		  AND NOT EXISTS (
			  SELECT 1
			  FROM notifications n
			  WHERE n.profile_id = nt.profile_id
			    AND n.created_at >= ?
			    AND n.type = ?
		  )
	`
	args = append(args, notifType.Value, timestampMinutesFromMidnight, prevMidnightTimestamp, notifType.Value)

	if profileIDs != nil && len(*profileIDs) > 0 {
		q += " AND nt.profile_id IN (" + inListPlaceholders(len(*profileIDs)) + ")"
		for _, id := range *profileIDs {
			args = append(args, id)
		}
	}
	q += " ORDER BY nt.profile_id"

	rows, err := s.db.QueryContext(ctx, s.bind(q), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]int, 0)
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *sqlPlannedNotificationStore) bind(query string) string {
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

func inListPlaceholders(n int) string {
	if n <= 0 {
		return ""
	}
	var b strings.Builder
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteByte('?')
	}
	return b.String()
}
