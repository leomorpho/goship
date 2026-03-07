package paidsubscriptions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	dbqueries "github.com/leomorpho/goship-modules/paidsubscriptions/db/queries"
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
	createSubscriptionQuery, err := dbqueries.Get("create_subscription")
	if err != nil {
		return err
	}
	if _, err := txx.ExecContext(ctx, s.bind(createSubscriptionQuery), now, now, ProductTypePro.Value, true, false, true, now, now.Add(s.proTrialTimespan), nil, profileID); err != nil {
		return err
	}

	createLinkQuery, err := dbqueries.Get("create_subscription_benefactor_link")
	if err != nil {
		return err
	}
	if _, err := txx.ExecContext(ctx, s.bind(createLinkQuery), profileID, profileID, true); err != nil {
		return err
	}

	if ownTx {
		return txx.Commit()
	}
	return nil
}

func (s *SQLStore) DeactivateExpiredSubscriptions(ctx context.Context) error {
	query, err := dbqueries.Get("deactivate_expired_subscriptions")
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, s.bind(query), false, time.Now().UTC(), true, time.Now().UTC())
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
		query, lookupErr := dbqueries.Get("update_to_paid_pro_existing")
		if lookupErr != nil {
			return lookupErr
		}
		_, err = s.db.ExecContext(ctx, s.bind(query), now, ProductTypePro.Value, false, true, now, profileID, true)
		return err
	default:
		createSubscriptionQuery, lookupErr := dbqueries.Get("create_subscription")
		if lookupErr != nil {
			return lookupErr
		}
		if _, err := s.db.ExecContext(ctx, s.bind(createSubscriptionQuery), now, now, ProductTypePro.Value, true, false, false, now, nil, nil, profileID); err != nil {
			return err
		}
		createLinkQuery, lookupErr := dbqueries.Get("create_subscription_benefactor_link")
		if lookupErr != nil {
			return lookupErr
		}
		_, err := s.db.ExecContext(ctx, s.bind(createLinkQuery), profileID, profileID, true)
		return err
	}
}

func (s *SQLStore) GetCurrentlyActiveProduct(
	ctx context.Context, profileID int,
) (*ProductType, *time.Time, bool, error) {
	var productRaw string
	var expiredOn sql.NullTime
	var isTrial bool
	query, err := dbqueries.Get("get_currently_active_product")
	if err != nil {
		return nil, nil, false, err
	}
	err = s.db.QueryRowContext(ctx, s.bind(query), true, profileID).Scan(&productRaw, &expiredOn, &isTrial)
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
	getQuery, err := dbqueries.Get("get_stripe_customer_id_by_profile")
	if err != nil {
		return err
	}
	err = s.db.QueryRowContext(ctx, s.bind(getQuery), profileID).Scan(&existing)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		insertQuery, lookupErr := dbqueries.Get("insert_stripe_customer_id")
		if lookupErr != nil {
			return lookupErr
		}
		_, err = s.db.ExecContext(ctx, s.bind(insertQuery), profileID, stripeCustomerID)
		return err
	case err != nil:
		return err
	default:
		if existing == stripeCustomerID {
			return nil
		}
		updateQuery, lookupErr := dbqueries.Get("update_stripe_customer_id")
		if lookupErr != nil {
			return lookupErr
		}
		_, err = s.db.ExecContext(ctx, s.bind(updateQuery), stripeCustomerID, profileID)
		return err
	}
}

func (s *SQLStore) GetProfileIDFromStripeCustomerID(ctx context.Context, stripeCustomerID string) (int, error) {
	query, err := dbqueries.Get("get_profile_id_by_stripe_customer")
	if err != nil {
		return 0, err
	}
	var profileID int
	err = s.db.QueryRowContext(ctx, s.bind(query), stripeCustomerID).Scan(&profileID)
	return profileID, err
}

func (s *SQLStore) GetStripeCustomerIDByProfileID(ctx context.Context, profileID int) (string, error) {
	query, err := dbqueries.Get("get_stripe_customer_id_by_profile")
	if err != nil {
		return "", err
	}
	var stripeCustomerID string
	err = s.db.QueryRowContext(ctx, s.bind(query), profileID).Scan(&stripeCustomerID)
	return stripeCustomerID, err
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
	getLatestExpiryQuery, err := dbqueries.Get("get_latest_expired_on")
	if err != nil {
		return err
	}
	err = s.db.QueryRowContext(ctx, s.bind(getLatestExpiryQuery), profileID, true).Scan(&expiredOn)
	if err != nil {
		return err
	}

	limit := time.Now().UTC().Add(s.paymentFailedGracePeriodInDays)
	if !expiredOn.Valid || expiredOn.Time.After(limit) {
		updateExpiryQuery, lookupErr := dbqueries.Get("update_expired_on_for_active")
		if lookupErr != nil {
			return lookupErr
		}
		_, err = s.db.ExecContext(ctx, s.bind(updateExpiryQuery), limit, profileID, true)
		return err
	}
	return nil
}

func (s *SQLStore) CancelOrRenew(ctx context.Context, profileID int, cancelDate *time.Time) error {
	if cancelDate == nil {
		query, err := dbqueries.Get("cancel_or_renew_clear")
		if err != nil {
			return err
		}
		_, err = s.db.ExecContext(ctx, s.bind(query), profileID, true)
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

	query, err := dbqueries.Get("cancel_or_renew_set")
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, s.bind(query), cancelDate.UTC(), time.Now().UTC(), profileID, true)
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
	query, err := dbqueries.Get("update_to_free")
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, s.bind(query), time.Now().UTC(), false, profileID, true)
	return err
}

func (s *SQLStore) countActiveSubscriptions(ctx context.Context, profileID int) (int, error) {
	query, err := dbqueries.Get("count_active_subscriptions")
	if err != nil {
		return 0, err
	}
	var count int
	err = s.db.QueryRowContext(ctx, s.bind(query), profileID, true).Scan(&count)
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
