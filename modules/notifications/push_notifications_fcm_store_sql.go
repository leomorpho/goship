package notifications

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"
)

type sqlFcmPushSubscriptionStore struct {
	db         *sql.DB
	postgresql bool
}

func NewSQLFcmPushService(
	db *sql.DB,
	dialect string,
	permissionService *NotificationPermissionService,
	firebaseJSONAccessKeys *[]byte,
) (*FcmPushService, error) {
	return newFcmPushServiceWithStore(
		newSQLFcmPushSubscriptionStore(db, dialect),
		permissionService,
		firebaseJSONAccessKeys,
	)
}

func newSQLFcmPushSubscriptionStore(db *sql.DB, dialect string) *sqlFcmPushSubscriptionStore {
	d := strings.ToLower(strings.TrimSpace(dialect))
	return &sqlFcmPushSubscriptionStore{
		db:         db,
		postgresql: d == "postgres" || d == "postgresql" || d == "pgx",
	}
}

func (s *sqlFcmPushSubscriptionStore) addSubscription(
	ctx context.Context, profileID int, token string,
) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, s.bind(`
		INSERT INTO fcm_subscriptions (
			created_at, updated_at, token, profile_id
		) VALUES (?, ?, ?, ?)
	`), now, now, token, profileID)
	return err
}

func (s *sqlFcmPushSubscriptionStore) listSubscriptions(
	ctx context.Context, profileID int,
) ([]fcmPushSubscriptionRecord, error) {
	rows, err := s.db.QueryContext(ctx, s.bind(`
		SELECT profile_id, token
		FROM fcm_subscriptions
		WHERE profile_id = ?
	`), profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]fcmPushSubscriptionRecord, 0)
	for rows.Next() {
		var rec fcmPushSubscriptionRecord
		if err := rows.Scan(&rec.ProfileID, &rec.Token); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (s *sqlFcmPushSubscriptionStore) deleteByToken(
	ctx context.Context, profileID int, token string,
) error {
	_, err := s.db.ExecContext(ctx, s.bind(`
		DELETE FROM fcm_subscriptions
		WHERE profile_id = ? AND token = ?
	`), profileID, token)
	return err
}

func (s *sqlFcmPushSubscriptionStore) hasAnyByProfileID(
	ctx context.Context, profileID int,
) (bool, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, s.bind(`
		SELECT COUNT(*)
		FROM fcm_subscriptions
		WHERE profile_id = ?
	`), profileID).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *sqlFcmPushSubscriptionStore) hasToken(
	ctx context.Context, profileID int, token string,
) (bool, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, s.bind(`
		SELECT COUNT(*)
		FROM fcm_subscriptions
		WHERE profile_id = ? AND token = ?
	`), profileID, token).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *sqlFcmPushSubscriptionStore) bind(query string) string {
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
