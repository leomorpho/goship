package notifications

import (
	"context"
	"errors"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
	"log/slog"
)

type FcmSubscription struct {
	Token string
}

type FcmPushService struct {
	store     fcmPushSubscriptionStore
	fcmClient *messaging.Client
}

type fcmPushSubscriptionRecord struct {
	ProfileID int
	Token     string
}

type fcmPushSubscriptionStore interface {
	addSubscription(ctx context.Context, profileID int, token string) error
	listSubscriptions(ctx context.Context, profileID int) ([]fcmPushSubscriptionRecord, error)
	deleteByToken(ctx context.Context, profileID int, token string) error
	hasAnyByProfileID(ctx context.Context, profileID int) (bool, error)
	hasToken(ctx context.Context, profileID int, token string) (bool, error)
}

func newFcmPushService(
	store fcmPushSubscriptionStore,
	fcmClient *messaging.Client,
) *FcmPushService {
	return &FcmPushService{
		store:     store,
		fcmClient: fcmClient,
	}
}

func newFcmPushServiceWithStore(
	store fcmPushSubscriptionStore,
	firebaseJSONAccessKeys *[]byte,
) (*FcmPushService, error) {

	var fcmClient *messaging.Client

	if firebaseJSONAccessKeys != nil {
		opt := option.WithCredentialsJSON(*firebaseJSONAccessKeys)
		app, err := firebase.NewApp(context.Background(), nil, opt)
		if err != nil {
			return nil, fmt.Errorf("error initializing firebase app: %w", err)
		}

		fcmClient, err = app.Messaging(context.Background())
		if err != nil {
			return nil, fmt.Errorf("error initializing firebase messaging client: %w", err)
		}
	}

	if store == nil {
		return nil, errors.New("fcm store must be set")
	}

	return newFcmPushService(store, fcmClient), nil
}

func (p *FcmPushService) AddPushSubscription(ctx context.Context, profileID int, sub FcmSubscription) error {
	return p.store.addSubscription(ctx, profileID, sub.Token)
}

func (p *FcmPushService) SendPushNotifications(ctx context.Context, profileID int, title, message string, numUnreadNotifs int, sendSound bool) error {
	if p.fcmClient == nil {
		slog.Warn("No FCM client is set, not actually sending any real messages")
		return nil
	}

	subs, err := p.store.listSubscriptions(ctx, profileID)
	if err != nil {
		return err
	}

	invalidSubscriptions := make([]string, 0)

	var sound string
	if sendSound {
		sound = "default"
	}
	for _, sub := range subs {
		slog.Debug("Sending FCM push notification", "token", sub.Token)
		msg := &messaging.Message{
			APNS: &messaging.APNSConfig{
				Payload: &messaging.APNSPayload{
					Aps: &messaging.Aps{
						Alert: &messaging.ApsAlert{
							Title: title,
							Body:  message,
						},
						Badge: &numUnreadNotifs,
						Sound: sound,
					},
				},
			},

			Token: sub.Token,
			// Data:  payload,
		}

		resp, err := p.fcmClient.Send(ctx, msg)
		if err != nil {
			// Handle invalid tokens and log error
			if messaging.IsUnregistered(err) || messaging.IsInvalidArgument(err) {
				slog.Warn("Invalid FCM token, marking for cleanup", "error", err, "token", sub.Token)
				invalidSubscriptions = append(invalidSubscriptions, sub.Token)
			} else {
				slog.Error("failed to send FCM push notification for this token", "error", err, "token", sub.Token)
			}
		}
		slog.Debug("Sent FCM push notification to token", "token", sub.Token, "response", resp)
	}

	// Cleanup invalid subscriptions
	for _, sub := range invalidSubscriptions {
		if err := p.DeletePushSubscriptionByToken(ctx, profileID, sub); err != nil {
			slog.Error("Failed to delete invalid FCM subscription", "error", err, "token", sub)
			return err // Handle or log failure to delete subscription
		}
	}

	return nil
}

func (p *FcmPushService) DeletePushSubscriptionByToken(ctx context.Context, profileID int, token string) error {
	return p.store.deleteByToken(ctx, profileID, token)
}

func (p *FcmPushService) hasProfilePushSubscriptions(ctx context.Context, profileID int) (bool, error) {
	return p.store.hasAnyByProfileID(ctx, profileID)
}

func (p *FcmPushService) HasTokenRegistered(
	ctx context.Context,
	profileID int,
	token string,
) (bool, error) {
	return p.store.hasToken(ctx, profileID, token)
}
