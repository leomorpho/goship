package paidsubscriptions

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestSQLStore_SubscriptionLifecycle(t *testing.T) {
	db := openPaidSubsTestDB(t)
	store := NewSQLStore(db, "sqlite3", 15, 3)
	ctx := context.Background()

	require.NoError(t, store.CreateSubscription(ctx, nil, 1, "pro", true, true, nil))
	prod, expiry, isTrial, err := store.GetCurrentlyActiveProduct(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, prod)
	require.Equal(t, "pro", prod.Value)
	require.NotNil(t, expiry)
	require.True(t, isTrial)

	require.NoError(t, store.UpdateToPlan(ctx, 1, "pro", true, false, nil))
	prod, expiry, isTrial, err = store.GetCurrentlyActiveProduct(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, prod)
	require.Equal(t, "pro", prod.Value)
	require.Nil(t, expiry)
	require.False(t, isTrial)

	require.NoError(t, store.UpdateToFree(ctx, 1))
	prod, expiry, isTrial, err = store.GetCurrentlyActiveProduct(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, prod)
	require.Equal(t, "free", prod.Value)
	require.Nil(t, expiry)
	require.False(t, isTrial)
}

func TestSQLStore_StripeIDAndCancellation(t *testing.T) {
	db := openPaidSubsTestDB(t)
	store := NewSQLStore(db, "sqlite3", 15, 3)
	ctx := context.Background()

	require.NoError(t, store.StoreStripeCustomerID(ctx, 1, "cus_123"))
	customerID, err := store.GetStripeCustomerIDByProfileID(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, "cus_123", customerID)

	pid, err := store.GetProfileIDFromStripeCustomerID(ctx, "cus_123")
	require.NoError(t, err)
	require.Equal(t, 1, pid)

	require.NoError(t, store.CreateSubscription(ctx, nil, 1, "pro", true, true, nil))

	require.NoError(t, store.UpdateToPlan(ctx, 1, "team", true, false, nil))
	prod, _, _, err := store.GetCurrentlyActiveProduct(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, prod)
	require.Equal(t, "team", prod.Value)
	require.NoError(t, store.CancelWithGracePeriod(ctx, 1))
	cancelAt := time.Now().UTC().Add(48 * time.Hour)
	require.NoError(t, store.CancelOrRenew(ctx, 1, &cancelAt))
	require.NoError(t, store.CancelOrRenew(ctx, 1, nil))
}

func openPaidSubsTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`
		CREATE TABLE monthly_subscriptions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			product TEXT NOT NULL,
			is_active BOOLEAN NOT NULL,
			paid BOOLEAN NOT NULL,
			is_trial BOOLEAN NOT NULL,
			started_at DATETIME NULL,
			expired_on DATETIME NULL,
			cancelled_at DATETIME NULL,
			paying_profile_id INTEGER NOT NULL
		);
		CREATE UNIQUE INDEX monthlysubscription_paying_profile_id_is_active
			ON monthly_subscriptions (paying_profile_id, is_active);

		CREATE TABLE monthly_subscription_benefactors (
			monthly_subscription_id INTEGER NOT NULL,
			profile_id INTEGER NOT NULL,
			PRIMARY KEY (monthly_subscription_id, profile_id)
		);

		CREATE TABLE subscription_customers (
			profile_id INTEGER PRIMARY KEY,
			stripe_customer_id TEXT NOT NULL UNIQUE
		);
	`)
	require.NoError(t, err)

	return db
}
