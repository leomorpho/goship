package notifications

import (
	"context"
	"database/sql"
	dbqueries "github.com/leomorpho/goship-modules/notifications/db/queries"
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
	vapidPublicKey, vapidPrivateKey, subscriberEmail string,
) *PwaPushService {
	return newPwaPushService(
		newSQLPwaPushSubscriptionStore(db, dialect),
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
	query, err := dbqueries.Get("insert_pwa_subscription")
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, s.bind(query), now, now, sub.Endpoint, sub.P256dh, sub.Auth, profileID)
	return err
}

func (s *sqlPwaPushSubscriptionStore) listSubscriptions(
	ctx context.Context, profileID int,
) ([]pwaPushSubscriptionRecord, error) {
	query, err := dbqueries.Get("list_pwa_subscriptions_by_profile")
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, s.bind(query), profileID)
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
	query, err := dbqueries.Get("delete_pwa_subscription_by_endpoint")
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, s.bind(query), profileID, endpoint)
	return err
}

func (s *sqlPwaPushSubscriptionStore) hasAnyByProfileID(
	ctx context.Context, profileID int,
) (bool, error) {
	query, err := dbqueries.Get("count_pwa_subscriptions_by_profile")
	if err != nil {
		return false, err
	}
	var count int
	if err := s.db.QueryRowContext(ctx, s.bind(query), profileID).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *sqlPwaPushSubscriptionStore) hasEndpoint(
	ctx context.Context, profileID int, endpoint string,
) (bool, error) {
	query, err := dbqueries.Get("count_pwa_subscription_by_endpoint")
	if err != nil {
		return false, err
	}
	var count int
	if err := s.db.QueryRowContext(ctx, s.bind(query), profileID, endpoint).Scan(&count); err != nil {
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
