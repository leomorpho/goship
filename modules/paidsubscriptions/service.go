package paidsubscriptions

import (
	"context"
	"time"
)

// Service is the public API for the paid subscriptions module.
type Service struct {
	store   Store
	catalog PlanCatalog
}

func NewService(store Store) *Service {
	return &Service{
		store:   store,
		catalog: mustDefaultPlanCatalog(),
	}
}

func NewServiceWithCatalog(store Store, catalog PlanCatalog) *Service {
	if catalog == nil {
		catalog = mustDefaultPlanCatalog()
	}
	return &Service{
		store:   store,
		catalog: catalog,
	}
}

// CreateSubscription creates a new trial subscription using the catalog default trial plan.
func (s *Service) CreateSubscription(ctx context.Context, tx any, profileID int) error {
	planKey := s.catalog.DefaultTrialPlanKey()
	plan, ok := s.catalog.PlanByKey(planKey)
	if !ok {
		return ErrPlanNotFound(planKey)
	}
	return s.store.CreateSubscription(ctx, tx, profileID, plan.Key, plan.Paid, true, nil)
}

func (s *Service) DeactivateExpiredSubscriptions(ctx context.Context) error {
	return s.store.DeactivateExpiredSubscriptions(ctx)
}

// ActivatePlan moves a profile to an active, non-trial plan from the catalog.
func (s *Service) ActivatePlan(ctx context.Context, profileID int, planKey string) error {
	plan, ok := s.catalog.PlanByKey(planKey)
	if !ok {
		return ErrPlanNotFound(planKey)
	}
	return s.store.UpdateToPlan(ctx, profileID, plan.Key, plan.Paid, false, nil)
}

// StartTrial creates or updates to a trial plan from the catalog.
func (s *Service) StartTrial(ctx context.Context, tx any, profileID int, planKey string) error {
	plan, ok := s.catalog.PlanByKey(planKey)
	if !ok {
		return ErrPlanNotFound(planKey)
	}
	return s.store.CreateSubscription(ctx, tx, profileID, plan.Key, plan.Paid, true, nil)
}

// UpdateToPaidPro is kept as a convenience wrapper for the default catalog.
func (s *Service) UpdateToPaidPro(ctx context.Context, profileID int) error {
	return s.ActivatePlan(ctx, profileID, ProductTypePro.Value)
}

// FreePlanKey returns the catalog-defined free plan key.
func (s *Service) FreePlanKey() string {
	return s.catalog.FreePlanKey()
}

// IsPaidPlanKey reports whether a plan key resolves to a paid plan in the catalog.
func (s *Service) IsPaidPlanKey(planKey string) bool {
	plan, ok := s.catalog.PlanByKey(planKey)
	if !ok {
		return false
	}
	return plan.Paid
}

// DefaultPaidPlanKey returns the catalog default trial plan when it is paid.
func (s *Service) DefaultPaidPlanKey() (string, error) {
	key := s.catalog.DefaultTrialPlanKey()
	plan, ok := s.catalog.PlanByKey(key)
	if !ok {
		return "", ErrPlanNotFound(key)
	}
	if !plan.Paid {
		return "", ErrNoDefaultPaidPlan(key)
	}
	return plan.Key, nil
}

func (s *Service) GetCurrentlyActiveProduct(ctx context.Context, profileID int) (*ProductType, *time.Time, bool, error) {
	return s.store.GetCurrentlyActiveProduct(ctx, profileID)
}

func (s *Service) StoreStripeCustomerID(ctx context.Context, profileID int, stripeCustomerID string) error {
	return s.store.StoreStripeCustomerID(ctx, profileID, stripeCustomerID)
}

func (s *Service) GetStripeCustomerIDByProfileID(ctx context.Context, profileID int) (string, error) {
	return s.store.GetStripeCustomerIDByProfileID(ctx, profileID)
}

func (s *Service) GetProfileIDFromStripeCustomerID(ctx context.Context, stripeCustomerID string) (int, error) {
	return s.store.GetProfileIDFromStripeCustomerID(ctx, stripeCustomerID)
}

func (s *Service) CancelWithGracePeriod(ctx context.Context, profileID int) error {
	return s.store.CancelWithGracePeriod(ctx, profileID)
}

func (s *Service) CancelOrRenew(ctx context.Context, profileID int, cancelDate *time.Time) error {
	return s.store.CancelOrRenew(ctx, profileID, cancelDate)
}

// UpdateToFree deactivates the active subscription and falls back to free-tier behavior.
func (s *Service) UpdateToFree(ctx context.Context, profileID int) error {
	return s.store.UpdateToFree(ctx, profileID)
}
