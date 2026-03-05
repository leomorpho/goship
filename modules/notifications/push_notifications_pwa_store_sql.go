package notifications

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"
)

type sqlPwaPushSubscriptionStore struct {
	db         *sql.DB
	postgresql bool
}

func NewSQLPwaPushService(
	db *sql.DB,
	dialect string,
	permissionService *NotificationPermissionService,
	vapidPublicKey, vapidPrivateKey, subscriberEmail string,
) *PwaPushService {
	return newPwaPushService(
		newSQLPwaPushSubscriptionStore(db, dialect),
		permissionService,
		vapidPublicKey,
		vapidPrivateKey,
		subscriberEmail,
	)
}

func newSQLPwaPushSubscriptionStore(db *sql.DB, dialect string) *sqlPwaPushSubscriptionStore {
	d := strings.ToLower(strings.TrimSpace(dialect))
	return &sqlPwaPushSubscriptionStore{
		db:         db,
		postgresql: d == "postgres" || d == "postgresql" || d == "pgx",
	}
}

func (s *sqlPwaPushSubscriptionStore) addSubscription(
	ctx context.Context, profileID int, sub Subscription,
) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, s.bind(`
		INSERT INTO pwa_push_subscriptions (
			created_at, updated_at, endpoint, p256dh, auth, profile_id
		) VALUES (?, ?, ?, ?, ?, ?)
	`), now, now, sub.Endpoint, sub.P256dh, sub.Auth, profileID)
	return err
}

func (s *sqlPwaPushSubscriptionStore) listSubscriptions(
	ctx context.Context, profileID int,
) ([]pwaPushSubscriptionRecord, error) {
	rows, err := s.db.QueryContext(ctx, s.bind(`
		SELECT profile_id, endpoint, p256dh, auth
		FROM pwa_push_subscriptions
		WHERE profile_id = ?
	`), profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]pwaPushSubscriptionRecord, 0)
	for rows.Next() {
		var rec pwaPushSubscriptionRecord
		if err := rows.Scan(&rec.ProfileID, &rec.Endpoint, &rec.P256dh, &rec.Auth); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (s *sqlPwaPushSubscriptionStore) deleteByEndpoint(
	ctx context.Context, profileID int, endpoint string,
) error {
	_, err := s.db.ExecContext(ctx, s.bind(`
		DELETE FROM pwa_push_subscriptions
		WHERE profile_id = ? AND endpoint = ?
	`), profileID, endpoint)
	return err
}

func (s *sqlPwaPushSubscriptionStore) hasAnyByProfileID(
	ctx context.Context, profileID int,
) (bool, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, s.bind(`
		SELECT COUNT(*)
		FROM pwa_push_subscriptions
		WHERE profile_id = ?
	`), profileID).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *sqlPwaPushSubscriptionStore) hasEndpoint(
	ctx context.Context, profileID int, endpoint string,
) (bool, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, s.bind(`
		SELECT COUNT(*)
		FROM pwa_push_subscriptions
		WHERE profile_id = ? AND endpoint = ?
	`), profileID, endpoint).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *sqlPwaPushSubscriptionStore) bind(query string) string {
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
