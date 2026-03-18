package paidsubscriptions_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/leomorpho/goship-modules/paidsubscriptions"
	"github.com/stretchr/testify/require"
)

type stubStore struct {
	createCalled bool
	updateCalled bool

	createPlan  string
	createPaid  bool
	createTrial bool

	updatePlan  string
	updatePaid  bool
	updateTrial bool

	createErr error
	updateErr error
}

func (s *stubStore) CreateSubscription(_ context.Context, _ any, _ int, planKey string, paid bool, isTrial bool, _ *time.Time) error {
	s.createCalled = true
	s.createPlan = planKey
	s.createPaid = paid
	s.createTrial = isTrial
	return s.createErr
}

func (s *stubStore) DeactivateExpiredSubscriptions(context.Context) error { return nil }

func (s *stubStore) UpdateToPlan(_ context.Context, _ int, planKey string, paid bool, isTrial bool, _ *time.Time) error {
	s.updateCalled = true
	s.updatePlan = planKey
	s.updatePaid = paid
	s.updateTrial = isTrial
	return s.updateErr
}

func (s *stubStore) GetCurrentlyActiveProduct(context.Context, int) (*paidsubscriptions.ProductType, *time.Time, bool, error) {
	return &paidsubscriptions.ProductTypeFree, nil, false, nil
}
func (s *stubStore) StoreStripeCustomerID(context.Context, int, string) error { return nil }
func (s *stubStore) GetStripeCustomerIDByProfileID(context.Context, int) (string, error) {
	return "", nil
}
func (s *stubStore) GetProfileIDFromStripeCustomerID(context.Context, string) (int, error) {
	return 0, nil
}
func (s *stubStore) CancelWithGracePeriod(context.Context, int) error     { return nil }
func (s *stubStore) CancelOrRenew(context.Context, int, *time.Time) error { return nil }
func (s *stubStore) UpdateToFree(context.Context, int) error              { return nil }

func TestService_ActivatePlan_UsesCatalogMetadata(t *testing.T) {
	t.Parallel()

	catalog, err := paidsubscriptions.NewStaticPlanCatalog(
		[]paidsubscriptions.Plan{
			{Key: "free", Paid: false},
			{Key: "starter", Paid: false},
			{Key: "team", Paid: true},
		},
		"free",
		"starter",
	)
	require.NoError(t, err)

	store := &stubStore{}
	svc := paidsubscriptions.NewServiceWithCatalog(store, catalog)

	require.NoError(t, svc.ActivatePlan(context.Background(), 123, "team"))
	require.True(t, store.updateCalled)
	require.Equal(t, "team", store.updatePlan)
	require.True(t, store.updatePaid)
	require.False(t, store.updateTrial)
}

func TestService_CreateSubscription_UsesDefaultTrialPlan(t *testing.T) {
	t.Parallel()

	catalog, err := paidsubscriptions.NewStaticPlanCatalog(
		[]paidsubscriptions.Plan{
			{Key: "free", Paid: false},
			{Key: "starter", Paid: false},
			{Key: "team", Paid: true},
		},
		"free",
		"starter",
	)
	require.NoError(t, err)

	store := &stubStore{}
	svc := paidsubscriptions.NewServiceWithCatalog(store, catalog)
	require.NoError(t, svc.CreateSubscription(context.Background(), nil, 7))

	require.True(t, store.createCalled)
	require.Equal(t, "starter", store.createPlan)
	require.False(t, store.createPaid)
	require.True(t, store.createTrial)
}

func TestService_StartTrial_UnknownPlan(t *testing.T) {
	t.Parallel()
	store := &stubStore{}
	svc := paidsubscriptions.NewService(store)
	err := svc.StartTrial(context.Background(), nil, 9, "missing")
	require.EqualError(t, err, `subscription plan "missing" not found in catalog`)
}

func TestService_ForwardStoreErrors(t *testing.T) {
	t.Parallel()

	store := &stubStore{updateErr: errors.New("boom")}
	svc := paidsubscriptions.NewService(store)
	err := svc.ActivatePlan(context.Background(), 1, "pro")
	require.EqualError(t, err, "boom")
}

func TestService_DefaultPaidPlanKey_UsesCatalogKey(t *testing.T) {
	t.Parallel()

	catalog, err := paidsubscriptions.NewStaticPlanCatalog(
		[]paidsubscriptions.Plan{
			{Key: "free", Paid: false},
			{Key: "business", Paid: true},
		},
		"free",
		"business",
	)
	require.NoError(t, err)

	svc := paidsubscriptions.NewServiceWithCatalog(&stubStore{}, catalog)
	key, err := svc.DefaultPaidPlanKey()
	require.NoError(t, err)
	require.Equal(t, "business", key)
}

func TestService_IsPaidPlanKey_UsesCatalogSemantics(t *testing.T) {
	t.Parallel()

	catalog, err := paidsubscriptions.NewStaticPlanCatalog(
		[]paidsubscriptions.Plan{
			{Key: "starter", Paid: false},
			{Key: "enterprise", Paid: true},
		},
		"starter",
		"enterprise",
	)
	require.NoError(t, err)

	svc := paidsubscriptions.NewServiceWithCatalog(&stubStore{}, catalog)
	require.Equal(t, "starter", svc.FreePlanKey())
	require.True(t, svc.IsPaidPlanKey("enterprise"))
	require.False(t, svc.IsPaidPlanKey("starter"))
}
