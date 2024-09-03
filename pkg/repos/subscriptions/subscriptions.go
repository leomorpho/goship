package subscriptions

import (
	"context"
	"errors"
	"time"

	"github.com/mikestefanello/pagoda/ent"
	"github.com/mikestefanello/pagoda/ent/monthlysubscription"
	"github.com/mikestefanello/pagoda/ent/profile"
	"github.com/mikestefanello/pagoda/pkg/domain"
	"github.com/rs/zerolog/log"
)

const HOURS_IN_DAY = 24 * time.Hour

type SubscriptionsRepo struct {
	orm                            *ent.Client
	proTrialTimespanInDays         time.Duration
	paymentFailedGracePeriodInDays time.Duration
}

func NewSubscriptionsRepo(orm *ent.Client, proTrialTimespanInDays, paymentFailedGracePeriodInDays int) *SubscriptionsRepo {
	return &SubscriptionsRepo{
		orm:                            orm,
		proTrialTimespanInDays:         time.Duration(proTrialTimespanInDays) * 24 * time.Hour,
		paymentFailedGracePeriodInDays: time.Duration(paymentFailedGracePeriodInDays) * 24 * time.Hour,
	}
}

// CreateSubscription creates a subscription when a user first onboards. It automatically gives a free trial.
func (s *SubscriptionsRepo) CreateSubscription(
	ctx context.Context, tx *ent.Tx, profileID int,
) (err error) {
	commit := false
	if tx == nil {
		commit = true
		tx, err = s.orm.Tx(ctx)
		if err != nil {
			return err
		}
	}
	err = tx.MonthlySubscription.
		Create().
		SetProduct(monthlysubscription.Product(domain.ProductTypePro.Value)).
		SetPayerID(profileID).
		AddBenefactorIDs(profileID).
		SetIsTrial(true).
		SetIsActive(true).
		SetStartedAt(time.Now()).
		SetExpiredOn(time.Now().Add(s.proTrialTimespanInDays)).
		Exec(ctx)
	if commit {
		if err != nil {
			tx.Rollback()
		}
		return tx.Commit()
	}
	return err
}

// DeactivateExpiredSubscriptions deactivates all subscriptions that have come to terms.
func (s *SubscriptionsRepo) DeactivateExpiredSubscriptions(ctx context.Context) error {
	return s.orm.MonthlySubscription.
		Update().
		Where(
			monthlysubscription.IsActive(true),
			monthlysubscription.ExpiredOnLTE(time.Now()),
		).
		SetIsActive(false).
		SetExpiredOn(time.Now()).
		Exec(ctx)
}

// UpdateToPaidPro idempotently updates a subscription to pro plan.
func (s *SubscriptionsRepo) UpdateToPaidPro(
	ctx context.Context, profileID int,
) error {
	count, err := s.orm.MonthlySubscription.
		Query().
		Where(
			monthlysubscription.HasPayerWith(profile.IDEQ(profileID)),
			monthlysubscription.IsActive(true),
		).Count(ctx)
	if err != nil {
		return err
	}
	if count == 1 {
		return s.orm.MonthlySubscription.
			Update().
			Where(
				monthlysubscription.HasPayerWith(profile.IDEQ(profileID)),
				monthlysubscription.IsActive(true),
			).
			SetProduct(monthlysubscription.Product(domain.ProductTypePro.Value)).
			SetPayerID(profileID).
			AddBenefactorIDs(profileID).
			SetIsTrial(false).
			SetIsActive(true).
			SetStartedAt(time.Now()).
			ClearExpiredOn().
			ClearCancelledAt().
			Exec(ctx)
	} else if count > 1 {
		return errors.New("there should only ever be 1 active subscription for a profile")
	}

	return s.orm.MonthlySubscription.
		Create().
		SetProduct(monthlysubscription.Product(domain.ProductTypePro.Value)).
		SetPayerID(profileID).
		AddBenefactorIDs(profileID).
		SetIsTrial(false).
		SetIsActive(true).
		SetStartedAt(time.Now()).
		Exec(ctx)
}

// TODO: refactor to return domain.ProductType and not *domain.ProductType
func (s *SubscriptionsRepo) GetCurrentlyActiveProduct(
	ctx context.Context, profileID int,
) (*domain.ProductType, *time.Time, bool, error) {
	sub, err := s.orm.MonthlySubscription.
		Query().
		Where(
			monthlysubscription.IsActive(true),
			monthlysubscription.HasPayerWith(
				profile.IDEQ(profileID),
			),
		).
		Only(ctx)

	if ent.IsNotFound(err) {
		return &domain.ProductTypeFree, nil, false, nil
	}
	if ent.IsNotSingular(err) {
		log.Error().Err(err)
		return nil, nil, false, nil
	}
	return domain.ProductTypes.Parse(string(sub.Product)), sub.ExpiredOn, sub.IsTrial, nil
}

// Helper function to store the Stripe customer ID in the database
func (s *SubscriptionsRepo) StoreStripeCustomerID(ctx context.Context, profileID int, stripeCustomerID string) error {
	// Retrieve the current profile to check the existing Stripe customer ID
	profile, err := s.orm.Profile.Get(ctx, profileID)
	if err != nil {
		return err
	}

	// Check if the Stripe customer ID is already set to the given value
	if profile.StripeID == stripeCustomerID {
		// No update necessary if the Stripe customer ID is already set to the given value
		return nil
	}

	// Update the profile record with the new Stripe customer ID
	_, err = s.orm.Profile.UpdateOneID(profileID).SetStripeID(stripeCustomerID).Save(ctx)
	return err
}

// Helper function to store the Stripe customer ID in the database
func (s *SubscriptionsRepo) GetProfileIDFromStripeCustomerID(ctx context.Context, stripeCustomerID string) (int, error) {
	profile, err := s.orm.Profile.Query().
		Where(
			profile.StripeIDEQ(stripeCustomerID),
		).
		Select(profile.FieldID).
		Only(ctx)
	return profile.ID, err
}

// CancelWithGracePeriod idempotently manages a subscription when a payment fails.
func (s *SubscriptionsRepo) CancelWithGracePeriod(ctx context.Context, profileID int) error {
	count, err := s.orm.MonthlySubscription.
		Query().
		Where(
			monthlysubscription.HasPayerWith(profile.IDEQ(profileID)),
			monthlysubscription.IsActive(true),
		).
		Count(ctx)
	if err != nil {
		return err
	}
	if count == 0 {
		// on free plan
		return nil
	} else if count > 1 {
		return errors.New("there should only ever be 1 active subscription for a profile")
	}

	m, err := s.orm.MonthlySubscription.
		Query().
		Where(
			monthlysubscription.HasPayerWith(profile.IDEQ(profileID)),
			monthlysubscription.IsActive(true),
		).
		Select(monthlysubscription.FieldExpiredOn).
		Only(ctx)
	if err != nil {
		return err
	}

	if m.ExpiredOn == nil || m.ExpiredOn.After(time.Now().Add(s.paymentFailedGracePeriodInDays)) {
		_, err := s.orm.MonthlySubscription.Update().
			Where(
				monthlysubscription.HasPayerWith(profile.IDEQ(profileID)),
				monthlysubscription.IsActive(true),
			).
			SetExpiredOn(time.Now().Add(s.paymentFailedGracePeriodInDays)).
			Save(ctx)
		return err
	}
	return nil
}

// CancelOrRenew idempotently updates the subscription status to cancelled and sets the expiry date.
// If the expiry date is nil, the subscription is renewed.
func (s *SubscriptionsRepo) CancelOrRenew(
	ctx context.Context, profileID int, cancelDate *time.Time,
) error {
	if cancelDate == nil {
		return s.orm.MonthlySubscription.Update().
			Where(
				monthlysubscription.HasPayerWith(profile.IDEQ(profileID)),
				monthlysubscription.IsActive(true),
			).
			ClearCancelledAt().
			ClearExpiredOn().
			Exec(ctx)

	}
	count, err := s.orm.MonthlySubscription.
		Query().
		Where(
			monthlysubscription.HasPayerWith(profile.IDEQ(profileID)),
			monthlysubscription.IsActive(true),
		).
		Count(ctx)
	if err != nil {
		return err
	}
	if count == 0 {
		// on free plan
		return nil
	} else if count > 1 {
		return errors.New("there should only ever be 1 active subscription for a profile")
	}

	_, err = s.orm.MonthlySubscription.Update().
		Where(
			monthlysubscription.HasPayerWith(profile.IDEQ(profileID)),
			monthlysubscription.IsActive(true),
		).
		SetNillableExpiredOn(cancelDate).
		SetCancelledAt(time.Now()).
		Save(ctx)
	return err
}

// UpdateToFree idempotently and IMMEDIATELY expires a pro membership. For now, a free subscription is represented by no active pro subscription.
// Note that this should only be used in controlled areas and CancelWithGracePeriod should be preferred over it for production flows.
func (s *SubscriptionsRepo) UpdateToFree(
	ctx context.Context, profileID int,
) error {
	count, err := s.orm.MonthlySubscription.
		Query().
		Where(
			monthlysubscription.HasPayerWith(profile.IDEQ(profileID)),
			monthlysubscription.IsActive(true),
		).Count(ctx)
	if err != nil {
		return err
	}
	if count == 0 {
		// already on free plan
		return nil
	} else if count > 1 {
		return errors.New("there should only ever be 1 active subscription for a profile")
	}
	return s.orm.MonthlySubscription.Update().
		Where(
			monthlysubscription.HasPayerWith(profile.IDEQ(profileID)),
			monthlysubscription.IsActive(true),
		).
		SetExpiredOn(time.Now()).
		SetIsActive(false).
		Exec(ctx)
}
