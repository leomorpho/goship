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
	"github.com/leomorpho/goship/framework/core"
	"github.com/leomorpho/goship/framework/dberrors"
	"github.com/leomorpho/goship/framework/domain"
	"github.com/leomorpho/goship/framework/flash"
	frameworkauthcontext "github.com/leomorpho/goship/framework/http/authcontext"
	layouts "github.com/leomorpho/goship/framework/http/layouts/gen"
	pages "github.com/leomorpho/goship/framework/http/pages/gen"
	"github.com/leomorpho/goship/framework/http/ui"
)

const notificationQueryParam = "notif"

type notificationItem struct {
	ID                        int
	Title                     string
	Text                      string
	ButtonText                string
	Link                      string
	CreatedAt                 time.Time
	Read                      bool
	ReadInNotificationsCenter bool
}

type normalNotificationsPageData struct {
	Notifications []notificationItem
	NextPageURL   string
}

type notificationPermissionsData struct {
	PermissionDailyNotif          domain.NotificationPermission
	PermissionPartnerActivity     domain.NotificationPermission
	VapidPublicKey                string
	SubscribedEndpoints           []string
	PhoneSubscriptionEnabled      bool
	NotificationTypeQueryParamKey string

	AddPushSubscriptionEndpoint    string
	DeletePushSubscriptionEndpoint string

	AddFCMPushSubscriptionEndpoint    string
	DeleteFCMPushSubscriptionEndpoint string

	AddEmailSubscriptionEndpoint    string
	DeleteEmailSubscriptionEndpoint string

	AddSmsSubscriptionEndpoint    string
	DeleteSmsSubscriptionEndpoint string
}

type pushNotificationSubscriptions struct {
	URLs []string `json:"urls"`
}

func mapNotificationItems(items []*domain.Notification) []notificationItem {
	if len(items) == 0 {
		return []notificationItem{}
	}
	out := make([]notificationItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, notificationItem{
			ID:                        item.ID,
			Title:                     item.Title,
			Text:                      item.Text,
			ButtonText:                item.ButtonText,
			Link:                      item.Link,
			CreatedAt:                 item.CreatedAt,
			Read:                      item.Read,
			ReadInNotificationsCenter: item.ReadInNotificationsCenter,
		})
	}
	return out
}

type notificationCountReader interface {
	GetCountOfUnseenNotifications(ctx stdcontext.Context, profileID int) (int, error)
}

type RouteModuleDeps struct {
	Controller                    ui.Controller
	NotifierService               *notifications.NotifierService
	PwaPushService                *notifications.PwaPushService
	FcmPushService                *notifications.FcmPushService
	NotificationPermissionService *notifications.NotificationPermissionService
}

type RouteModule struct {
	controller                    ui.Controller
	notifierService               *notifications.NotifierService
	pwaPushService                *notifications.PwaPushService
	fcmPushService                *notifications.FcmPushService
	notificationPermissionService *notifications.NotificationPermissionService
}

func NewRouteModule(deps RouteModuleDeps) *RouteModule {
	return &RouteModule{
		controller:                    deps.Controller,
		notifierService:               deps.NotifierService,
		pwaPushService:                deps.PwaPushService,
		fcmPushService:                deps.FcmPushService,
		notificationPermissionService: deps.NotificationPermissionService,
	}
}

func (m *RouteModule) ID() string {
	return notifications.ModuleID
}

func (m *RouteModule) Migrations() fs.FS {
	return nil
}

func (m *RouteModule) RegisterRoutes(r core.Router) error {
	normalNotifications := NewNormalNotificationsRoute(m.controller, m.notifierService)
	r.GET("/notifications", normalNotifications.Get).Name = "notifications"
	r.GET("/notifications/mark-all-read", normalNotifications.MarkAllAsRead).Name = "normalNotificationsMarkAllAsRead"
	r.DELETE("/notifications/:notification_id", normalNotifications.Delete).Name = "notifications.delete"

	normalNotificationsCount := NewNormalNotificationsCountRoute(m.controller, m.notifierService)
	r.GET("/notifications/normalNotificationsCount", normalNotificationsCount.Get).Name = "normal_notifications_count"

	markRead := NewMarkNormalNotificationReadRoute(m.controller, m.notifierService)
	r.POST("/notifications/:notification_id/read", markRead.Post).Name = "markNormalNotificationRead"

	markUnread := NewMarkNormalNotificationUnreadRoute(m.controller, m.notifierService)
	r.POST("/notifications/unread", markUnread.Post).Name = "notifications.unread"

	return nil
}

func (m *RouteModule) RegisterOnboardingRoutes(r core.Router) error {
	outgoingNotifications := NewPushNotifsRoute(
		m.controller,
		m.pwaPushService,
		m.fcmPushService,
		m.notifierService,
		m.notificationPermissionService,
	)
	r.GET("/subscription/push", outgoingNotifications.GetPushSubscriptions).Name = "push_subscriptions.get"
	r.POST("/subscription/:platform", outgoingNotifications.RegisterSubscription).Name = "notification_subscriptions.register"
	r.DELETE("/subscription/:platform", outgoingNotifications.DeleteSubscription).Name = "notification_subscriptions.delete"
	r.GET("/email-subscription/unsubscribe/:permission/:token", outgoingNotifications.DeleteEmailSubscription).Name = "email_subscriptions.delete_with_token"
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
	page.Name = "notifications"
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
		nextPageURL = ctx.Echo().Reverse("notifications") + "?timestamp=" + oldestTimestamp.Format(time.RFC3339Nano)
	}

	data := normalNotificationsPageData{Notifications: []notificationItem{}}
	data.Notifications = mapNotificationItems(notifications)
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

	return r.controller.Redirect(ctx, "notifications")
}

type OutgoingNotificationsRoute struct {
	controller                    ui.Controller
	pwaPushService                *notifications.PwaPushService
	fcmPushService                *notifications.FcmPushService
	notifierService               notificationCountReader
	notificationPermissionService *notifications.NotificationPermissionService
}

func NewPushNotifsRoute(
	controller ui.Controller,
	pwaPushService *notifications.PwaPushService,
	fcmPushService *notifications.FcmPushService,
	notifierService notificationCountReader,
	notificationPermissionService *notifications.NotificationPermissionService,
) *OutgoingNotificationsRoute {
	return &OutgoingNotificationsRoute{
		controller:                    controller,
		pwaPushService:                pwaPushService,
		fcmPushService:                fcmPushService,
		notifierService:               notifierService,
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

	subscriptions := pushNotificationSubscriptions{URLs: []string{}}
	subscriptions.URLs = subscribedEndpoints
	return ctx.JSON(http.StatusOK, subscriptions)
}

func (r *OutgoingNotificationsRoute) RegisterSubscription(ctx echo.Context) error {
	platformStr := ctx.Param("platform")
	if platformStr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid profile ID")
	}

	platform := notifications.ParsePlatform(platformStr)
	notificationType := ctx.QueryParam(domain.PermissionNotificationType)
	var permission *notifications.PermissionType
	if notificationType != "" {
		permission = notifications.Permissions.Parse(notificationType)
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
		for _, perm := range notifications.Permissions.Members() {
			if err := r.notificationPermissionService.CreatePermission(ctx.Request().Context(), profileID, perm, platform); err != nil {
				slog.Error("failed to create notification permission", "error", err)
			}
		}
	}

	switch *platform {
	case notifications.PlatformPWAPush:
		hasPermissionsAlready, err := r.pwaPushService.HasEndpointRegistered(ctx.Request().Context(), profileID, req.Endpoint)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "error checking if user still has registered endpoints")
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

	case notifications.PlatformFCMPush:
		hasPermissionsAlready, err := r.fcmPushService.HasTokenRegistered(ctx.Request().Context(), profileID, req.FCMToken)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "error checking if user still has registered fcm subscriptions")
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

	platform := notifications.ParsePlatform(platformStr)
	notificationType := ctx.QueryParam(domain.PermissionNotificationType)
	var permission *notifications.PermissionType
	if notificationType != "" {
		permission = notifications.Permissions.Parse(notificationType)
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
		for _, perm := range notifications.Permissions.Members() {
			if err := r.notificationPermissionService.DeletePermission(ctx.Request().Context(), profileID, perm, platform, nil); err != nil {
				slog.Error("failed to delete notification permission", "error", err)
			}
		}
	}

	switch *platform {
	case notifications.PlatformPWAPush:
		hasPermissionsLeft, err := r.pwaPushService.HasEndpointRegistered(ctx.Request().Context(), profileID, req.Endpoint)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "error checking if user still has registered pwa endpoints")
		}
		if !hasPermissionsLeft {
			if err := r.pwaPushService.DeletePushSubscriptionByEndpoint(ctx.Request().Context(), profileID, req.Endpoint); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete pwa push subscription")
			}
		}

	case notifications.PlatformFCMPush:
		hasPermissionsLeft, err := r.fcmPushService.HasTokenRegistered(ctx.Request().Context(), profileID, req.FCMToken)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "error checking if user still has registered fcm registration")
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

	permission := notifications.Permissions.Parse(notificationType)
	profileID, err := authenticatedProfileID(ctx)
	if err != nil {
		return err
	}

	var notificationName string
	switch *permission {
	case notifications.PermissionDailyReminder:
		notificationName = "daily updates"
	case notifications.PermissionNewFriendActivity:
		notificationName = "partner activity"
	default:
		slog.Error("no notification exists with that name", "notifPermission", permission.Value)
	}

	permissionErr := r.notificationPermissionService.DeletePermission(
		ctx.Request().Context(),
		profileID,
		*permission,
		&notifications.PlatformEmail,
		&token,
	)

	page, err := r.createNotificationsPage(ctx, profileID, "", false)
	if err != nil {
		return err
	}

	if permissionErr != nil {
		slog.Error("failed to delete email notification permission",
			"error", permissionErr,
			"profileID", profileID,
			"notifPermission", permission.Value,
			"platform", notifications.PlatformEmail.Value,
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
		ctx.Echo().Reverse("notification_subscriptions.register", notifications.PlatformPWAPush.Value),
	) + "?csrf=" + page.CSRF
	deletePushSubscriptionEndpoint := fmt.Sprintf(
		"%s%s",
		r.controller.Container.Config.HTTP.Domain,
		ctx.Echo().Reverse("notification_subscriptions.delete", notifications.PlatformPWAPush.Value),
	) + "?csrf=" + page.CSRF

	addEmailSubscriptionEndpoint := fmt.Sprintf(
		"%s%s",
		r.controller.Container.Config.HTTP.Domain,
		ctx.Echo().Reverse("notification_subscriptions.register", notifications.PlatformEmail.Value),
	) + "?csrf=" + page.CSRF
	deleteEmailSubscriptionEndpoint := fmt.Sprintf(
		"%s%s",
		r.controller.Container.Config.HTTP.Domain,
		ctx.Echo().Reverse("notification_subscriptions.delete", notifications.PlatformEmail.Value),
	) + "?csrf=" + page.CSRF

	addSMSSubscriptionEndpoint := fmt.Sprintf(
		"%s%s",
		r.controller.Container.Config.HTTP.Domain,
		ctx.Echo().Reverse("notification_subscriptions.register", notifications.PlatformSMS.Value),
	) + "?csrf=" + page.CSRF
	deleteSMSSubscriptionEndpoint := fmt.Sprintf(
		"%s%s",
		r.controller.Container.Config.HTTP.Domain,
		ctx.Echo().Reverse("notification_subscriptions.delete", notifications.PlatformSMS.Value),
	) + "?csrf=" + page.CSRF

	notificationPermissions := notificationPermissionsData{SubscribedEndpoints: []string{}}
	notificationPermissions.VapidPublicKey = r.controller.Container.Config.App.VapidPublicKey
	notificationPermissions.PermissionDailyNotif = permissions[notifications.PermissionDailyReminder]
	notificationPermissions.PermissionPartnerActivity = permissions[notifications.PermissionNewFriendActivity]
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
	page.Name = "preferences"
	return &page, nil
}

func authenticatedProfileID(ctx echo.Context) (int, error) {
	return frameworkauthcontext.AuthenticatedProfileID(ctx)
}
