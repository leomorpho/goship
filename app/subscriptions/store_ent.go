package subscriptions

import (
	"context"
	"errors"
	"fmt"
	"time"

	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	"github.com/leomorpho/goship/db/ent"
	"github.com/leomorpho/goship/db/ent/monthlysubscription"
	"github.com/leomorpho/goship/db/ent/profile"
	"github.com/rs/zerolog/log"
)

type EntStore struct {
	orm                            *ent.Client
	proTrialTimespanInDays         time.Duration
	paymentFailedGracePeriodInDays time.Duration
}

func NewEntStore(orm *ent.Client, proTrialTimespanInDays, paymentFailedGracePeriodInDays int) *EntStore {
	return &EntStore{
		orm:                            orm,
		proTrialTimespanInDays:         time.Duration(proTrialTimespanInDays) * 24 * time.Hour,
		paymentFailedGracePeriodInDays: time.Duration(paymentFailedGracePeriodInDays) * 24 * time.Hour,
	}
}

func (s *EntStore) CreateSubscription(ctx context.Context, tx any, profileID int) (err error) {
	var entTx *ent.Tx
	commit := false
	if tx == nil {
		commit = true
		entTx, err = s.orm.Tx(ctx)
		if err != nil {
			return err
		}
	} else {
		var ok bool
		entTx, ok = tx.(*ent.Tx)
		if !ok {
			return fmt.Errorf("unsupported transaction type %T; expected *ent.Tx", tx)
		}
	}
	err = entTx.MonthlySubscription.
		Create().
		SetProduct(monthlysubscription.Product(paidsubscriptions.ProductTypePro.Value)).
		SetPayerID(profileID).
		AddBenefactorIDs(profileID).
		SetIsTrial(true).
		SetIsActive(true).
		SetStartedAt(time.Now()).
		SetExpiredOn(time.Now().Add(s.proTrialTimespanInDays)).
		Exec(ctx)
	if commit {
		if err != nil {
			_ = entTx.Rollback()
		}
		return entTx.Commit()
	}
	return err
}

func (s *EntStore) DeactivateExpiredSubscriptions(ctx context.Context) error {
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

func (s *EntStore) UpdateToPaidPro(ctx context.Context, profileID int) error {
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
			SetProduct(monthlysubscription.Product(paidsubscriptions.ProductTypePro.Value)).
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
		SetProduct(monthlysubscription.Product(paidsubscriptions.ProductTypePro.Value)).
		SetPayerID(profileID).
		AddBenefactorIDs(profileID).
		SetIsTrial(false).
		SetIsActive(true).
		SetStartedAt(time.Now()).
		Exec(ctx)
}

func (s *EntStore) GetCurrentlyActiveProduct(ctx context.Context, profileID int) (*paidsubscriptions.ProductType, *time.Time, bool, error) {
	sub, err := s.orm.MonthlySubscription.
		Query().
		Where(
			monthlysubscription.IsActive(true),
			monthlysubscription.HasPayerWith(profile.IDEQ(profileID)),
		).
		Only(ctx)

	if ent.IsNotFound(err) {
		return &paidsubscriptions.ProductTypeFree, nil, false, nil
	}
	if ent.IsNotSingular(err) {
		log.Error().Err(err)
		return nil, nil, false, nil
	}
	return paidsubscriptions.ParseProductType(string(sub.Product)), sub.ExpiredOn, sub.IsTrial, nil
}

func (s *EntStore) StoreStripeCustomerID(ctx context.Context, profileID int, stripeCustomerID string) error {
	prof, err := s.orm.Profile.Get(ctx, profileID)
	if err != nil {
		return err
	}
	if prof.StripeID == stripeCustomerID {
		return nil
	}
	_, err = s.orm.Profile.UpdateOneID(profileID).SetStripeID(stripeCustomerID).Save(ctx)
	return err
}

func (s *EntStore) GetProfileIDFromStripeCustomerID(ctx context.Context, stripeCustomerID string) (int, error) {
	prof, err := s.orm.Profile.Query().
		Where(profile.StripeIDEQ(stripeCustomerID)).
		Select(profile.FieldID).
		Only(ctx)
	if err != nil {
		return 0, err
	}
	return prof.ID, nil
}

func (s *EntStore) CancelWithGracePeriod(ctx context.Context, profileID int) error {
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

func (s *EntStore) CancelOrRenew(ctx context.Context, profileID int, cancelDate *time.Time) error {
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

func (s *EntStore) UpdateToFree(ctx context.Context, profileID int) error {
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
		return nil
	} else if count > 1 {
		return errors.New("there should only ever be 1 active subscription for a profile")
	}

	_, err = s.orm.MonthlySubscription.Update().
		Where(
			monthlysubscription.HasPayerWith(profile.IDEQ(profileID)),
			monthlysubscription.IsActive(true),
		).
		SetExpiredOn(time.Now()).
		SetIsActive(false).
		Save(ctx)
	return err
}
