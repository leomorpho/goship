package notifications

import (
	"context"
	"errors"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/leomorpho/goship/framework/domain"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/option"
)

type FcmSubscription struct {
	Token string
}

type FcmPushService struct {
	store                         fcmPushSubscriptionStore
	fcmClient                     *messaging.Client
	notificationPermissionService *NotificationPermissionService
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
	permissionService *NotificationPermissionService,
	fcmClient *messaging.Client,
) *FcmPushService {
	return &FcmPushService{
		store:                         store,
		fcmClient:                     fcmClient,
		notificationPermissionService: permissionService,
	}
}

func newFcmPushServiceWithStore(
	store fcmPushSubscriptionStore,
	permissionService *NotificationPermissionService,
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
	if permissionService == nil {
		return nil, errors.New("notification permission service must be set")
	}

	return newFcmPushService(store, permissionService, fcmClient), nil
}

func (p *FcmPushService) AddPushSubscription(ctx context.Context, profileID int, sub FcmSubscription) error {
	return p.store.addSubscription(ctx, profileID, sub.Token)
}

func (p *FcmPushService) SendPushNotifications(ctx context.Context, profileID int, title, message string, numUnreadNotifs int, sendSound bool) error {
	if p.fcmClient == nil {
		log.Warn().Msg("No FCM client is set, not actually sending any real messages")
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
		log.Debug().Str("token", sub.Token).Msg("Sending FCM push notification")
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
				log.Warn().Err(err).Str("token", sub.Token).Msg("Invalid FCM token, marking for cleanup")
				invalidSubscriptions = append(invalidSubscriptions, sub.Token)
			} else {
				log.Error().Err(err).
					Str("token", sub.Token).
					Msg("failed to send FCM push notification for this token")
			}
		}
		log.Debug().Str("token", sub.Token).
			Str("response", resp).
			Msg("Sent FCM push notification to token")
	}

	// Cleanup invalid subscriptions
	for _, sub := range invalidSubscriptions {
		if err := p.DeletePushSubscriptionByToken(ctx, profileID, sub); err != nil {
			log.Error().Err(err).Str("token", sub).Msg("Failed to delete invalid FCM subscription")
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

// TODO: this is bad design, this repo should know NOTHING about permissions
func (p *FcmPushService) HasPermissionsLeftAndTokenIsRegistered(
	ctx context.Context,
	profileID int,
	token string,
) (bool, error) {
	// Check if the endpoint exists
	exists, err := p.store.hasToken(ctx, profileID, token)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}

	// Check if there are any permissions for the given platform.
	hasPerms, err := p.notificationPermissionService.HasPermissionsForPlatform(
		ctx, profileID, domain.NotificationPlatformFCMPush,
	)
	if err != nil {
		return false, err
	}

	return hasPerms, nil
}
