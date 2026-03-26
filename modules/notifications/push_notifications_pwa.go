package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/SherClockHolmes/webpush-go"
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

type PwaPushService struct {
	vapidDetails    *VAPIDDetails
	subscriberEmail string
	store           pwaPushSubscriptionStore
}

type pwaPushSubscriptionRecord struct {
	ProfileID int
	Endpoint  string
	P256dh    string
	Auth      string
}

type pwaPushSubscriptionStore interface {
	addSubscription(ctx context.Context, profileID int, sub Subscription) error
	listSubscriptions(ctx context.Context, profileID int) ([]pwaPushSubscriptionRecord, error)
	deleteByEndpoint(ctx context.Context, profileID int, endpoint string) error
	hasAnyByProfileID(ctx context.Context, profileID int) (bool, error)
	hasEndpoint(ctx context.Context, profileID int, endpoint string) (bool, error)
}

func newPwaPushService(
	store pwaPushSubscriptionStore,
	vapidPublicKey, vapidPrivateKey, subscriberEmail string,
) *PwaPushService {
	return &PwaPushService{
		vapidDetails: &VAPIDDetails{
			PublicKey:  vapidPublicKey,
			PrivateKey: vapidPrivateKey,
		},
		subscriberEmail: subscriberEmail,
		store:           store,
	}
}

func (p *PwaPushService) AddPushSubscription(ctx context.Context, profileID int, sub Subscription) error {
	return p.store.addSubscription(ctx, profileID, sub)
}

func (p *PwaPushService) SendPushNotifications(ctx context.Context, profileID int, title, message string, numUnreadNotifs int) error {
	subs, err := p.store.listSubscriptions(ctx, profileID)
	if err != nil {
		return err
	}
	var invalidSubscriptions []string

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
					invalidSubscriptions = append(invalidSubscriptions, sub.Endpoint)
				}
				resp.Body.Close() // Always close the response body
			}
			slog.Error("failed to send push notification for this endpoint", "error", err, "endpoint", sub.Endpoint)
			continue // Skip to the next subscription
		}
		slog.Debug("Sent push notification to endpoint", "endpoint", sub.Endpoint)
		defer resp.Body.Close()
	}

	// Cleanup invalid subscriptions
	if len(invalidSubscriptions) > 0 {
		for _, sub := range invalidSubscriptions {
			if err := p.DeletePushSubscriptionByEndpoint(ctx, profileID, sub); err != nil {
				return err // Handle or log failure to delete subscription
			}
		}
	}
	return nil
}

func (p *PwaPushService) GetPushSubscriptionEndpoints(ctx context.Context, profileID int) ([]string, error) {
	subs, err := p.store.listSubscriptions(ctx, profileID)
	if err != nil {
		return nil, err
	}

	var subscribedEndpoints []string
	for _, sub := range subs {
		subscribedEndpoints = append(subscribedEndpoints, sub.Endpoint)
	}
	return subscribedEndpoints, nil
}

func (p *PwaPushService) DeletePushSubscriptionByEndpoint(ctx context.Context, profileID int, endpoint string) error {
	return p.store.deleteByEndpoint(ctx, profileID, endpoint)
}

func (p *PwaPushService) hasProfilePushSubscriptionEndpoints(ctx context.Context, profileID int) (bool, error) {
	return p.store.hasAnyByProfileID(ctx, profileID)
}

func (p *PwaPushService) HasEndpointRegistered(
	ctx context.Context,
	profileID int,
	endpoint string,
) (bool, error) {
	return p.store.hasEndpoint(ctx, profileID, endpoint)
}
