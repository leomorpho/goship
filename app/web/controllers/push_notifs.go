package controllers

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	profilesvc "github.com/leomorpho/goship/app/profile"
	routeNames "github.com/leomorpho/goship/app/web/routenames"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/framework/dberrors"
	"github.com/leomorpho/goship/framework/domain"
	"github.com/leomorpho/goship/framework/repos/uxflashmessages"

	"github.com/leomorpho/goship-modules/notifications"
	"github.com/leomorpho/goship/app/views"
	"github.com/leomorpho/goship/app/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/views/web/pages/gen"
	"github.com/leomorpho/goship/app/web/viewmodels"
	"github.com/rs/zerolog/log"
)

type outgoingNotifications struct {
	ctr                           ui.Controller
	profileService                *profilesvc.ProfileService
	pwaPushService                *notifications.PwaPushService
	fcmPushService                *notifications.FcmPushService
	notificationPermissionService *notifications.NotificationPermissionService
}

func NewPushNotifsRoute(
	ctr ui.Controller,
	profileService *profilesvc.ProfileService,
	pwaPushService *notifications.PwaPushService,
	fcmPushService *notifications.FcmPushService,
	notificationPermissionService *notifications.NotificationPermissionService,
) outgoingNotifications {
	return outgoingNotifications{
		ctr:                           ctr,
		profileService:                profileService,
		pwaPushService:                pwaPushService,
		fcmPushService:                fcmPushService,
		notificationPermissionService: notificationPermissionService,
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
	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}

	subscribedEndpoints, err := c.pwaPushService.GetPushSubscriptionEndpoints(ctx.Request().Context(), profileID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Could not fetch subscription endpoints")
	}

	// Create an instance of PushNotificationSubscriptions and populate it
	subscriptions := viewmodels.PushNotificationSubscriptions{
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

	var req PushSubscriptionRequest
	if err := ctx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}

	// If the permission is specified, use it, else add all permissions.
	if notifPermission != nil {
		err := c.notificationPermissionService.CreatePermission(
			ctx.Request().Context(), profileID, *notifPermission, platform)
		if err != nil {
			log.Error().Err(err).Msg("failed to create notification permission")
		}
	} else {
		for _, perm := range domain.NotificationPermissions.Members() {
			err := c.notificationPermissionService.CreatePermission(
				ctx.Request().Context(), profileID, perm, platform)
			if err != nil {
				log.Error().Err(err).Msg("failed to create notification permission")
			}

		}
	}

	switch *platform {
	case domain.NotificationPlatformPush:

		hasPushPermissionsAlready, err := c.pwaPushService.HasPermissionsLeftAndEndpointIsRegistered(
			ctx.Request().Context(), profileID, req.Endpoint)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "error checking if user still has push permissions and registered endpoints left")
		}
		if !hasPushPermissionsAlready {
			// Create or update the subscription in the database
			err := c.pwaPushService.AddPushSubscription(
				ctx.Request().Context(),
				profileID,
				notifications.Subscription{Endpoint: req.Endpoint, P256dh: req.Keys.P256dh, Auth: req.Keys.Auth},
			)

			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "could not register push subscription")
			}
		}
	case domain.NotificationPlatformFCMPush:

		hasPushPermissionsAlready, err := c.fcmPushService.HasPermissionsLeftAndTokenIsRegistered(
			ctx.Request().Context(), profileID, req.FCMToken)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "error checking if user still has push permissions and registered fcm subscriptions left")
		}
		if !hasPushPermissionsAlready {
			// Create or update the subscription in the database
			err := c.fcmPushService.AddPushSubscription(
				ctx.Request().Context(),
				profileID,
				notifications.FcmSubscription{
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

	var req PushSubscriptionRequest
	if err := ctx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}

	// If the permission is specified, use it, else delete all permissions.
	if notifPermission != nil {
		err := c.notificationPermissionService.DeletePermission(
			ctx.Request().Context(), profileID, *notifPermission, platform, nil)
		if err != nil {
			log.Error().Err(err).Msg("failed to delete notification permission")
		}
	} else {
		for _, perm := range domain.NotificationPermissions.Members() {
			err := c.notificationPermissionService.DeletePermission(
				ctx.Request().Context(), profileID, perm, platform, nil)
			if err != nil {
				log.Error().Err(err).Msg("failed to delete notification permission")
			}

		}
	}

	switch *platform {
	case domain.NotificationPlatformPush:

		// Check if the user has any push permissions still set up, else delete all
		hasPushPermissionsLeft, err := c.pwaPushService.HasPermissionsLeftAndEndpointIsRegistered(
			ctx.Request().Context(), profileID, req.Endpoint)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "error checking if user still has push permissions and registered pwa endpoints left")
		}
		if !hasPushPermissionsLeft {
			// Delete the subscription in the database based on the endpoint
			// This assumes your subscriptions are uniquely identified by their endpoint in your database.
			err := c.pwaPushService.DeletePushSubscriptionByEndpoint(
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
		hasPushPermissionsLeft, err := c.fcmPushService.HasPermissionsLeftAndTokenIsRegistered(
			ctx.Request().Context(), profileID, req.FCMToken)

		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "error checking if user still has push permissions and registered fcm registration left")
		}
		if !hasPushPermissionsLeft {
			// Delete the subscription in the database based on the endpoint
			// This assumes your subscriptions are uniquely identified by their endpoint in your database.
			err := c.fcmPushService.DeletePushSubscriptionByToken(
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

	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}
	profileData, err := c.profileService.GetProfileSettingsByID(ctx.Request().Context(), profileID)
	if err != nil {
		return err
	}

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

	permissionErr := c.notificationPermissionService.DeletePermission(
		ctx.Request().Context(), profileID, *notifPermission, &domain.NotificationPlatformEmail, &token)

	page, err := c.createNotificationsPage(ctx, profileData.ID, profileData.PhoneNumberE164, profileData.PhoneVerified)
	if err != nil {
		return err
	}

	// TODO: lol this error handling is growse, refactor later.
	if permissionErr != nil {
		log.Error().Err(permissionErr).
			Int("profileID", profileData.ID).
			Str("notifPermission", notifPermission.Value).
			Str("platform", domain.NotificationPlatformEmail.Value).
			Str("token", token).
			Msg("failed to delete email notification permission")
		if dberrors.IsNotFound(permissionErr) {
			if notifName != "" {
				uxflashmessages.Warning(ctx, fmt.Sprintf("You already unsubscribed email notifications for %s.", notifName))

			} else {
				uxflashmessages.Warning(ctx, "You already unsubscribed for this email notifications.")
			}
		} else {
			uxflashmessages.Danger(ctx, "Something went wrong on our end. Feel free to manually unsubscribe below.")
		}

		return c.ctr.RenderPage(ctx, *page)
	}

	if notifName != "" {
		uxflashmessages.Success(ctx, fmt.Sprintf("You successfully unsubscribed email notifications for %s.", notifName))
	}

	return c.ctr.RenderPage(ctx, *page)
}

func (c *outgoingNotifications) createNotificationsPage(
	ctx echo.Context,
	profileID int,
	phoneNumberE164 string,
	phoneVerified bool,
) (*ui.Page, error) {
	// Create response page
	// TODO: create a route to get notif permissions with the below, and use it in prefs. Redirect
	// to new page below, setting the same message.
	page := ui.NewPage(ctx)

	permissions, err := c.notificationPermissionService.GetPermissions(ctx.Request().Context(), profileID)
	subscribedEndpoints, err := c.pwaPushService.GetPushSubscriptionEndpoints(ctx.Request().Context(), profileID)
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

	notificationPermissions := viewmodels.NotificationPermissionsData{
		VapidPublicKey:                c.ctr.Container.Config.App.VapidPublicKey,
		PermissionDailyNotif:          permissions[domain.NotificationPermissionDailyReminder],
		PermissionPartnerActivity:     permissions[domain.NotificationPermissionNewFriendActivity],
		SubscribedEndpoints:           subscribedEndpoints,
		PhoneSubscriptionEnabled:      phoneNumberE164 != "" && phoneVerified,
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

// 	var req PermissionRequest
// 	if err := ctx.Bind(&req); err != nil {
// 		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
// 	}

// 	permission := domain.NotificationPermissions.Parse(req.Permission)
// 	err := c.notificationPermissionService.DeletePermission(
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
