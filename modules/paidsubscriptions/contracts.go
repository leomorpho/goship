package paidsubscriptions

import (
	"context"
	"time"
)

// Store defines the DB boundary for the paid subscriptions module.
type Store interface {
	CreateSubscription(ctx context.Context, tx any, profileID int) error
	DeactivateExpiredSubscriptions(ctx context.Context) error
	UpdateToPaidPro(ctx context.Context, profileID int) error
	GetCurrentlyActiveProduct(ctx context.Context, profileID int) (*ProductType, *time.Time, bool, error)
	StoreStripeCustomerID(ctx context.Context, profileID int, stripeCustomerID string) error
	GetProfileIDFromStripeCustomerID(ctx context.Context, stripeCustomerID string) (int, error)
	CancelWithGracePeriod(ctx context.Context, profileID int) error
	CancelOrRenew(ctx context.Context, profileID int, cancelDate *time.Time) error
	UpdateToFree(ctx context.Context, profileID int) error
}
