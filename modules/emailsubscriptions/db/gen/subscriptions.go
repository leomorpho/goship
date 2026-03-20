package gen

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/leomorpho/goship-modules/emailsubscriptions/db/queries"
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
	query, err := queries.Get("find_list_id_base")
	if err != nil {
		return 0, err
	}
	args := []any{listName}
	if onlyActive {
		activeSuffix, lookupErr := queries.Get("find_list_id_active_suffix")
		if lookupErr != nil {
			return 0, lookupErr
		}
		query += "\n" + activeSuffix
		args = append(args, true)
	}

	var id int
	err = q.QueryRowContext(ctx, bind(query, dialect), args...).Scan(&id)
	return id, err
}

func InsertList(ctx context.Context, e Execer, dialect string, listName string, now time.Time) error {
	query, err := queries.Get("insert_list")
	if err != nil {
		return err
	}
	_, err = e.ExecContext(ctx, bind(query, dialect), now, now, listName, true)
	return err
}

func FindSubscriptionByEmail(ctx context.Context, q QueryRower, dialect string, email string) (SQLSubscription, error) {
	var sub SQLSubscription
	query, err := queries.Get("find_subscription_by_email")
	if err != nil {
		return sub, err
	}
	err = q.QueryRowContext(ctx, bind(query, dialect), email).Scan(&sub.ID, &sub.Email, &sub.Verified, &sub.ConfirmationCode, &sub.Lat, &sub.Lon)
	return sub, err
}

func FindSubscriptionByEmailAndCode(ctx context.Context, q QueryRower, dialect string, email, code string) (SQLSubscription, error) {
	var sub SQLSubscription
	query, err := queries.Get("find_subscription_by_email_and_code")
	if err != nil {
		return sub, err
	}
	err = q.QueryRowContext(ctx, bind(query, dialect), email, code).Scan(&sub.ID, &sub.Email, &sub.Verified, &sub.ConfirmationCode, &sub.Lat, &sub.Lon)
	return sub, err
}

func InsertSubscription(
	ctx context.Context, e Execer, dialect string, now time.Time, email string, confirmationCode string, lat sql.NullFloat64, lon sql.NullFloat64,
) (int, error) {
	query, err := queries.Get("insert_subscription")
	if err != nil {
		return 0, err
	}
	res, err := e.ExecContext(ctx, bind(query, dialect), now, now, email, false, confirmationCode, lat, lon)
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
	query, err := queries.Get("find_subscription_link")
	if err != nil {
		return false, err
	}
	var found int
	err = q.QueryRowContext(ctx, bind(query, dialect), subscriptionID, listTypeID).Scan(&found)
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
	query, err := queries.Get("insert_subscription_link")
	if err != nil {
		return err
	}
	_, err = e.ExecContext(ctx, bind(query, dialect), subscriptionID, listTypeID)
	return err
}

func UpdateSubscriptionLocation(
	ctx context.Context, e Execer, dialect string, now time.Time, subscriptionID int, latitude float64, longitude float64,
) error {
	query, err := queries.Get("update_subscription_location")
	if err != nil {
		return err
	}
	_, err = e.ExecContext(ctx, bind(query, dialect), now, latitude, longitude, subscriptionID)
	return err
}

func DeleteSubscriptionLink(ctx context.Context, e Execer, dialect string, subscriptionID, listTypeID int) error {
	query, err := queries.Get("delete_subscription_link")
	if err != nil {
		return err
	}
	_, err = e.ExecContext(ctx, bind(query, dialect), subscriptionID, listTypeID)
	return err
}

func CountSubscriptionLinks(ctx context.Context, q QueryRower, dialect string, subscriptionID int) (int, error) {
	query, err := queries.Get("count_subscription_links")
	if err != nil {
		return 0, err
	}
	var count int
	err = q.QueryRowContext(ctx, bind(query, dialect), subscriptionID).Scan(&count)
	return count, err
}

func DeleteSubscriptionByID(ctx context.Context, e Execer, dialect string, subscriptionID int) error {
	query, err := queries.Get("delete_subscription_by_id")
	if err != nil {
		return err
	}
	_, err = e.ExecContext(ctx, bind(query, dialect), subscriptionID)
	return err
}

func RotateSubscriptionCode(ctx context.Context, e Execer, dialect string, now time.Time, subscriptionID int, code string) error {
	query, err := queries.Get("rotate_subscription_code")
	if err != nil {
		return err
	}
	_, err = e.ExecContext(ctx, bind(query, dialect), now, code, subscriptionID)
	return err
}

func FindSubscriptionByConfirmationCode(ctx context.Context, q QueryRower, dialect string, code string) (int, bool, error) {
	query, err := queries.Get("find_subscription_by_confirmation_code")
	if err != nil {
		return 0, false, err
	}
	var id int
	var verified bool
	err = q.QueryRowContext(ctx, bind(query, dialect), code).Scan(&id, &verified)
	return id, verified, err
}

func MarkSubscriptionVerified(ctx context.Context, e Execer, dialect string, now time.Time, subscriptionID int, code string) error {
	query, err := queries.Get("mark_subscription_verified")
	if err != nil {
		return err
	}
	_, err = e.ExecContext(ctx, bind(query, dialect), now, true, code, subscriptionID)
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
