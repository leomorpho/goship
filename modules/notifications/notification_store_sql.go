package notifications

import (
	"context"
	"database/sql"
	"fmt"
	dbqueries "github.com/leomorpho/goship-modules/notifications/db/queries"
	"strings"
	"time"

	dbmigrate "github.com/leomorpho/goship-modules/notifications/db/migrate"
	"github.com/leomorpho/goship/framework/domain"
)

type SQLNotificationStore struct {
	db         *sql.DB
	postgresql bool
}

// NotificationStorage defines storage operations on notifications.
type NotificationStorage interface {
	CreateNotification(ctx context.Context, n domain.Notification) (*domain.Notification, error)
	GetNotificationsByProfileID(ctx context.Context, profileID int, onlyUnread bool, beforeTimestamp *time.Time, pageSize *int) ([]*domain.Notification, error)
	MarkNotificationAsRead(ctx context.Context, notificationID int, profileID *int) error
	MarkAllNotificationAsRead(ctx context.Context, profileID int) error
	MarkNotificationAsUnread(ctx context.Context, notificationID int, profileID *int) error
	DeleteNotification(ctx context.Context, notificationID int, profileID *int) error
	HasNotificationForResourceAndPerson(ctx context.Context, notifType domain.NotificationType, profileIDWhoCausedNotif, resourceID *int, maxAge time.Duration) (exists bool, err error)
}

func NewSQLNotificationStore(db *sql.DB, dialect string) *SQLNotificationStore {
	d := strings.ToLower(strings.TrimSpace(dialect))
	return &SQLNotificationStore{
		db:         db,
		postgresql: d == "postgres" || d == "postgresql" || d == "pgx",
	}
}

func NewSQLNotificationStoreWithSchema(db *sql.DB, dialect string) (*SQLNotificationStore, error) {
	store := NewSQLNotificationStore(db, dialect)
	if err := store.ensureSchema(context.Background()); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *SQLNotificationStore) ensureSchema(ctx context.Context) error {
	ddl, err := dbmigrate.LoadInitNotificationsUpSQL()
	if err != nil {
		return err
	}
	if !s.postgresql {
		// SQLite compatibility for module migration DDL.
		ddl = strings.ReplaceAll(ddl, "BIGSERIAL PRIMARY KEY", "INTEGER PRIMARY KEY AUTOINCREMENT")
		ddl = strings.ReplaceAll(ddl, "TIMESTAMPTZ", "TIMESTAMP")
	}
	_, err = s.db.ExecContext(ctx, ddl)
	return err
}

func (s *SQLNotificationStore) CreateNotification(ctx context.Context, n domain.Notification) (*domain.Notification, error) {
	now := time.Now().UTC()
	query, err := dbqueries.Get("insert_notification")
	if err != nil {
		return nil, err
	}
	result, err := s.db.ExecContext(ctx, s.bind(query),
		now,
		now,
		n.Type.Value,
		n.Title,
		n.Text,
		nullableString(n.Link),
		false,
		nil,
		n.ProfileID,
		n.ProfileIDWhoCausedNotif,
		n.ResourceIDTiedToNotif,
		n.ReadInNotificationsCenter,
	)
	if err != nil {
		return nil, err
	}
	id64, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	created := n
	created.ID = int(id64)
	created.CreatedAt = now
	created.Read = false
	return &created, nil
}

func (s *SQLNotificationStore) GetNotificationsByProfileID(
	ctx context.Context, profileID int, onlyUnread bool, beforeTimestamp *time.Time, pageSize *int,
) ([]*domain.Notification, error) {
	query, err := dbqueries.Get("select_notifications_by_profile_base")
	if err != nil {
		return nil, err
	}
	args := []any{profileID}
	if onlyUnread {
		query += " AND read = ?"
		args = append(args, false)
	}
	if beforeTimestamp != nil {
		query += " AND created_at < ?"
		args = append(args, beforeTimestamp.UTC())
	}
	query += " ORDER BY created_at DESC"
	if pageSize != nil {
		query += " LIMIT ?"
		args = append(args, *pageSize)
	}

	rows, err := s.db.QueryContext(ctx, s.bind(query), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*domain.Notification, 0)
	for rows.Next() {
		var (
			n               domain.Notification
			notifTypeRaw    string
			link            sql.NullString
			readAt          sql.NullTime
			readInCenter    bool
			createdAt       time.Time
			profileCauserID int
			resourceID      int
		)
		if err := rows.Scan(
			&n.ID,
			&notifTypeRaw,
			&n.Title,
			&n.Text,
			&link,
			&createdAt,
			&n.Read,
			&readAt,
			&n.ProfileID,
			&profileCauserID,
			&resourceID,
			&readInCenter,
		); err != nil {
			return nil, err
		}
		parsed := domain.NotificationTypes.Parse(notifTypeRaw)
		if parsed == nil {
			continue
		}
		n.Type = *parsed
		n.CreatedAt = createdAt
		n.ProfileIDWhoCausedNotif = profileCauserID
		n.ResourceIDTiedToNotif = resourceID
		n.ReadInNotificationsCenter = readInCenter
		if link.Valid {
			n.Link = link.String
		}
		if readAt.Valid {
			n.ReadAt = readAt.Time
		}
		out = append(out, &n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *SQLNotificationStore) MarkNotificationAsRead(ctx context.Context, notificationID int, profileID *int) error {
	if profileID != nil {
		if err := s.checkNotificationBelongsToProfile(ctx, *profileID, notificationID); err != nil {
			return err
		}
	}

	var notifTypeRaw string
	query, err := dbqueries.Get("select_notification_type_by_id")
	if err != nil {
		return err
	}
	err = s.db.QueryRowContext(ctx, s.bind(query), notificationID).Scan(&notifTypeRaw)
	if err != nil {
		return err
	}

	parsed := domain.NotificationTypes.Parse(notifTypeRaw)
	if parsed != nil && domain.DeleteOnceReadNotificationTypesMap[*parsed] {
		deleteQuery, lookupErr := dbqueries.Get("delete_notification_by_id")
		if lookupErr != nil {
			return lookupErr
		}
		_, err = s.db.ExecContext(ctx, s.bind(deleteQuery), notificationID)
		return err
	}
	updateQuery, lookupErr := dbqueries.Get("mark_notification_read_by_id")
	if lookupErr != nil {
		return lookupErr
	}
	_, err = s.db.ExecContext(ctx, s.bind(updateQuery), true, time.Now().UTC(), time.Now().UTC(), notificationID)
	return err
}

func (s *SQLNotificationStore) MarkAllNotificationAsRead(ctx context.Context, profileID int) error {
	query, err := dbqueries.Get("mark_all_notifications_read_by_profile")
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, s.bind(query), true, time.Now().UTC(), time.Now().UTC(), profileID)
	return err
}

func (s *SQLNotificationStore) MarkNotificationAsUnread(ctx context.Context, notificationID int, profileID *int) error {
	if profileID != nil {
		if err := s.checkNotificationBelongsToProfile(ctx, *profileID, notificationID); err != nil {
			return err
		}
	}
	query, err := dbqueries.Get("mark_notification_unread_by_id")
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, s.bind(query), false, time.Now().UTC(), notificationID)
	return err
}

func (s *SQLNotificationStore) DeleteNotification(ctx context.Context, notificationID int, profileID *int) error {
	if profileID != nil {
		if err := s.checkNotificationBelongsToProfile(ctx, *profileID, notificationID); err != nil {
			return err
		}
	}
	query, err := dbqueries.Get("delete_notification_by_id")
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, s.bind(query), notificationID)
	return err
}

func (s *SQLNotificationStore) HasNotificationForResourceAndPerson(
	ctx context.Context, notifType domain.NotificationType, profileIDWhoCausedNotif, resourceID *int, maxAge time.Duration,
) (bool, error) {
	query, err := dbqueries.Get("count_notifications_for_type_since_base")
	if err != nil {
		return false, err
	}
	args := []any{notifType.Value, time.Now().UTC().Add(-maxAge)}
	if profileIDWhoCausedNotif != nil {
		query += " AND profile_id_who_caused_notification = ?"
		args = append(args, *profileIDWhoCausedNotif)
	}
	if resourceID != nil {
		query += " AND resource_id_tied_to_notif = ?"
		args = append(args, *resourceID)
	}
	var count int
	if err := s.db.QueryRowContext(ctx, s.bind(query), args...).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *SQLNotificationStore) checkNotificationBelongsToProfile(
	ctx context.Context, profileID, notificationID int,
) error {
	query, err := dbqueries.Get("count_notification_belongs_to_profile")
	if err != nil {
		return err
	}
	var count int
	if err := s.db.QueryRowContext(ctx, s.bind(query), notificationID, profileID).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("notification does not belong to the provided profile")
	}
	return nil
}

func (s *SQLNotificationStore) bind(query string) string {
	if !s.postgresql {
		return query
	}
	var b strings.Builder
	index := 1
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			b.WriteString(fmt.Sprintf("$%d", index))
			index++
			continue
		}
		b.WriteByte(query[i])
	}
	return b.String()
}

func nullableString(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}
