package emailsubscriptions

import (
	"context"

	"github.com/leomorpho/goship/db/ent"
	"github.com/leomorpho/goship/db/ent/emailsubscription"
	"github.com/leomorpho/goship/db/ent/emailsubscriptiontype"
	"github.com/rs/zerolog/log"
)

// EntStore adapts goship Ent models to the extracted module Store contract.
type EntStore struct {
	orm *ent.Client
}

func NewEntStore(orm *ent.Client) *EntStore {
	return &EntStore{orm: orm}
}

func (s *EntStore) CreateList(ctx context.Context, emailList List) error {
	_, err := s.orm.EmailSubscriptionType.
		Create().SetActive(true).SetName(emailsubscriptiontype.Name(emailList)).Save(ctx)
	if ent.IsConstraintError(err) {
		return nil
	}
	return err
}

func (s *EntStore) Subscribe(
	ctx context.Context, email string, emailList List, latitude, longitude *float64,
) (*Subscription, error) {
	if (latitude != nil && longitude == nil) || (latitude == nil && longitude != nil) {
		log.Fatal().Str("error", "both latitude/longitude should either be nil or have a value")
	}

	alreadySubscribed, err := s.orm.EmailSubscription.
		Query().
		Where(emailsubscription.EmailEQ(email)).
		QuerySubscriptions().
		Where(emailsubscriptiontype.NameEQ(emailsubscriptiontype.Name(emailList))).
		Exist(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return nil, err
	}
	if alreadySubscribed {
		return nil, &ErrAlreadySubscribed{Err: err, EmailList: string(emailList)}
	}

	subscriptionEmailListObj, err := s.orm.EmailSubscriptionType.
		Query().
		Where(
			emailsubscriptiontype.Active(true),
			emailsubscriptiontype.NameEQ(emailsubscriptiontype.Name(emailList)),
		).Only(ctx)
	if err != nil {
		return nil, err
	}

	existingSubscription, err := s.orm.EmailSubscription.
		Query().
		Where(emailsubscription.EmailEQ(email)).
		Only(ctx)
	if ent.IsNotFound(err) {
		confirmationCode, genErr := generateUniqueCode()
		if genErr != nil {
			return nil, genErr
		}

		createQuery := s.orm.EmailSubscription.
			Create().
			SetEmail(email).
			SetConfirmationCode(confirmationCode).
			AddSubscriptions(subscriptionEmailListObj)
		if latitude != nil && longitude != nil {
			createQuery.SetLatitude(*latitude).SetLongitude(*longitude)
		}

		existingSubscription, err = createQuery.Save(ctx)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	updateQuery := s.orm.EmailSubscription.
		UpdateOne(existingSubscription).
		AddSubscriptions(subscriptionEmailListObj)
	if latitude != nil && longitude != nil {
		updateQuery.SetLatitude(*latitude).SetLongitude(*longitude)
	}
	_, err = updateQuery.Save(ctx)

	return &Subscription{
		ID:               existingSubscription.ID,
		Email:            existingSubscription.Email,
		Verified:         existingSubscription.Verified,
		ConfirmationCode: existingSubscription.ConfirmationCode,
		Lat:              existingSubscription.Latitude,
		Lon:              existingSubscription.Longitude,
	}, err
}

func (s *EntStore) Unsubscribe(ctx context.Context, email string, token string, emailList List) error {
	getSubscriptionQuery := s.orm.EmailSubscription.
		Query().
		Where(emailsubscription.EmailEQ(email)).
		WithSubscriptions()
	subscription, err := getSubscriptionQuery.Only(ctx)
	if err != nil {
		return err
	}

	subscriptionEmailListObj, err := s.orm.EmailSubscriptionType.
		Query().
		Where(emailsubscriptiontype.NameEQ(emailsubscriptiontype.Name(emailList))).
		Only(ctx)
	if err != nil {
		return err
	}

	_, err = subscription.Update().RemoveSubscriptions(subscriptionEmailListObj).Save(ctx)
	if err != nil {
		return nil
	}

	subscription, err = getSubscriptionQuery.Only(ctx)
	if err != nil {
		return err
	}

	if len(subscription.Edges.Subscriptions) == 0 {
		return s.orm.EmailSubscription.DeleteOne(subscription).Exec(ctx)
	}

	confirmationCode, err := generateUniqueCode()
	if err != nil {
		return err
	}
	_, err = subscription.Update().SetConfirmationCode(confirmationCode).Save(ctx)
	return err
}

func (s *EntStore) Confirm(ctx context.Context, code string) error {
	subscription, err := s.orm.EmailSubscription.
		Query().
		Where(emailsubscription.ConfirmationCodeEQ(code)).
		Only(ctx)
	if err != nil {
		return ErrInvalidEmailConfirmationCode
	}
	if subscription.Verified {
		return nil
	}

	confirmationCode, err := generateUniqueCode()
	if err != nil {
		return err
	}
	_, err = subscription.Update().SetVerified(true).SetConfirmationCode(confirmationCode).Save(ctx)
	return err
}
