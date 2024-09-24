package subscriptions_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/jackc/pgx/stdlib"
	"github.com/stretchr/testify/assert"

	"github.com/mikestefanello/pagoda/ent/monthlysubscription"
	"github.com/mikestefanello/pagoda/ent/profile"
	"github.com/mikestefanello/pagoda/pkg/domain"
	"github.com/mikestefanello/pagoda/pkg/repos/profilerepo"
	"github.com/mikestefanello/pagoda/pkg/repos/subscriptions"
	"github.com/mikestefanello/pagoda/pkg/tests"
)

func init() {
	// Register "pgx" as "postgres" explicitly for database/sql
	sql.Register("postgres", stdlib.GetDefaultDriver())
}

func TestGetCurrentlyActiveProduct(t *testing.T) {
	client, ctx := tests.CreateTestContainerPostgresEntClient(t)
	defer client.Close()

	// Create users
	user1 := tests.CreateUser(ctx, client, "Jo Bandi", "jo@gmail.com", "password", true)

	// Create pofilesr
	profileRepo := profilerepo.NewProfileRepo(client, nil, nil)
	profile1Obj, err := profileRepo.CreateProfile(
		ctx, user1, "bio", time.Time{}, nil, nil,
	)
	assert.Nil(t, err)

	subscriptionsRepo := subscriptions.NewSubscriptionsRepo(client, 15, 3)

	err = subscriptionsRepo.CreateSubscription(
		ctx, nil, profile1Obj.ID,
	)
	assert.Nil(t, err)

	count, err := client.MonthlySubscription.Query().Count(ctx)
	assert.Nil(t, err)
	assert.Equal(t, 1, count)

	prodName, expiredOn, isTrial, err := subscriptionsRepo.GetCurrentlyActiveProduct(ctx, profile1Obj.ID)
	assert.Nil(t, err)
	assert.True(t, isTrial)
	assert.NotNil(t, prodName)
	assert.NotNil(t, expiredOn)
	assert.Equal(t, domain.ProductTypePro, *prodName)

	for i := 0; i < 5; i++ {
		// For idempotency
		err = subscriptionsRepo.UpdateToFree(ctx, profile1Obj.ID)
		assert.Nil(t, err)
	}

	count, err = client.MonthlySubscription.Query().Count(ctx)
	assert.Nil(t, err)
	assert.Equal(t, 1, count)

	prodName, expiredOn, isTrial, err = subscriptionsRepo.GetCurrentlyActiveProduct(ctx, profile1Obj.ID)
	assert.Nil(t, err)
	assert.False(t, isTrial)
	assert.NotNil(t, prodName)
	assert.Nil(t, expiredOn)
	assert.Equal(t, domain.ProductTypeFree, *prodName)

	for i := 0; i < 5; i++ {
		// For idempotency
		err = subscriptionsRepo.UpdateToPaidPro(ctx, profile1Obj.ID)
		assert.Nil(t, err)
	}

	count, err = client.MonthlySubscription.Query().Count(ctx)
	assert.Nil(t, err)
	assert.Equal(t, 2, count)

	prodName, expiredOn, isTrial, err = subscriptionsRepo.GetCurrentlyActiveProduct(ctx, profile1Obj.ID)
	assert.Nil(t, err)
	assert.NotNil(t, prodName)
	assert.Nil(t, expiredOn) // Not a trial anymore, valid until cancelled
	assert.False(t, isTrial)

	assert.Equal(t, domain.ProductTypePro, *prodName)
}

func TestDeactivateExpiredSubscriptions(t *testing.T) {
	client, ctx := tests.CreateTestContainerPostgresEntClient(t)
	defer client.Close()

	// Create users
	user1 := tests.CreateUser(ctx, client, "Jo Bandi", "jo@gmail.com", "password", true)
	user2 := tests.CreateUser(ctx, client, "Boris Yelstin", "boris@gmail.com", "password", true)

	// Create pofilesr
	profileRepo := profilerepo.NewProfileRepo(client, nil, nil)
	profile1Obj, err := profileRepo.CreateProfile(
		ctx, user1, "bio", time.Time{}, nil, nil,
	)
	assert.Nil(t, err)
	profile2Obj, err := profileRepo.CreateProfile(
		ctx, user2, "bio", time.Time{}, nil, nil,
	)
	assert.Nil(t, err)

	subscriptionsRepo := subscriptions.NewSubscriptionsRepo(client, 15, 3)

	err = subscriptionsRepo.CreateSubscription(
		ctx, nil, profile1Obj.ID,
	)
	assert.Nil(t, err)
	err = subscriptionsRepo.CreateSubscription(
		ctx, nil, profile2Obj.ID,
	)
	assert.Nil(t, err)

	// Set expired timestamp for 1 subscription in past
	err = client.MonthlySubscription.Update().
		Where(
			monthlysubscription.HasPayerWith(profile.IDEQ(profile1Obj.ID)),
			monthlysubscription.IsActiveEQ(true),
		).
		SetExpiredOn(time.Now().Add(time.Second * -1)).
		Exec(ctx)
	assert.Nil(t, err)

	err = subscriptionsRepo.DeactivateExpiredSubscriptions(ctx)
	assert.Nil(t, err)

	prod, expiry, isTrial, err := subscriptionsRepo.GetCurrentlyActiveProduct(ctx, profile1Obj.ID)
	assert.Equal(t, domain.ProductTypeFree, *prod)
	assert.Nil(t, expiry)
	assert.False(t, isTrial)
	assert.Nil(t, err)

	prod, expiry, isTrial, err = subscriptionsRepo.GetCurrentlyActiveProduct(ctx, profile2Obj.ID)
	assert.Equal(t, domain.ProductTypePro, *prod)
	assert.NotNil(t, expiry)
	assert.True(t, isTrial)
	assert.Nil(t, err)
}

func TestStripeCustomerIDs(t *testing.T) {
	client, ctx := tests.CreateTestContainerPostgresEntClient(t)
	defer client.Close()

	// Create users
	user1 := tests.CreateUser(ctx, client, "Jo Bandi", "jo@gmail.com", "password", true)

	// Create pofilesr
	profileRepo := profilerepo.NewProfileRepo(client, nil, nil)
	profile1Obj, err := profileRepo.CreateProfile(
		ctx, user1, "bio", time.Time{}, nil, nil,
	)
	assert.Nil(t, err)

	subscriptionsRepo := subscriptions.NewSubscriptionsRepo(client, 15, 3)
	stripeCustomerID := "cus_1234"

	err = subscriptionsRepo.StoreStripeCustomerID(ctx, profile1Obj.ID, stripeCustomerID)
	assert.Nil(t, err)

	profileID, err := subscriptionsRepo.GetProfileIDFromStripeCustomerID(ctx, stripeCustomerID)
	assert.Nil(t, err)
	assert.Equal(t, profile1Obj.ID, profileID)
}

func TestCancelWithGracePeriod(t *testing.T) {
	client, ctx := tests.CreateTestContainerPostgresEntClient(t)
	defer client.Close()

	// Create users
	user1 := tests.CreateUser(ctx, client, "Jo Bandi", "jo@gmail.com", "password", true)
	user2 := tests.CreateUser(ctx, client, "Boris Yelstin", "boris@gmail.com", "password", true)

	// Create pofilesr
	profileRepo := profilerepo.NewProfileRepo(client, nil, nil)
	profile1Obj, err := profileRepo.CreateProfile(
		ctx, user1, "bio", time.Time{}, nil, nil,
	)
	assert.Nil(t, err)
	profile2Obj, err := profileRepo.CreateProfile(
		ctx, user2, "bio", time.Time{}, nil, nil,
	)
	assert.Nil(t, err)

	subscriptionsRepo := subscriptions.NewSubscriptionsRepo(client, 15, 3)

	// Only have profile1 start with a trial, profile2 will start with a full pro plan
	err = subscriptionsRepo.CreateSubscription(
		ctx, nil, profile1Obj.ID,
	)
	assert.Nil(t, err)

	// Update all to PRO plans (not trials)
	err = subscriptionsRepo.UpdateToPaidPro(ctx, profile1Obj.ID)
	assert.Nil(t, err)
	err = subscriptionsRepo.UpdateToPaidPro(ctx, profile2Obj.ID)
	assert.Nil(t, err)

	subProfile1, err := client.MonthlySubscription.
		Query().
		Where(
			monthlysubscription.HasPayerWith(profile.IDEQ(profile1Obj.ID)),
			monthlysubscription.IsActiveEQ(true),
		).
		Only(ctx)
	assert.Nil(t, err)
	assert.Nil(t, subProfile1.ExpiredOn)
	assert.True(t, subProfile1.IsActive)
	assert.False(t, subProfile1.IsTrial)
	assert.False(t, subProfile1.Paid)
	assert.Equal(t, domain.ProductTypePro, *domain.ProductTypes.Parse(string(subProfile1.Product)))
	assert.NotNil(t, subProfile1.StartedAt)
	assert.Nil(t, subProfile1.CancelledAt)

	for i := 0; i < 5; i++ {
		// For idempotency
		err = subscriptionsRepo.CancelWithGracePeriod(ctx, profile1Obj.ID)
		assert.Nil(t, err)
	}

	subProfile1, err = client.MonthlySubscription.
		Query().
		Where(
			monthlysubscription.HasPayerWith(profile.IDEQ(profile1Obj.ID)),
			monthlysubscription.IsActiveEQ(true),
		).
		Only(ctx)
	assert.Nil(t, err)
	assert.NotNil(t, subProfile1.ExpiredOn)
	assert.True(t, subProfile1.IsActive)
	assert.False(t, subProfile1.IsTrial)
	assert.False(t, subProfile1.Paid)
	assert.Equal(t, domain.ProductTypePro, *domain.ProductTypes.Parse(string(subProfile1.Product)))
	assert.NotNil(t, subProfile1.StartedAt)
	assert.Nil(t, subProfile1.CancelledAt)

	subProfile2, err := client.MonthlySubscription.
		Query().
		Where(
			monthlysubscription.HasPayerWith(profile.IDEQ(profile2Obj.ID)),
			monthlysubscription.IsActiveEQ(true),
		).
		Only(ctx)
	assert.Nil(t, err)
	assert.Nil(t, subProfile2.ExpiredOn)
	assert.True(t, subProfile2.IsActive)
	assert.False(t, subProfile1.IsTrial)
	assert.False(t, subProfile1.Paid)
	assert.Equal(t, domain.ProductTypePro, *domain.ProductTypes.Parse(string(subProfile1.Product)))
	assert.NotNil(t, subProfile1.StartedAt)
	assert.Nil(t, subProfile1.CancelledAt)
}

func TestCancelOrRenew(t *testing.T) {
	client, ctx := tests.CreateTestContainerPostgresEntClient(t)
	defer client.Close()

	// Create users
	user1 := tests.CreateUser(ctx, client, "Jo Bandi", "jo@gmail.com", "password", true)
	user2 := tests.CreateUser(ctx, client, "Boris Yelstin", "boris@gmail.com", "password", true)

	// Create pofilesr
	profileRepo := profilerepo.NewProfileRepo(client, nil, nil)
	profile1Obj, err := profileRepo.CreateProfile(
		ctx, user1, "bio", time.Time{}, nil, nil,
	)
	assert.Nil(t, err)
	profile2Obj, err := profileRepo.CreateProfile(
		ctx, user2, "bio", time.Time{}, nil, nil,
	)
	assert.Nil(t, err)

	subscriptionsRepo := subscriptions.NewSubscriptionsRepo(client, 15, 3)
	// Update all to PRO plans (not trials)
	err = subscriptionsRepo.UpdateToPaidPro(ctx, profile1Obj.ID)
	assert.Nil(t, err)
	err = subscriptionsRepo.UpdateToPaidPro(ctx, profile2Obj.ID)
	assert.Nil(t, err)

	subProfile1, err := client.MonthlySubscription.
		Query().
		Where(
			monthlysubscription.HasPayerWith(profile.IDEQ(profile1Obj.ID)),
			monthlysubscription.IsActiveEQ(true),
		).
		Only(ctx)
	assert.Nil(t, err)
	assert.Nil(t, subProfile1.ExpiredOn)
	assert.Nil(t, subProfile1.CancelledAt)

	err = subscriptionsRepo.CancelOrRenew(ctx, profile1Obj.ID, nil)
	assert.Nil(t, err)
	subProfile1, err = client.MonthlySubscription.
		Query().
		Where(
			monthlysubscription.HasPayerWith(profile.IDEQ(profile1Obj.ID)),
			monthlysubscription.IsActiveEQ(true),
		).
		Only(ctx)
	assert.Nil(t, err)
	assert.Nil(t, subProfile1.ExpiredOn)
	assert.Nil(t, subProfile1.CancelledAt)

	timestamp := time.Now().Add(time.Hour * 12)

	// Cancel
	for i := 0; i < 5; i++ {
		// Test for idempotency
		err = subscriptionsRepo.CancelOrRenew(ctx, profile1Obj.ID, &timestamp)
		assert.Nil(t, err)

	}
	subProfile1, err = client.MonthlySubscription.
		Query().
		Where(
			monthlysubscription.HasPayerWith(profile.IDEQ(profile1Obj.ID)),
			monthlysubscription.IsActiveEQ(true),
		).
		Only(ctx)
	assert.Nil(t, err)
	assert.Equal(t, timestamp.Truncate(time.Second).UTC(), subProfile1.ExpiredOn.Truncate(time.Second).UTC())
	assert.NotNil(t, subProfile1.CancelledAt)

	// Renew
	for i := 0; i < 5; i++ {
		// Test for idempotency
		err = subscriptionsRepo.CancelOrRenew(ctx, profile1Obj.ID, nil)
		assert.Nil(t, err)

	}
	subProfile1, err = client.MonthlySubscription.
		Query().
		Where(
			monthlysubscription.HasPayerWith(profile.IDEQ(profile1Obj.ID)),
			monthlysubscription.IsActiveEQ(true),
		).
		Only(ctx)
	assert.Nil(t, err)
	assert.Nil(t, subProfile1.ExpiredOn)
	assert.Nil(t, subProfile1.CancelledAt)

	// Make sure the other profile is still as expected
	subProfile2, err := client.MonthlySubscription.
		Query().
		Where(
			monthlysubscription.HasPayerWith(profile.IDEQ(profile2Obj.ID)),
			monthlysubscription.IsActiveEQ(true),
		).
		Only(ctx)
	assert.Nil(t, err)
	assert.Nil(t, subProfile2.ExpiredOn)
	assert.Nil(t, subProfile2.CancelledAt)
}
