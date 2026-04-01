package notifications

import (
	"context"
	"database/sql"
	dbqueries "github.com/leomorpho/goship/v2-modules/notifications/db/queries"
	"strconv"
	"strings"
	"time"

	paidsubscriptions "github.com/leomorpho/goship/v2-modules/paidsubscriptions"
	"github.com/leomorpho/goship/v2/framework/domain"
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
	ctx context.Context, permission PermissionType, notifType domain.NotificationType,
) ([]plannedNotificationCandidate, error) {
	query, err := dbqueries.Get("list_profiles_for_permission")
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, s.bind(query), notifType.Value, permission.Value)
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
	query, err := dbqueries.Get("delete_stale_last_seen_before")
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, s.bind(query), deleteBeforeTime)
	return err
}

func (s *sqlPlannedNotificationStore) listLastSeenForProfile(
	ctx context.Context, profileID int,
) ([]time.Time, error) {
	query, err := dbqueries.Get("list_last_seen_for_profile")
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, s.bind(query), profileID)
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
	query, err := dbqueries.Get("upsert_notification_time")
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, s.bind(query), now, now, notificationType.Value, sendMinute, profileID)
	return err
}

func (s *sqlPlannedNotificationStore) listProfileIDsCanGetPlannedNotificationNow(
	ctx context.Context, notifType domain.NotificationType, prevMidnightTimestamp time.Time, timestampMinutesFromMidnight int, profileIDs *[]int,
) ([]int, error) {
	var args []any
	q, err := dbqueries.Get("list_profile_ids_can_get_planned_notification_now_base")
	if err != nil {
		return nil, err
	}
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
