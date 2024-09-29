package routes

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/mikestefanello/pagoda/ent"
	"github.com/mikestefanello/pagoda/pkg/context"
	"github.com/mikestefanello/pagoda/pkg/controller"
	"github.com/mikestefanello/pagoda/pkg/domain"
	"github.com/mikestefanello/pagoda/pkg/repos/msg"
	routeNames "github.com/mikestefanello/pagoda/pkg/routing/routenames"

	"github.com/mikestefanello/pagoda/pkg/repos/notifierrepo"
	"github.com/mikestefanello/pagoda/pkg/types"
	"github.com/mikestefanello/pagoda/templates"
	"github.com/mikestefanello/pagoda/templates/layouts"
	"github.com/mikestefanello/pagoda/templates/pages"
	"github.com/rs/zerolog/log"
)

type outgoingNotifications struct {
	ctr                            controller.Controller
	pwaPushNotificationsRepo       *notifierrepo.PwaPushNotificationsRepo
	fcmPushNotificationsRepo       *notifierrepo.FcmPushNotificationsRepo
	notificationSendPermissionRepo *notifierrepo.NotificationSendPermissionRepo
}

func NewPushNotifsRoute(
	ctr controller.Controller,
	pwaPushNotificationsRepo *notifierrepo.PwaPushNotificationsRepo,
	fcmPushNotificationsRepo *notifierrepo.FcmPushNotificationsRepo,
	notificationSendPermissionRepo *notifierrepo.NotificationSendPermissionRepo,
) outgoingNotifications {
	return outgoingNotifications{
		ctr:                            ctr,
		pwaPushNotificationsRepo:       pwaPushNotificationsRepo,
		fcmPushNotificationsRepo:       fcmPushNotificationsRepo,
		notificationSendPermissionRepo: notificationSendPermissionRepo,
	}
}

type PushSubscriptionRequest struct {
	// PWA push subscription param
	Endpoint string `json:"endpoint,omitempty"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys,omitempty"`

	// FCM subscription param
	FCMToken string `json:"fcm_token,omitempty"`
}

func (c *outgoingNotifications) GetPushSubscriptions(ctx echo.Context) error {
	usr := ctx.Get(context.AuthenticatedUserKey).(*ent.User)

	// TODO: why is WithGendersInterestedIn used below?
	profile := usr.QueryProfile().FirstX(ctx.Request().Context())

	subscribedEndpoints, err := c.pwaPushNotificationsRepo.GetPushSubscriptionEndpoints(ctx.Request().Context(), profile.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Could not fetch subscription endpoints")
	}

	// Create an instance of PushNotificationSubscriptions and populate it
	subscriptions := types.PushNotificationSubscriptions{
		URLs: subscribedEndpoints,
	}

	return ctx.JSON(http.StatusOK, subscriptions)
}

func (c *outgoingNotifications) RegisterSubscription(ctx echo.Context) error {
	plaftormStr := ctx.Param("platform")
	if plaftormStr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid profile ID")
	}

	platform := domain.NotificationPlatforms.Parse(plaftormStr)

	notificationType := ctx.QueryParam(domain.PermissionNotificationType)
	var notifPermission *domain.NotificationPermissionType

	if notificationType != "" {
		notifPermission = domain.NotificationPermissions.Parse(notificationType)
	}

	usr := ctx.Get(context.AuthenticatedUserKey).(*ent.User)
	if usr == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "User must be logged in")
	}

	var req PushSubscriptionRequest
	if err := ctx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	profileID := usr.QueryProfile().FirstX(ctx.Request().Context()).ID

	// If the permission is specified, use it, else add all permissions.
	if notifPermission != nil {
		err := c.notificationSendPermissionRepo.CreatePermission(
			ctx.Request().Context(), profileID, *notifPermission, platform)
		if err != nil {
			log.Error().Err(err).Msg("failed to create notification permission")
		}
	} else {
		for _, perm := range domain.NotificationPermissions.Members() {
			err := c.notificationSendPermissionRepo.CreatePermission(
				ctx.Request().Context(), profileID, perm, platform)
			if err != nil {
				log.Error().Err(err).Msg("failed to create notification permission")
			}

		}
	}

	switch *platform {
	case domain.NotificationPlatformPush:

		hasPushPermissionsAlready, err := c.pwaPushNotificationsRepo.HasPermissionsLeftAndEndpointIsRegistered(
			ctx.Request().Context(), profileID, req.Endpoint)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "error checking if user still has push permissions and registered endpoints left")
		}
		if !hasPushPermissionsAlready {
			// Create or update the subscription in the database
			err := c.pwaPushNotificationsRepo.AddPushSubscription(
				ctx.Request().Context(),
				profileID,
				notifierrepo.Subscription{Endpoint: req.Endpoint, P256dh: req.Keys.P256dh, Auth: req.Keys.Auth},
			)

			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "could not register push subscription")
			}
		}
	case domain.NotificationPlatformFCMPush:

		hasPushPermissionsAlready, err := c.fcmPushNotificationsRepo.HasPermissionsLeftAndTokenIsRegistered(
			ctx.Request().Context(), profileID, req.FCMToken)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "error checking if user still has push permissions and registered fcm subscriptions left")
		}
		if !hasPushPermissionsAlready {
			// Create or update the subscription in the database
			err := c.fcmPushNotificationsRepo.AddPushSubscription(
				ctx.Request().Context(),
				profileID,
				notifierrepo.FcmSubscription{
					Token: req.FCMToken,
				},
			)

			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "could not register fcm subscription")
			}
		}
	}

	return ctx.NoContent(http.StatusOK)
}

func (c *outgoingNotifications) DeleteSubscription(ctx echo.Context) error {
	plaftormStr := ctx.Param("platform")
	if plaftormStr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid profile ID")
	}

	platform := domain.NotificationPlatforms.Parse(plaftormStr)
	notificationType := ctx.QueryParam(domain.PermissionNotificationType)

	var notifPermission *domain.NotificationPermissionType

	if notificationType != "" {

		notifPermission = domain.NotificationPermissions.Parse(notificationType)
	}

	usr := ctx.Get(context.AuthenticatedUserKey).(*ent.User)
	if usr == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "User must be logged in")
	}

	var req PushSubscriptionRequest
	if err := ctx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	profileID := usr.QueryProfile().FirstX(ctx.Request().Context()).ID

	// If the permission is specified, use it, else delete all permissions.
	if notifPermission != nil {
		err := c.notificationSendPermissionRepo.DeletePermission(
			ctx.Request().Context(), profileID, *notifPermission, platform, nil)
		if err != nil {
			log.Error().Err(err).Msg("failed to delete notification permission")
		}
	} else {
		for _, perm := range domain.NotificationPermissions.Members() {
			err := c.notificationSendPermissionRepo.DeletePermission(
				ctx.Request().Context(), profileID, perm, platform, nil)
			if err != nil {
				log.Error().Err(err).Msg("failed to delete notification permission")
			}

		}
	}

	switch *platform {
	case domain.NotificationPlatformPush:

		// Check if the user has any push permissions still set up, else delete all
		hasPushPermissionsLeft, err := c.pwaPushNotificationsRepo.HasPermissionsLeftAndEndpointIsRegistered(
			ctx.Request().Context(), profileID, req.Endpoint)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "error checking if user still has push permissions and registered pwa endpoints left")
		}
		if !hasPushPermissionsLeft {
			// Delete the subscription in the database based on the endpoint
			// This assumes your subscriptions are uniquely identified by their endpoint in your database.
			err := c.pwaPushNotificationsRepo.DeletePushSubscriptionByEndpoint(
				ctx.Request().Context(),
				profileID,
				req.Endpoint,
			)

			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete pwa push subscription")
			}
		}
	case domain.NotificationPlatformFCMPush:
		// Check if the user has any push permissions still set up, else delete all
		hasPushPermissionsLeft, err := c.fcmPushNotificationsRepo.HasPermissionsLeftAndTokenIsRegistered(
			ctx.Request().Context(), profileID, req.FCMToken)

		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "error checking if user still has push permissions and registered fcm registration left")
		}
		if !hasPushPermissionsLeft {
			// Delete the subscription in the database based on the endpoint
			// This assumes your subscriptions are uniquely identified by their endpoint in your database.
			err := c.fcmPushNotificationsRepo.DeletePushSubscriptionByToken(
				ctx.Request().Context(),
				profileID,
				req.FCMToken,
			)

			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete fcm push subscription")
			}
		}
	}

	return ctx.NoContent(http.StatusOK)
}

func (c *outgoingNotifications) DeleteEmailSubscription(ctx echo.Context) error {
	token := ctx.Param("token")
	if token == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid token")
	}

	notificationType := ctx.Param(domain.PermissionNotificationType)
	if notificationType == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid notification type")
	}

	notifPermission := domain.NotificationPermissions.Parse(notificationType)

	usr := ctx.Get(context.AuthenticatedUserKey).(*ent.User)
	if usr == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "User must be logged in")
	}

	profileEnt := usr.QueryProfile().FirstX(ctx.Request().Context())

	var notifName string

	if *notifPermission == domain.NotificationPermissionDailyReminder {
		notifName = "daily updates"
	} else if *notifPermission == domain.NotificationPermissionNewFriendActivity {
		notifName = "partner activity"
	} else {
		log.Error().
			Str("notifPermission", notifPermission.Value).
			Msg("no notification exists with that name")
	}

	permissionErr := c.notificationSendPermissionRepo.DeletePermission(
		ctx.Request().Context(), profileEnt.ID, *notifPermission, &domain.NotificationPlatformEmail, &token)

	page, err := c.createNotificationsPage(ctx, profileEnt)
	if err != nil {
		return err
	}

	// TODO: lol this error handling is growse, refactor later.
	if permissionErr != nil {
		log.Error().Err(permissionErr).
			Int("profileID", profileEnt.ID).
			Str("notifPermission", notifPermission.Value).
			Str("platform", domain.NotificationPlatformEmail.Value).
			Str("token", token).
			Msg("failed to delete email notification permission")
		if ent.IsNotFound(permissionErr) {
			if notifName != "" {
				msg.Warning(ctx, fmt.Sprintf("You already unsubscribed email notifications for %s.", notifName))

			} else {
				msg.Warning(ctx, "You already unsubscribed for this email notifications.")
			}
		} else {
			msg.Danger(ctx, "Something went wrong on our end. Feel free to manually unsubscribe below.")
		}

		return c.ctr.RenderPage(ctx, *page)
	}

	if notifName != "" {
		msg.Success(ctx, fmt.Sprintf("You successfully unsubscribed email notifications for %s.", notifName))
	}

	return c.ctr.RenderPage(ctx, *page)
}

func (c *outgoingNotifications) createNotificationsPage(ctx echo.Context, profileEnt *ent.Profile) (*controller.Page, error) {
	// Create response page
	// TODO: create a route to get notif permissions with the below, and use it in prefs. Redirect
	// to new page below, setting the same message.
	page := controller.NewPage(ctx)

	permissions, err := c.notificationSendPermissionRepo.GetPermissions(ctx.Request().Context(), profileEnt.ID)
	subscribedEndpoints, err := c.pwaPushNotificationsRepo.GetPushSubscriptionEndpoints(ctx.Request().Context(), profileEnt.ID)
	if err != nil {
		return nil, err
	}
	addPushSubscriptionEndpoint := fmt.Sprintf("%s%s",
		c.ctr.Container.Config.HTTP.Domain, ctx.Echo().Reverse(
			routeNames.RouteNameRegisterSubscription, domain.NotificationPlatformPush.Value)) + "?csrf=" + page.CSRF
	deletePushSubscriptionEndpoint := fmt.Sprintf("%s%s",
		c.ctr.Container.Config.HTTP.Domain, ctx.Echo().Reverse(
			routeNames.RouteNameDeleteSubscription, domain.NotificationPlatformPush.Value)) + "?csrf=" + page.CSRF

	addEmailSubscriptionEndpoint := fmt.Sprintf("%s%s",
		c.ctr.Container.Config.HTTP.Domain, ctx.Echo().Reverse(
			routeNames.RouteNameRegisterSubscription, domain.NotificationPlatformEmail.Value)) + "?csrf=" + page.CSRF
	deleteEmailSubscriptionEndpoint := fmt.Sprintf("%s%s",
		c.ctr.Container.Config.HTTP.Domain, ctx.Echo().Reverse(
			routeNames.RouteNameDeleteSubscription, domain.NotificationPlatformEmail.Value)) + "?csrf=" + page.CSRF

	addSmsSubscriptionEndpoint := fmt.Sprintf("%s%s",
		c.ctr.Container.Config.HTTP.Domain, ctx.Echo().Reverse(
			routeNames.RouteNameRegisterSubscription, domain.NotificationPlatformSMS.Value)) + "?csrf=" + page.CSRF
	deleteSmsSubscriptionEndpoint := fmt.Sprintf("%s%s",
		c.ctr.Container.Config.HTTP.Domain, ctx.Echo().Reverse(
			routeNames.RouteNameDeleteSubscription, domain.NotificationPlatformSMS.Value)) + "?csrf=" + page.CSRF

	notificationPermissions := types.NotificationPermissionsData{
		VapidPublicKey:                c.ctr.Container.Config.App.VapidPublicKey,
		PermissionDailyNotif:          permissions[domain.NotificationPermissionDailyReminder],
		PermissionPartnerActivity:     permissions[domain.NotificationPermissionNewFriendActivity],
		SubscribedEndpoints:           subscribedEndpoints,
		PhoneSubscriptionEnabled:      profileEnt.PhoneNumberE164 != "" && profileEnt.PhoneVerified,
		NotificationTypeQueryParamKey: domain.PermissionNotificationType,

		AddPushSubscriptionEndpoint:     addPushSubscriptionEndpoint,
		DeletePushSubscriptionEndpoint:  deletePushSubscriptionEndpoint,
		AddEmailSubscriptionEndpoint:    addEmailSubscriptionEndpoint,
		DeleteEmailSubscriptionEndpoint: deleteEmailSubscriptionEndpoint,
		AddSmsSubscriptionEndpoint:      addSmsSubscriptionEndpoint,
		DeleteSmsSubscriptionEndpoint:   deleteSmsSubscriptionEndpoint,
	}

	page.Layout = layouts.Main
	page.Component = pages.NotificationPermissions(&page, notificationPermissions)
	page.Name = templates.PagePreferences
	return &page, nil
}

// type PermissionRequest struct {
// 	Permission string `json:"permission"`
// }

// func (c *outgoingNotifications) CreateNotificationPermissions(ctx echo.Context) error {
// 	usr := ctx.Get(context.AuthenticatedUserKey).(*ent.User)
// 	if usr == nil {
// 		return echo.NewHTTPError(http.StatusUnauthorized, "User must be logged in")
// 	}
// 	profileID := usr.QueryProfile().FirstX(ctx.Request().Context()).ID

// 	var req PermissionRequest
// 	if err := ctx.Bind(&req); err != nil {
// 		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
// 	}

// 	permission := domain.NotificationPermissions.Parse(req.Permission)
// 	err := c.notificationSendPermissionRepo.CreatePermission(
// 		ctx.Request().Context(),
// 		profileID,
// 		*permission,
// 		&domain.NotificationPlatformPush,
// 	)

// 	if err != nil {
// 		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete push subscription")
// 	}

// 	return ctx.NoContent(http.StatusOK)
// }

// func (c *outgoingNotifications) DeleteNotificationPermissions(ctx echo.Context) error {
// 	usr := ctx.Get(context.AuthenticatedUserKey).(*ent.User)
// 	if usr == nil {
// 		return echo.NewHTTPError(http.StatusUnauthorized, "User must be logged in")
// 	}
// 	profileID := usr.QueryProfile().FirstX(ctx.Request().Context()).ID

// 	var req PermissionRequest
// 	if err := ctx.Bind(&req); err != nil {
// 		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
// 	}

// 	permission := domain.NotificationPermissions.Parse(req.Permission)
// 	err := c.notificationSendPermissionRepo.DeletePermission(
// 		ctx.Request().Context(),
// 		profileID,
// 		*permission,
// 		&domain.NotificationPlatformPush,
// 	)

// 	if err != nil {
// 		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete push subscription")
// 	}

// 	return ctx.NoContent(http.StatusOK)
// }
