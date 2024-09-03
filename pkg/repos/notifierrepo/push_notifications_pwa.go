package notifierrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/mikestefanello/pagoda/ent"
	"github.com/mikestefanello/pagoda/ent/notificationpermission"
	"github.com/mikestefanello/pagoda/ent/profile"
	"github.com/mikestefanello/pagoda/ent/pwapushsubscription"
	"github.com/mikestefanello/pagoda/pkg/domain"
)

type Subscription struct {
	Endpoint string
	P256dh   string
	Auth     string
}

type VAPIDDetails struct {
	PublicKey  string
	PrivateKey string
}

type PwaPushNotificationsRepo struct {
	vapidDetails                   *VAPIDDetails
	subscriberEmail                string
	orm                            *ent.Client
	notificationSendPermissionRepo *NotificationSendPermissionRepo
}

func NewPwaPushNotificationsRepo(
	orm *ent.Client, vapidPublicKey, vapidPrivateKey, subscriberEmail string,
) *PwaPushNotificationsRepo {

	notificationSendPermissionRepo := NewNotificationSendPermissionRepo(orm)
	return &PwaPushNotificationsRepo{
		vapidDetails: &VAPIDDetails{
			PublicKey:  vapidPublicKey,
			PrivateKey: vapidPrivateKey,
		},
		subscriberEmail:                subscriberEmail,
		orm:                            orm,
		notificationSendPermissionRepo: notificationSendPermissionRepo,
	}
}

func (p *PwaPushNotificationsRepo) AddPushSubscription(ctx context.Context, profileID int, sub Subscription) error {
	_, err := p.orm.PwaPushSubscription.
		Create().
		SetProfileID(profileID).
		SetEndpoint(sub.Endpoint).
		SetP256dh(sub.P256dh).
		SetAuth(sub.Auth).
		Save(ctx)
	return err
}

func (p *PwaPushNotificationsRepo) SendPushNotifications(ctx context.Context, profileID int, title, message string, numUnreadNotifs int) error {
	subs, err := p.orm.Profile.
		Query().
		Where(profile.IDEQ(profileID)).
		QueryPwaPushSubscriptions().
		All(ctx)
	if err != nil {
		return err
	}
	var invalidSubscriptions []*ent.PwaPushSubscription

	payload := map[string]string{
		"title":       title,
		"body":        message,
		"unreadCount": fmt.Sprintf("%d", numUnreadNotifs),
		// "url":   url,
	}
	payloadBytes, _ := json.Marshal(payload)

	for _, sub := range subs {
		pushSub := webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{
				P256dh: sub.P256dh,
				Auth:   sub.Auth,
			},
		}

		// !!!!!!TODO: this should be a task and not run in the main coroutine!
		resp, err := webpush.SendNotification(payloadBytes, &pushSub, &webpush.Options{
			Subscriber:      p.subscriberEmail,
			VAPIDPublicKey:  p.vapidDetails.PublicKey,
			VAPIDPrivateKey: p.vapidDetails.PrivateKey,
		})
		if err != nil {
			if resp != nil {
				if resp.StatusCode == http.StatusGone ||
					resp.StatusCode == http.StatusNotFound ||
					resp.StatusCode == http.StatusBadRequest {
					// These status codes suggest the subscription is no longer valid
					invalidSubscriptions = append(invalidSubscriptions, sub)
				}
				resp.Body.Close() // Always close the response body
			}
			log.Error().Err(err).
				Str("endpoint", sub.Endpoint).
				Msg("failed to send push notification for this endpoint")
			continue // Skip to the next subscription
		}
		log.Debug().Err(err).
			Str("endpoint", sub.Endpoint).
			Msg("Sent push notification to endpoint")
		defer resp.Body.Close()
	}

	// Cleanup invalid subscriptions
	if len(invalidSubscriptions) > 0 {
		for _, sub := range invalidSubscriptions {
			if err := p.DeletePushSubscriptionByEndpoint(ctx, sub.ProfileID, sub.Endpoint); err != nil {
				return err // Handle or log failure to delete subscription
			}
		}
	}
	return nil
}

func (p *PwaPushNotificationsRepo) GetPushSubscriptionEndpoints(ctx context.Context, profileID int) ([]string, error) {
	subs, err := p.orm.Profile.
		Query().
		Where(profile.IDEQ(profileID)).
		QueryPwaPushSubscriptions().
		All(ctx)
	if err != nil {
		return nil, err
	}

	var subscribedEndpoints []string
	for _, sub := range subs {
		subscribedEndpoints = append(subscribedEndpoints, sub.Endpoint)
	}
	return subscribedEndpoints, nil
}

func (p *PwaPushNotificationsRepo) DeletePushSubscriptionByEndpoint(ctx context.Context, profileID int, endpoint string) error {
	_, err := p.orm.PwaPushSubscription.Delete().
		Where(
			pwapushsubscription.HasProfileWith(profile.IDEQ(profileID)),
			pwapushsubscription.EndpointEQ(endpoint),
		).
		Exec(ctx)

	return err
}

func (p *PwaPushNotificationsRepo) hasProfilePushSubscriptionEndpoints(ctx context.Context, profileID int) (bool, error) {
	return p.orm.PwaPushSubscription.
		Query().
		Where(
			pwapushsubscription.HasProfileWith(profile.IDEQ(profileID)),
		).
		Exist(ctx)
}

// TODO: this is bad design, this repo should know NOTHING about permissions
func (p *PwaPushNotificationsRepo) HasPermissionsLeftAndEndpointIsRegistered(
	ctx context.Context,
	profileID int,
	endpoint string,
) (bool, error) {
	// Check if the endpoint exists
	exists, err := p.orm.PwaPushSubscription.
		Query().
		Where(
			pwapushsubscription.HasProfileWith(profile.IDEQ(profileID)),
			pwapushsubscription.EndpointEQ(endpoint),
		).
		Exist(ctx)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}

	// Check if there are any permissions for the given platform
	count, err := p.orm.NotificationPermission.
		Query().
		Where(
			notificationpermission.HasProfileWith(profile.IDEQ(profileID)),
			notificationpermission.PlatformEQ(notificationpermission.Platform(domain.NotificationPlatformPush.Value)),
		).
		Count(ctx)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}
