package gen

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type QueryRower interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type SQLSubscription struct {
	ID               int
	Email            string
	Verified         bool
	ConfirmationCode string
	Lat              sql.NullFloat64
	Lon              sql.NullFloat64
}

func FindListID(ctx context.Context, q QueryRower, dialect string, listName string, onlyActive bool) (int, error) {
	query := `
		SELECT id
		FROM email_subscription_types
		WHERE name = ?
	`
	args := []any{listName}
	if onlyActive {
		query += " AND active = ?"
		args = append(args, true)
	}

	var id int
	err := q.QueryRowContext(ctx, bind(query, dialect), args...).Scan(&id)
	return id, err
}

func InsertList(ctx context.Context, e Execer, dialect string, listName string, now time.Time) error {
	_, err := e.ExecContext(ctx, bind(`
		INSERT INTO email_subscription_types (created_at, updated_at, name, active)
		VALUES (?, ?, ?, ?)
	`, dialect), now, now, listName, true)
	return err
}

func FindSubscriptionByEmail(ctx context.Context, q QueryRower, dialect string, email string) (SQLSubscription, error) {
	var sub SQLSubscription
	err := q.QueryRowContext(ctx, bind(`
		SELECT id, email, verified, confirmation_code, latitude, longitude
		FROM email_subscriptions
		WHERE email = ?
		LIMIT 1
	`, dialect), email).Scan(&sub.ID, &sub.Email, &sub.Verified, &sub.ConfirmationCode, &sub.Lat, &sub.Lon)
	return sub, err
}

func InsertSubscription(
	ctx context.Context, e Execer, dialect string, now time.Time, email string, confirmationCode string, lat sql.NullFloat64, lon sql.NullFloat64,
) (int, error) {
	res, err := e.ExecContext(ctx, bind(`
		INSERT INTO email_subscriptions (created_at, updated_at, email, verified, confirmation_code, latitude, longitude)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, dialect), now, now, email, false, confirmationCode, lat, lon)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func FindSubscriptionLink(ctx context.Context, q QueryRower, dialect string, subscriptionID, listTypeID int) (bool, error) {
	var found int
	err := q.QueryRowContext(ctx, bind(`
		SELECT 1
		FROM email_subscription_subscriptions
		WHERE email_subscription_id = ? AND email_subscription_type_id = ?
		LIMIT 1
	`, dialect), subscriptionID, listTypeID).Scan(&found)
	switch {
	case err == sql.ErrNoRows:
		return false, nil
	case err != nil:
		return false, err
	default:
		return true, nil
	}
}

func InsertSubscriptionLink(ctx context.Context, e Execer, dialect string, subscriptionID, listTypeID int) error {
	_, err := e.ExecContext(ctx, bind(`
		INSERT INTO email_subscription_subscriptions (email_subscription_id, email_subscription_type_id)
		VALUES (?, ?)
	`, dialect), subscriptionID, listTypeID)
	return err
}

func UpdateSubscriptionLocation(
	ctx context.Context, e Execer, dialect string, now time.Time, subscriptionID int, latitude float64, longitude float64,
) error {
	_, err := e.ExecContext(ctx, bind(`
		UPDATE email_subscriptions
		SET updated_at = ?, latitude = ?, longitude = ?
		WHERE id = ?
	`, dialect), now, latitude, longitude, subscriptionID)
	return err
}

func DeleteSubscriptionLink(ctx context.Context, e Execer, dialect string, subscriptionID, listTypeID int) error {
	_, err := e.ExecContext(ctx, bind(`
		DELETE FROM email_subscription_subscriptions
		WHERE email_subscription_id = ? AND email_subscription_type_id = ?
	`, dialect), subscriptionID, listTypeID)
	return err
}

func CountSubscriptionLinks(ctx context.Context, q QueryRower, dialect string, subscriptionID int) (int, error) {
	var count int
	err := q.QueryRowContext(ctx, bind(`
		SELECT COUNT(*)
		FROM email_subscription_subscriptions
		WHERE email_subscription_id = ?
	`, dialect), subscriptionID).Scan(&count)
	return count, err
}

func DeleteSubscriptionByID(ctx context.Context, e Execer, dialect string, subscriptionID int) error {
	_, err := e.ExecContext(ctx, bind(`DELETE FROM email_subscriptions WHERE id = ?`, dialect), subscriptionID)
	return err
}

func RotateSubscriptionCode(ctx context.Context, e Execer, dialect string, now time.Time, subscriptionID int, code string) error {
	_, err := e.ExecContext(ctx, bind(`
		UPDATE email_subscriptions
		SET updated_at = ?, confirmation_code = ?
		WHERE id = ?
	`, dialect), now, code, subscriptionID)
	return err
}

func FindSubscriptionByConfirmationCode(ctx context.Context, q QueryRower, dialect string, code string) (int, bool, error) {
	var id int
	var verified bool
	err := q.QueryRowContext(ctx, bind(`
		SELECT id, verified
		FROM email_subscriptions
		WHERE confirmation_code = ?
		LIMIT 1
	`, dialect), code).Scan(&id, &verified)
	return id, verified, err
}

func MarkSubscriptionVerified(ctx context.Context, e Execer, dialect string, now time.Time, subscriptionID int, code string) error {
	_, err := e.ExecContext(ctx, bind(`
		UPDATE email_subscriptions
		SET updated_at = ?, verified = ?, confirmation_code = ?
		WHERE id = ?
	`, dialect), now, true, code, subscriptionID)
	return err
}

func bind(query, dialect string) string {
	normalized := strings.ToLower(strings.TrimSpace(dialect))
	if normalized != "postgres" && normalized != "postgresql" && normalized != "pgx" {
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
