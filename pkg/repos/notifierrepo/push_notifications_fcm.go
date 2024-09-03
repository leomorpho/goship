package notifierrepo

import (
	"context"
	"errors"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/mikestefanello/pagoda/ent"
	"github.com/mikestefanello/pagoda/ent/fcmsubscriptions"
	"github.com/mikestefanello/pagoda/ent/notificationpermission"
	"github.com/mikestefanello/pagoda/ent/profile"
	"github.com/mikestefanello/pagoda/pkg/domain"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/option"
)

type FcmSubscription struct {
	Token string
}

type FcmPushNotificationsRepo struct {
	orm       *ent.Client
	fcmClient *messaging.Client
}

func NewFcmPushNotificationsRepo(
	orm *ent.Client, firebaseJSONAccessKeys *[]byte,
) (*FcmPushNotificationsRepo, error) {

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

	if orm == nil {
		return nil, errors.New("orm must be set")
	}

	return &FcmPushNotificationsRepo{
		orm:       orm,
		fcmClient: fcmClient,
	}, nil
}

func (p *FcmPushNotificationsRepo) AddPushSubscription(ctx context.Context, profileID int, sub FcmSubscription) error {
	_, err := p.orm.FCMSubscriptions.
		Create().
		SetProfileID(profileID).
		SetToken(sub.Token).
		Save(ctx)
	return err
}

func (p *FcmPushNotificationsRepo) SendPushNotifications(ctx context.Context, profileID int, title, message string, numUnreadNotifs int, sendSound bool) error {
	if p.fcmClient == nil {
		log.Warn().Msg("No FCM client is set, not actually sending any real messages")
		return nil
	}

	subs, err := p.orm.Profile.
		Query().
		Where(profile.IDEQ(profileID)).
		QueryFcmPushSubscriptions().
		All(ctx)
	if err != nil {
		return err
	}

	invalidSubscriptions := make([]*ent.FCMSubscriptions, 0)

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
				invalidSubscriptions = append(invalidSubscriptions, sub)
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
		if err := p.DeletePushSubscriptionByToken(ctx, sub.ProfileID, sub.Token); err != nil {
			log.Error().Err(err).Str("token", sub.Token).Msg("Failed to delete invalid FCM subscription")
			return err // Handle or log failure to delete subscription
		}
	}

	return nil
}

func (p *FcmPushNotificationsRepo) DeletePushSubscriptionByToken(ctx context.Context, profileID int, token string) error {
	_, err := p.orm.FCMSubscriptions.Delete().
		Where(
			fcmsubscriptions.HasProfileWith(profile.IDEQ(profileID)),
			fcmsubscriptions.TokenEQ(token),
		).
		Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (p *FcmPushNotificationsRepo) hasProfilePushSubscriptions(ctx context.Context, profileID int) (bool, error) {
	return p.orm.FCMSubscriptions.
		Query().
		Where(
			fcmsubscriptions.HasProfileWith(profile.IDEQ(profileID)),
		).
		Exist(ctx)
}

// TODO: this is bad design, this repo should know NOTHING about permissions
func (p *FcmPushNotificationsRepo) HasPermissionsLeftAndTokenIsRegistered(
	ctx context.Context,
	profileID int,
	token string,
) (bool, error) {
	// Check if the endpoint exists
	exists, err := p.orm.FCMSubscriptions.
		Query().
		Where(
			fcmsubscriptions.HasProfileWith(profile.IDEQ(profileID)),
			fcmsubscriptions.TokenEQ(token),
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
			notificationpermission.PlatformEQ(notificationpermission.Platform(domain.NotificationPlatformFCMPush.Value)),
		).
		Count(ctx)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}
