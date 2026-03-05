package notifications

import (
	"context"

	"github.com/leomorpho/goship/db/ent"
	"github.com/leomorpho/goship/db/ent/profile"
	"github.com/leomorpho/goship/db/ent/pwapushsubscription"
)

type entPwaPushSubscriptionStore struct {
	orm *ent.Client
}

func NewPwaPushService(
	orm *ent.Client, vapidPublicKey, vapidPrivateKey, subscriberEmail string,
) *PwaPushService {
	return newPwaPushService(
		&entPwaPushSubscriptionStore{orm: orm},
		NewNotificationPermissionService(orm),
		vapidPublicKey,
		vapidPrivateKey,
		subscriberEmail,
	)
}

func (s *entPwaPushSubscriptionStore) addSubscription(
	ctx context.Context, profileID int, sub Subscription,
) error {
	_, err := s.orm.PwaPushSubscription.
		Create().
		SetProfileID(profileID).
		SetEndpoint(sub.Endpoint).
		SetP256dh(sub.P256dh).
		SetAuth(sub.Auth).
		Save(ctx)
	return err
}

func (s *entPwaPushSubscriptionStore) listSubscriptions(
	ctx context.Context, profileID int,
) ([]pwaPushSubscriptionRecord, error) {
	subs, err := s.orm.Profile.
		Query().
		Where(profile.IDEQ(profileID)).
		QueryPwaPushSubscriptions().
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]pwaPushSubscriptionRecord, 0, len(subs))
	for _, sub := range subs {
		out = append(out, pwaPushSubscriptionRecord{
			ProfileID: sub.ProfileID,
			Endpoint:  sub.Endpoint,
			P256dh:    sub.P256dh,
			Auth:      sub.Auth,
		})
	}
	return out, nil
}

func (s *entPwaPushSubscriptionStore) deleteByEndpoint(
	ctx context.Context, profileID int, endpoint string,
) error {
	_, err := s.orm.PwaPushSubscription.Delete().
		Where(
			pwapushsubscription.HasProfileWith(profile.IDEQ(profileID)),
			pwapushsubscription.EndpointEQ(endpoint),
		).
		Exec(ctx)
	return err
}

func (s *entPwaPushSubscriptionStore) hasAnyByProfileID(
	ctx context.Context, profileID int,
) (bool, error) {
	return s.orm.PwaPushSubscription.
		Query().
		Where(pwapushsubscription.HasProfileWith(profile.IDEQ(profileID))).
		Exist(ctx)
}

func (s *entPwaPushSubscriptionStore) hasEndpoint(
	ctx context.Context, profileID int, endpoint string,
) (bool, error) {
	return s.orm.PwaPushSubscription.
		Query().
		Where(
			pwapushsubscription.HasProfileWith(profile.IDEQ(profileID)),
			pwapushsubscription.EndpointEQ(endpoint),
		).
		Exist(ctx)
}
