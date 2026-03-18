package routes

import (
	stdcontext "context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	notifications "github.com/leomorpho/goship-modules/notifications"
	routeNames "github.com/leomorpho/goship/app/web/routenames"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/framework/core"
	"github.com/leomorpho/goship/framework/dberrors"
	"github.com/leomorpho/goship/framework/domain"
	"github.com/leomorpho/goship/framework/repos/uxflashmessages"
	profilesvc "github.com/leomorpho/goship/modules/profile"

	"github.com/leomorpho/goship/app/views"
	"github.com/leomorpho/goship/app/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/views/web/pages/gen"
	"github.com/leomorpho/goship/app/web/viewmodels"
	customctx "github.com/leomorpho/goship/framework/context"
)

const notificationQueryParam = "notif"

type notificationCountReader interface {
	GetCountOfUnseenNotifications(ctx stdcontext.Context, profileID int) (int, error)
}

type RouteModuleDeps struct {
	Controller                    ui.Controller
	ProfileService                *profilesvc.ProfileService
	NotifierService               *notifications.NotifierService
	PwaPushService                *notifications.PwaPushService
	FcmPushService                *notifications.FcmPushService
	NotificationPermissionService *notifications.NotificationPermissionService
}

type RouteModule struct {
	controller                    ui.Controller
	profileService                *profilesvc.ProfileService
	notifierService               *notifications.NotifierService
	pwaPushService                *notifications.PwaPushService
	fcmPushService                *notifications.FcmPushService
	notificationPermissionService *notifications.NotificationPermissionService
}

func NewRouteModule(deps RouteModuleDeps) *RouteModule {
	return &RouteModule{
		controller:                    deps.Controller,
		profileService:                deps.ProfileService,
		notifierService:               deps.NotifierService,
		pwaPushService:                deps.PwaPushService,
		fcmPushService:                deps.FcmPushService,
		notificationPermissionService: deps.NotificationPermissionService,
	}
}

func (m *RouteModule) ID() string {
	return "notifications"
}

func (m *RouteModule) Migrations() fs.FS {
	return nil
}

func (m *RouteModule) RegisterRoutes(r core.Router) error {
	normalNotifications := NewNormalNotificationsRoute(m.controller, m.notifierService)
	r.GET("/notifications", normalNotifications.Get).Name = routeNames.RouteNameNotifications
	r.GET("/notifications/mark-all-read", normalNotifications.MarkAllAsRead).Name = routeNames.RouteNameMarkAllNotificationsAsRead
	r.DELETE("/notifications/:notification_id", normalNotifications.Delete).Name = routeNames.RouteNameDeleteNotification

	normalNotificationsCount := NewNormalNotificationsCountRoute(m.controller, m.profileService)
	r.GET("/notifications/normalNotificationsCount", normalNotificationsCount.Get).Name = routeNames.RouteNameNormalNotificationsCount

	markRead := NewMarkNormalNotificationReadRoute(m.controller, m.notifierService)
	r.POST("/notifications/:notification_id/read", markRead.Post).Name = routeNames.RouteNameMarkNotificationsAsRead

	markUnread := NewMarkNormalNotificationUnreadRoute(m.controller, m.notifierService)
	r.POST("/notifications/unread", markUnread.Post).Name = routeNames.RouteNameMarkNotificationsAsUnread

	return nil
}

func (m *RouteModule) RegisterOnboardingRoutes(r core.Router) error {
	outgoingNotifications := NewPushNotifsRoute(
		m.controller,
		m.profileService,
		m.pwaPushService,
		m.fcmPushService,
		m.notificationPermissionService,
	)
	r.GET("/subscription/push", outgoingNotifications.GetPushSubscriptions).Name = routeNames.RouteNameGetPushSubscriptions
	r.POST("/subscription/:platform", outgoingNotifications.RegisterSubscription).Name = routeNames.RouteNameRegisterSubscription
	r.DELETE("/subscription/:platform", outgoingNotifications.DeleteSubscription).Name = routeNames.RouteNameDeleteSubscription
	r.GET("/email-subscription/unsubscribe/:permission/:token", outgoingNotifications.DeleteEmailSubscription).Name = routeNames.RouteNameDeleteEmailSubscriptionWithToken
	return nil
}

type NormalNotificationsCountRoute struct {
	controller     ui.Controller
	profileService notificationCountReader
}

func NewNormalNotificationsCountRoute(controller ui.Controller, profileService notificationCountReader) *NormalNotificationsCountRoute {
	return &NormalNotificationsCountRoute{
		controller:     controller,
		profileService: profileService,
	}
}

func (r *NormalNotificationsCountRoute) Get(ctx echo.Context) error {
	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}

	num, err := r.profileService.GetCountOfUnseenNotifications(ctx.Request().Context(), profileID)
	if err != nil {
		return err
	}
	if num == 0 {
		return ctx.String(http.StatusOK, "<span class='hidden'>0</span>")
	}
	return ctx.String(http.StatusOK, fmt.Sprintf("<span>%d</span>", num))
}

type NormalNotificationsRoute struct {
	controller      ui.Controller
	notifierService *notifications.NotifierService
}

func NewNormalNotificationsRoute(controller ui.Controller, notifierService *notifications.NotifierService) *NormalNotificationsRoute {
	return &NormalNotificationsRoute{
		controller:      controller,
		notifierService: notifierService,
	}
}

func (r *NormalNotificationsRoute) Get(ctx echo.Context) error {
	timestampParam := ctx.QueryParam("timestamp")
	var timestamp *time.Time
	if timestampParam != "" {
		parsedTime, err := time.Parse(time.RFC3339Nano, timestampParam)
		if err != nil {
			slog.Error("invalid timestamp format", "convo", "invalid timestamp format")
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid timestamp format")
		}
		timestamp = &parsedTime
	}

	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Name = templates.PageNotifications
	page.Component = pages.NotificationsPage(&page)
	page.HTMX.Request.Boosted = true
	page.ShowBottomNavbar = true
	page.SelectedBottomNavbarItem = domain.BottomNavbarItemNotifications

	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}

	notifications, err := r.notifierService.GetNotifications(
		ctx.Request().Context(),
		profileID,
		false,
		timestamp,
		&r.controller.Container.Config.App.PageSize,
	)
	if err != nil {
		return err
	}

	for _, notification := range notifications {
		if notification == nil {
			continue
		}
		if buttonText, ok := domain.NotificationCenterButtonText[notification.Type]; ok {
			notification.ButtonText = buttonText
		} else {
			notification.ButtonText = "See more"
		}
	}

	if len(notifications) == 0 && timestamp != nil {
		return nil
	}

	var nextPageURL string
	if len(notifications) > 0 {
		oldestTimestamp := notifications[len(notifications)-1].CreatedAt
		nextPageURL = ctx.Echo().Reverse(routeNames.RouteNameNotifications) + "?timestamp=" + oldestTimestamp.Format(time.RFC3339Nano)
	}

	data := viewmodels.NewNormalNotificationsPageData()
	data.Notifications = viewmodels.NotificationItemsFromDomain(notifications)
	data.NextPageURL = nextPageURL
	page.Data = data

	return r.controller.RenderPage(ctx, page)
}

func (r *NormalNotificationsRoute) MarkAllAsRead(ctx echo.Context) error {
	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}

	if err := r.notifierService.MarkAllNotificationRead(ctx.Request().Context(), profileID); err != nil {
		return err
	}

	return r.Get(ctx)
}

func (r *NormalNotificationsRoute) Delete(ctx echo.Context) error {
	notificationID, err := strconv.Atoi(ctx.Param("notification_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid question ID")
	}

	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}

	if err := r.notifierService.DeleteNotification(ctx.Request().Context(), notificationID, &profileID); err != nil {
		return err
	}

	return ctx.String(http.StatusOK, "")
}

type MarkNormalNotificationReadRoute struct {
	controller      ui.Controller
	notifierService *notifications.NotifierService
}

func NewMarkNormalNotificationReadRoute(controller ui.Controller, notifierService *notifications.NotifierService) *MarkNormalNotificationReadRoute {
	return &MarkNormalNotificationReadRoute{
		controller:      controller,
		notifierService: notifierService,
	}
}

func (r *MarkNormalNotificationReadRoute) Post(ctx echo.Context) error {
	notificationID, err := strconv.Atoi(ctx.Param("notification_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid notification ID")
	}

	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}

	if err := r.notifierService.MarkNotificationRead(ctx.Request().Context(), notificationID, &profileID); err != nil {
		return err
	}

	return ctx.String(http.StatusOK, "")
}

type MarkNormalNotificationUnreadRoute struct {
	controller      ui.Controller
	notifierService *notifications.NotifierService
}

func NewMarkNormalNotificationUnreadRoute(controller ui.Controller, notifierService *notifications.NotifierService) *MarkNormalNotificationUnreadRoute {
	return &MarkNormalNotificationUnreadRoute{
		controller:      controller,
		notifierService: notifierService,
	}
}

type SeenEventRequest struct {
	ID int `form:"id" validate:"required"`
}

func (r *MarkNormalNotificationUnreadRoute) Post(ctx echo.Context) error {
	var req SeenEventRequest
	if err := ctx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request")
	}

	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}

	if err := r.notifierService.MarkNotificationUnread(ctx.Request().Context(), req.ID, &profileID); err != nil {
		return err
	}

	return r.controller.Redirect(ctx, routeNames.RouteNameNotifications)
}

type OutgoingNotificationsRoute struct {
	controller                    ui.Controller
	profileService                *profilesvc.ProfileService
	pwaPushService                *notifications.PwaPushService
	fcmPushService                *notifications.FcmPushService
	notificationPermissionService *notifications.NotificationPermissionService
}

func NewPushNotifsRoute(
	controller ui.Controller,
	profileService *profilesvc.ProfileService,
	pwaPushService *notifications.PwaPushService,
	fcmPushService *notifications.FcmPushService,
	notificationPermissionService *notifications.NotificationPermissionService,
) *OutgoingNotificationsRoute {
	return &OutgoingNotificationsRoute{
		controller:                    controller,
		profileService:                profileService,
		pwaPushService:                pwaPushService,
		fcmPushService:                fcmPushService,
		notificationPermissionService: notificationPermissionService,
	}
}

type PushSubscriptionRequest struct {
	Endpoint string `json:"endpoint,omitempty"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys,omitempty"`
	FCMToken string `json:"fcm_token,omitempty"`
}

func (r *OutgoingNotificationsRoute) GetPushSubscriptions(ctx echo.Context) error {
	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}

	subscribedEndpoints, err := r.pwaPushService.GetPushSubscriptionEndpoints(ctx.Request().Context(), profileID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Could not fetch subscription endpoints")
	}

	subscriptions := viewmodels.NewPushNotificationSubscriptions()
	subscriptions.URLs = subscribedEndpoints
	return ctx.JSON(http.StatusOK, subscriptions)
}

func (r *OutgoingNotificationsRoute) RegisterSubscription(ctx echo.Context) error {
	platformStr := ctx.Param("platform")
	if platformStr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid profile ID")
	}

	platform := domain.NotificationPlatforms.Parse(platformStr)
	notificationType := ctx.QueryParam(domain.PermissionNotificationType)
	var permission *domain.NotificationPermissionType
	if notificationType != "" {
		permission = domain.NotificationPermissions.Parse(notificationType)
	}

	var req PushSubscriptionRequest
	if err := ctx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}

	if permission != nil {
		if err := r.notificationPermissionService.CreatePermission(ctx.Request().Context(), profileID, *permission, platform); err != nil {
			slog.Error("failed to create notification permission", "error", err)
		}
	} else {
		for _, perm := range domain.NotificationPermissions.Members() {
			if err := r.notificationPermissionService.CreatePermission(ctx.Request().Context(), profileID, perm, platform); err != nil {
				slog.Error("failed to create notification permission", "error", err)
			}
		}
	}

	switch *platform {
	case domain.NotificationPlatformPush:
		hasPermissionsAlready, err := r.pwaPushService.HasPermissionsLeftAndEndpointIsRegistered(ctx.Request().Context(), profileID, req.Endpoint)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "error checking if user still has push permissions and registered endpoints left")
		}
		if !hasPermissionsAlready {
			if err := r.pwaPushService.AddPushSubscription(
				ctx.Request().Context(),
				profileID,
				notifications.Subscription{Endpoint: req.Endpoint, P256dh: req.Keys.P256dh, Auth: req.Keys.Auth},
			); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "could not register push subscription")
			}
		}

	case domain.NotificationPlatformFCMPush:
		hasPermissionsAlready, err := r.fcmPushService.HasPermissionsLeftAndTokenIsRegistered(ctx.Request().Context(), profileID, req.FCMToken)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "error checking if user still has push permissions and registered fcm subscriptions left")
		}
		if !hasPermissionsAlready {
			if err := r.fcmPushService.AddPushSubscription(
				ctx.Request().Context(),
				profileID,
				notifications.FcmSubscription{Token: req.FCMToken},
			); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "could not register fcm subscription")
			}
		}
	}

	return ctx.NoContent(http.StatusOK)
}

func (r *OutgoingNotificationsRoute) DeleteSubscription(ctx echo.Context) error {
	platformStr := ctx.Param("platform")
	if platformStr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid profile ID")
	}

	platform := domain.NotificationPlatforms.Parse(platformStr)
	notificationType := ctx.QueryParam(domain.PermissionNotificationType)
	var permission *domain.NotificationPermissionType
	if notificationType != "" {
		permission = domain.NotificationPermissions.Parse(notificationType)
	}

	var req PushSubscriptionRequest
	if err := ctx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}

	if permission != nil {
		if err := r.notificationPermissionService.DeletePermission(ctx.Request().Context(), profileID, *permission, platform, nil); err != nil {
			slog.Error("failed to delete notification permission", "error", err)
		}
	} else {
		for _, perm := range domain.NotificationPermissions.Members() {
			if err := r.notificationPermissionService.DeletePermission(ctx.Request().Context(), profileID, perm, platform, nil); err != nil {
				slog.Error("failed to delete notification permission", "error", err)
			}
		}
	}

	switch *platform {
	case domain.NotificationPlatformPush:
		hasPermissionsLeft, err := r.pwaPushService.HasPermissionsLeftAndEndpointIsRegistered(ctx.Request().Context(), profileID, req.Endpoint)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "error checking if user still has push permissions and registered pwa endpoints left")
		}
		if !hasPermissionsLeft {
			if err := r.pwaPushService.DeletePushSubscriptionByEndpoint(ctx.Request().Context(), profileID, req.Endpoint); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete pwa push subscription")
			}
		}

	case domain.NotificationPlatformFCMPush:
		hasPermissionsLeft, err := r.fcmPushService.HasPermissionsLeftAndTokenIsRegistered(ctx.Request().Context(), profileID, req.FCMToken)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "error checking if user still has push permissions and registered fcm registration left")
		}
		if !hasPermissionsLeft {
			if err := r.fcmPushService.DeletePushSubscriptionByToken(ctx.Request().Context(), profileID, req.FCMToken); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete fcm push subscription")
			}
		}
	}

	return ctx.NoContent(http.StatusOK)
}

func (r *OutgoingNotificationsRoute) DeleteEmailSubscription(ctx echo.Context) error {
	token := ctx.Param("token")
	if token == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid token")
	}

	notificationType := ctx.Param(domain.PermissionNotificationType)
	if notificationType == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid notification type")
	}

	permission := domain.NotificationPermissions.Parse(notificationType)
	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}

	profileData, err := r.profileService.GetProfileSettingsByID(ctx.Request().Context(), profileID)
	if err != nil {
		return err
	}

	var notificationName string
	switch *permission {
	case domain.NotificationPermissionDailyReminder:
		notificationName = "daily updates"
	case domain.NotificationPermissionNewFriendActivity:
		notificationName = "partner activity"
	default:
		slog.Error("no notification exists with that name", "notifPermission", permission.Value)
	}

	permissionErr := r.notificationPermissionService.DeletePermission(
		ctx.Request().Context(),
		profileID,
		*permission,
		&domain.NotificationPlatformEmail,
		&token,
	)

	page, err := r.createNotificationsPage(ctx, profileData.ID, profileData.PhoneNumberE164, profileData.PhoneVerified)
	if err != nil {
		return err
	}

	if permissionErr != nil {
		slog.Error("failed to delete email notification permission",
			"error", permissionErr,
			"profileID", profileData.ID,
			"notifPermission", permission.Value,
			"platform", domain.NotificationPlatformEmail.Value,
			"token", token,
		)

		if dberrors.IsNotFound(permissionErr) {
			if notificationName != "" {
				uxflashmessages.Warning(ctx, fmt.Sprintf("You already unsubscribed email notifications for %s.", notificationName))
			} else {
				uxflashmessages.Warning(ctx, "You already unsubscribed for this email notifications.")
			}
		} else {
			uxflashmessages.Danger(ctx, "Something went wrong on our end. Feel free to manually unsubscribe below.")
		}

		return r.controller.RenderPage(ctx, *page)
	}

	if notificationName != "" {
		uxflashmessages.Success(ctx, fmt.Sprintf("You successfully unsubscribed email notifications for %s.", notificationName))
	}

	return r.controller.RenderPage(ctx, *page)
}

func (r *OutgoingNotificationsRoute) createNotificationsPage(
	ctx echo.Context,
	profileID int,
	phoneNumberE164 string,
	phoneVerified bool,
) (*ui.Page, error) {
	page := ui.NewPage(ctx)

	permissions, err := r.notificationPermissionService.GetPermissions(ctx.Request().Context(), profileID)
	if err != nil {
		return nil, err
	}
	subscribedEndpoints, err := r.pwaPushService.GetPushSubscriptionEndpoints(ctx.Request().Context(), profileID)
	if err != nil {
		return nil, err
	}

	addPushSubscriptionEndpoint := fmt.Sprintf(
		"%s%s",
		r.controller.Container.Config.HTTP.Domain,
		ctx.Echo().Reverse(routeNames.RouteNameRegisterSubscription, domain.NotificationPlatformPush.Value),
	) + "?csrf=" + page.CSRF
	deletePushSubscriptionEndpoint := fmt.Sprintf(
		"%s%s",
		r.controller.Container.Config.HTTP.Domain,
		ctx.Echo().Reverse(routeNames.RouteNameDeleteSubscription, domain.NotificationPlatformPush.Value),
	) + "?csrf=" + page.CSRF

	addEmailSubscriptionEndpoint := fmt.Sprintf(
		"%s%s",
		r.controller.Container.Config.HTTP.Domain,
		ctx.Echo().Reverse(routeNames.RouteNameRegisterSubscription, domain.NotificationPlatformEmail.Value),
	) + "?csrf=" + page.CSRF
	deleteEmailSubscriptionEndpoint := fmt.Sprintf(
		"%s%s",
		r.controller.Container.Config.HTTP.Domain,
		ctx.Echo().Reverse(routeNames.RouteNameDeleteSubscription, domain.NotificationPlatformEmail.Value),
	) + "?csrf=" + page.CSRF

	addSMSSubscriptionEndpoint := fmt.Sprintf(
		"%s%s",
		r.controller.Container.Config.HTTP.Domain,
		ctx.Echo().Reverse(routeNames.RouteNameRegisterSubscription, domain.NotificationPlatformSMS.Value),
	) + "?csrf=" + page.CSRF
	deleteSMSSubscriptionEndpoint := fmt.Sprintf(
		"%s%s",
		r.controller.Container.Config.HTTP.Domain,
		ctx.Echo().Reverse(routeNames.RouteNameDeleteSubscription, domain.NotificationPlatformSMS.Value),
	) + "?csrf=" + page.CSRF

	notificationPermissions := viewmodels.NewNotificationPermissionsData()
	notificationPermissions.VapidPublicKey = r.controller.Container.Config.App.VapidPublicKey
	notificationPermissions.PermissionDailyNotif = permissions[domain.NotificationPermissionDailyReminder]
	notificationPermissions.PermissionPartnerActivity = permissions[domain.NotificationPermissionNewFriendActivity]
	notificationPermissions.SubscribedEndpoints = subscribedEndpoints
	notificationPermissions.PhoneSubscriptionEnabled = phoneNumberE164 != "" && phoneVerified
	notificationPermissions.NotificationTypeQueryParamKey = domain.PermissionNotificationType
	notificationPermissions.AddPushSubscriptionEndpoint = addPushSubscriptionEndpoint
	notificationPermissions.DeletePushSubscriptionEndpoint = deletePushSubscriptionEndpoint
	notificationPermissions.AddEmailSubscriptionEndpoint = addEmailSubscriptionEndpoint
	notificationPermissions.DeleteEmailSubscriptionEndpoint = deleteEmailSubscriptionEndpoint
	notificationPermissions.AddSmsSubscriptionEndpoint = addSMSSubscriptionEndpoint
	notificationPermissions.DeleteSmsSubscriptionEndpoint = deleteSMSSubscriptionEndpoint

	page.Layout = layouts.Main
	page.Component = pages.NotificationPermissions(&page, notificationPermissions)
	page.Name = templates.PagePreferences
	return &page, nil
}

func authenticatedProfileID(ctx echo.Context) (int, error) {
	v := ctx.Get(customctx.AuthenticatedProfileIDKey)
	profileID, ok := v.(int)
	if !ok || profileID <= 0 {
		return 0, echo.NewHTTPError(http.StatusUnauthorized, "authenticated profile id missing from context")
	}
	return profileID, nil
}
