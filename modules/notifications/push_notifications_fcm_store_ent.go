package notifications

import (
	"context"

	"github.com/leomorpho/goship/db/ent"
	"github.com/leomorpho/goship/db/ent/fcmsubscriptions"
	"github.com/leomorpho/goship/db/ent/profile"
)

type entFcmPushSubscriptionStore struct {
	orm *ent.Client
}

func NewFcmPushService(
	orm *ent.Client, firebaseJSONAccessKeys *[]byte,
) (*FcmPushService, error) {
	return newFcmPushServiceWithStore(
		&entFcmPushSubscriptionStore{orm: orm},
		NewNotificationPermissionService(orm),
		firebaseJSONAccessKeys,
	)
}

func (s *entFcmPushSubscriptionStore) addSubscription(
	ctx context.Context, profileID int, token string,
) error {
	_, err := s.orm.FCMSubscriptions.
		Create().
		SetProfileID(profileID).
		SetToken(token).
		Save(ctx)
	return err
}

func (s *entFcmPushSubscriptionStore) listSubscriptions(
	ctx context.Context, profileID int,
) ([]fcmPushSubscriptionRecord, error) {
	subs, err := s.orm.Profile.
		Query().
		Where(profile.IDEQ(profileID)).
		QueryFcmPushSubscriptions().
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]fcmPushSubscriptionRecord, 0, len(subs))
	for _, sub := range subs {
		out = append(out, fcmPushSubscriptionRecord{
			ProfileID: sub.ProfileID,
			Token:     sub.Token,
		})
	}
	return out, nil
}

func (s *entFcmPushSubscriptionStore) deleteByToken(
	ctx context.Context, profileID int, token string,
) error {
	_, err := s.orm.FCMSubscriptions.Delete().
		Where(
			fcmsubscriptions.HasProfileWith(profile.IDEQ(profileID)),
			fcmsubscriptions.TokenEQ(token),
		).
		Exec(ctx)
	return err
}

func (s *entFcmPushSubscriptionStore) hasAnyByProfileID(
	ctx context.Context, profileID int,
) (bool, error) {
	return s.orm.FCMSubscriptions.
		Query().
		Where(fcmsubscriptions.HasProfileWith(profile.IDEQ(profileID))).
		Exist(ctx)
}

func (s *entFcmPushSubscriptionStore) hasToken(
	ctx context.Context, profileID int, token string,
) (bool, error) {
	return s.orm.FCMSubscriptions.
		Query().
		Where(
			fcmsubscriptions.HasProfileWith(profile.IDEQ(profileID)),
			fcmsubscriptions.TokenEQ(token),
		).
		Exist(ctx)
}
