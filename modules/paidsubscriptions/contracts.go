package paidsubscriptions

import (
	"context"
	"time"

	"github.com/leomorpho/goship/db/ent"
	"github.com/leomorpho/goship/framework/domain"
)

// Store defines the DB boundary for the paid subscriptions module.
type Store interface {
	CreateSubscription(ctx context.Context, tx *ent.Tx, profileID int) error
	DeactivateExpiredSubscriptions(ctx context.Context) error
	UpdateToPaidPro(ctx context.Context, profileID int) error
	GetCurrentlyActiveProduct(ctx context.Context, profileID int) (*domain.ProductType, *time.Time, bool, error)
	StoreStripeCustomerID(ctx context.Context, profileID int, stripeCustomerID string) error
	GetProfileIDFromStripeCustomerID(ctx context.Context, stripeCustomerID string) (int, error)
	CancelWithGracePeriod(ctx context.Context, profileID int) error
	CancelOrRenew(ctx context.Context, profileID int, cancelDate *time.Time) error
	UpdateToFree(ctx context.Context, profileID int) error
}
