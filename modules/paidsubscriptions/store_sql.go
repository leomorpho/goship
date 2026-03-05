package paidsubscriptions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type SQLStore struct {
	db                             *sql.DB
	postgresQL                     bool
	proTrialTimespan               time.Duration
	paymentFailedGracePeriodInDays time.Duration
}

func NewSQLStore(db *sql.DB, dialect string, proTrialTimespanInDays, paymentFailedGracePeriodInDays int) *SQLStore {
	normalized := strings.ToLower(strings.TrimSpace(dialect))
	return &SQLStore{
		db:                             db,
		postgresQL:                     normalized == "postgres" || normalized == "postgresql" || normalized == "pgx",
		proTrialTimespan:               time.Duration(proTrialTimespanInDays) * 24 * time.Hour,
		paymentFailedGracePeriodInDays: time.Duration(paymentFailedGracePeriodInDays) * 24 * time.Hour,
	}
}

func (s *SQLStore) CreateSubscription(ctx context.Context, tx any, profileID int) error {
	txx, ownTx, err := s.resolveTx(ctx, tx)
	if err != nil {
		return err
	}
	if ownTx {
		defer txx.Rollback()
	}

	now := time.Now().UTC()
	if _, err := txx.ExecContext(ctx, s.bind(`
		INSERT INTO monthly_subscriptions
			(created_at, updated_at, product, is_active, paid, is_trial, started_at, expired_on, cancelled_at, paying_profile_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`), now, now, ProductTypePro.Value, true, false, true, now, now.Add(s.proTrialTimespan), nil, profileID); err != nil {
		return err
	}

	if _, err := txx.ExecContext(ctx, s.bind(`
		INSERT INTO monthly_subscription_benefactors (monthly_subscription_id, profile_id)
		SELECT id, ?
		FROM monthly_subscriptions
		WHERE paying_profile_id = ? AND is_active = ?
		ORDER BY id DESC
		LIMIT 1
	`), profileID, profileID, true); err != nil {
		return err
	}

	if ownTx {
		return txx.Commit()
	}
	return nil
}

func (s *SQLStore) DeactivateExpiredSubscriptions(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, s.bind(`
		UPDATE monthly_subscriptions
		SET is_active = ?, expired_on = ?
		WHERE is_active = ? AND expired_on IS NOT NULL AND expired_on <= ?
	`), false, time.Now().UTC(), true, time.Now().UTC())
	return err
}

func (s *SQLStore) UpdateToPaidPro(ctx context.Context, profileID int) error {
	count, err := s.countActiveSubscriptions(ctx, profileID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()

	switch {
	case count > 1:
		return errors.New("there should only ever be 1 active subscription for a profile")
	case count == 1:
		_, err = s.db.ExecContext(ctx, s.bind(`
			UPDATE monthly_subscriptions
			SET
				updated_at = ?,
				product = ?,
				is_trial = ?,
				is_active = ?,
				started_at = ?,
				expired_on = NULL,
				cancelled_at = NULL
			WHERE paying_profile_id = ? AND is_active = ?
		`), now, ProductTypePro.Value, false, true, now, profileID, true)
		return err
	default:
		if _, err := s.db.ExecContext(ctx, s.bind(`
			INSERT INTO monthly_subscriptions
				(created_at, updated_at, product, is_active, paid, is_trial, started_at, expired_on, cancelled_at, paying_profile_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`), now, now, ProductTypePro.Value, true, false, false, now, nil, nil, profileID); err != nil {
			return err
		}
		_, err := s.db.ExecContext(ctx, s.bind(`
			INSERT INTO monthly_subscription_benefactors (monthly_subscription_id, profile_id)
			SELECT id, ?
			FROM monthly_subscriptions
			WHERE paying_profile_id = ? AND is_active = ?
			ORDER BY id DESC
			LIMIT 1
		`), profileID, profileID, true)
		return err
	}
}

func (s *SQLStore) GetCurrentlyActiveProduct(
	ctx context.Context, profileID int,
) (*ProductType, *time.Time, bool, error) {
	var productRaw string
	var expiredOn sql.NullTime
	var isTrial bool
	err := s.db.QueryRowContext(ctx, s.bind(`
		SELECT product, expired_on, is_trial
		FROM monthly_subscriptions
		WHERE is_active = ? AND paying_profile_id = ?
		ORDER BY id DESC
		LIMIT 1
	`), true, profileID).Scan(&productRaw, &expiredOn, &isTrial)
	if errors.Is(err, sql.ErrNoRows) {
		return &ProductTypeFree, nil, false, nil
	}
	if err != nil {
		return nil, nil, false, err
	}
	product := ParseProductType(productRaw)
	if product == nil {
		return nil, nil, false, nil
	}
	if expiredOn.Valid {
		t := expiredOn.Time
		return product, &t, isTrial, nil
	}
	return product, nil, isTrial, nil
}

func (s *SQLStore) StoreStripeCustomerID(ctx context.Context, profileID int, stripeCustomerID string) error {
	var existing string
	err := s.db.QueryRowContext(ctx, s.bind(`
		SELECT stripe_customer_id
		FROM subscription_customers
		WHERE profile_id = ?
	`), profileID).Scan(&existing)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		_, err = s.db.ExecContext(ctx, s.bind(`
			INSERT INTO subscription_customers (profile_id, stripe_customer_id)
			VALUES (?, ?)
		`), profileID, stripeCustomerID)
		return err
	case err != nil:
		return err
	default:
		if existing == stripeCustomerID {
			return nil
		}
		_, err = s.db.ExecContext(ctx, s.bind(`
			UPDATE subscription_customers
			SET stripe_customer_id = ?
			WHERE profile_id = ?
		`), stripeCustomerID, profileID)
		return err
	}
}

func (s *SQLStore) GetProfileIDFromStripeCustomerID(ctx context.Context, stripeCustomerID string) (int, error) {
	var profileID int
	err := s.db.QueryRowContext(ctx, s.bind(`SELECT profile_id FROM subscription_customers WHERE stripe_customer_id = ?`), stripeCustomerID).Scan(&profileID)
	return profileID, err
}

func (s *SQLStore) CancelWithGracePeriod(ctx context.Context, profileID int) error {
	count, err := s.countActiveSubscriptions(ctx, profileID)
	if err != nil {
		return err
	}
	switch {
	case count == 0:
		return nil
	case count > 1:
		return errors.New("there should only ever be 1 active subscription for a profile")
	}

	var expiredOn sql.NullTime
	err = s.db.QueryRowContext(ctx, s.bind(`
		SELECT expired_on
		FROM monthly_subscriptions
		WHERE paying_profile_id = ? AND is_active = ?
		ORDER BY id DESC
		LIMIT 1
	`), profileID, true).Scan(&expiredOn)
	if err != nil {
		return err
	}

	limit := time.Now().UTC().Add(s.paymentFailedGracePeriodInDays)
	if !expiredOn.Valid || expiredOn.Time.After(limit) {
		_, err = s.db.ExecContext(ctx, s.bind(`
			UPDATE monthly_subscriptions
			SET expired_on = ?
			WHERE paying_profile_id = ? AND is_active = ?
		`), limit, profileID, true)
		return err
	}
	return nil
}

func (s *SQLStore) CancelOrRenew(ctx context.Context, profileID int, cancelDate *time.Time) error {
	if cancelDate == nil {
		_, err := s.db.ExecContext(ctx, s.bind(`
			UPDATE monthly_subscriptions
			SET cancelled_at = NULL, expired_on = NULL
			WHERE paying_profile_id = ? AND is_active = ?
		`), profileID, true)
		return err
	}

	count, err := s.countActiveSubscriptions(ctx, profileID)
	if err != nil {
		return err
	}
	switch {
	case count == 0:
		return nil
	case count > 1:
		return errors.New("there should only ever be 1 active subscription for a profile")
	}

	_, err = s.db.ExecContext(ctx, s.bind(`
		UPDATE monthly_subscriptions
		SET expired_on = ?, cancelled_at = ?
		WHERE paying_profile_id = ? AND is_active = ?
	`), cancelDate.UTC(), time.Now().UTC(), profileID, true)
	return err
}

func (s *SQLStore) UpdateToFree(ctx context.Context, profileID int) error {
	count, err := s.countActiveSubscriptions(ctx, profileID)
	if err != nil {
		return err
	}
	switch {
	case count == 0:
		return nil
	case count > 1:
		return errors.New("there should only ever be 1 active subscription for a profile")
	}
	_, err = s.db.ExecContext(ctx, s.bind(`
		UPDATE monthly_subscriptions
		SET expired_on = ?, is_active = ?
		WHERE paying_profile_id = ? AND is_active = ?
	`), time.Now().UTC(), false, profileID, true)
	return err
}

func (s *SQLStore) countActiveSubscriptions(ctx context.Context, profileID int) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, s.bind(`
		SELECT COUNT(*)
		FROM monthly_subscriptions
		WHERE paying_profile_id = ? AND is_active = ?
	`), profileID, true).Scan(&count)
	return count, err
}

func (s *SQLStore) resolveTx(ctx context.Context, tx any) (*sql.Tx, bool, error) {
	if tx == nil {
		txx, err := s.db.BeginTx(ctx, nil)
		return txx, true, err
	}
	txx, ok := tx.(*sql.Tx)
	if !ok {
		return nil, false, fmt.Errorf("unsupported transaction type %T; expected *sql.Tx", tx)
	}
	return txx, false, nil
}

func (s *SQLStore) bind(query string) string {
	if !s.postgresQL {
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
