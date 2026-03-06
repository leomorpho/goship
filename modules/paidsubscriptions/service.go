package paidsubscriptions

import (
	"context"
	"time"
)

// Service is the public API for the paid subscriptions module.
type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) CreateSubscription(ctx context.Context, tx any, profileID int) error {
	return s.store.CreateSubscription(ctx, tx, profileID)
}

func (s *Service) DeactivateExpiredSubscriptions(ctx context.Context) error {
	return s.store.DeactivateExpiredSubscriptions(ctx)
}

func (s *Service) UpdateToPaidPro(ctx context.Context, profileID int) error {
	return s.store.UpdateToPaidPro(ctx, profileID)
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

func (s *Service) UpdateToFree(ctx context.Context, profileID int) error {
	return s.store.UpdateToFree(ctx, profileID)
}
