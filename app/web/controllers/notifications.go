package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/leomorpho/goship-modules/notifications"
	profilesvc "github.com/leomorpho/goship/app/profile"
	"github.com/leomorpho/goship/app/views"
	"github.com/leomorpho/goship/app/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/views/web/pages/gen"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/app/web/viewmodels"
	"github.com/leomorpho/goship/db/ent"
	"github.com/leomorpho/goship/framework/context"
	"github.com/leomorpho/goship/framework/domain"
	"github.com/rs/zerolog/log"

	"github.com/labstack/echo/v4"
)

const NOTIFICATION_QUERY_PARAM = "notif"

type (
	normalNotificationsCount struct {
		ctr            ui.Controller
		profileService profilesvc.ProfileService
	}

	normalNotifications struct {
		ctr             ui.Controller
		notifierService *notifications.NotifierService
	}
)

func NewNormalNotificationsCountRoute(
	ctr ui.Controller,
	profileService profilesvc.ProfileService,
) *normalNotificationsCount {
	return &normalNotificationsCount{
		ctr:            ctr,
		profileService: profileService,
	}
}

func (c *normalNotificationsCount) Get(ctx echo.Context) error {
	usr := ctx.Get(context.AuthenticatedUserKey).(*ent.User)
	profile := usr.QueryProfile().FirstX(ctx.Request().Context())

	num, err := c.profileService.GetCountOfUnseenNotifications(ctx.Request().Context(), profile.ID)
	if err != nil {
		return err
	}
	var htmlResponse string
	if num == 0 {
		htmlResponse = fmt.Sprintf("<span class='hidden'>%d</span>", num)
	} else {
		htmlResponse = fmt.Sprintf("<span>%d</span>", num)

	}

	return ctx.String(http.StatusOK, htmlResponse)
}

func NewNormalNotificationsRoute(
	ctr ui.Controller,
	notifierService *notifications.NotifierService,
) *normalNotifications {
	return &normalNotifications{
		ctr:             ctr,
		notifierService: notifierService,
	}
}

func (n *normalNotifications) Get(ctx echo.Context) error {
	timestampParam := ctx.QueryParam("timestamp")
	var timestamp *time.Time
	if timestampParam != "" {
		parsedTime, err := time.Parse(time.RFC3339Nano, timestampParam)
		if err != nil {
			log.Error().Str("convo", "invalid timestamp format")
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid timestamp format")
		}
		timestamp = &parsedTime
	}

	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Name = templates.PageNotifications
	page.Component = pages.NotificationsPage(&page)
	// page.Title = "Your Notifications"
	page.HTMX.Request.Boosted = true
	page.ShowBottomNavbar = true
	page.SelectedBottomNavbarItem = domain.BottomNavbarItemNotifications

	usr := ctx.Get(context.AuthenticatedUserKey).(*ent.User)
	profile := usr.QueryProfile().FirstX(ctx.Request().Context())

	notifications, err := n.notifierService.GetNotifications(ctx.Request().Context(), profile.ID, false, timestamp, &n.ctr.Container.Config.App.PageSize)
	if err != nil {
		return err
	}

	for _, notif := range notifications {
		if notif == nil {
			continue
		}
		buttonText, ok := domain.NotificationCenterButtonText[notif.Type]
		if ok {
			notif.ButtonText = buttonText
		} else {
			notif.ButtonText = "See more"
		}
	}

	if len(notifications) == 0 && timestamp != nil {
		return nil
	}

	var nextPageURL string
	if len(notifications) > 0 {
		oldestNotifTimestamp := notifications[len(notifications)-1].CreatedAt
		nextPageURL = ctx.Echo().Reverse("normalNotifications") + "?timestamp=" + oldestNotifTimestamp.Format(time.RFC3339Nano)
	}

	page.Data = viewmodels.NormalNotificationsPageData{
		Notifications: notifications,
		NextPageURL:   nextPageURL,
	}

	return n.ctr.RenderPage(ctx, page)
}

func (n *normalNotifications) MarkAllAsRead(ctx echo.Context) error {

	usr := ctx.Get(context.AuthenticatedUserKey).(*ent.User)
	profile := usr.QueryProfile().FirstX(ctx.Request().Context())

	err := n.notifierService.MarkAllNotificationRead(ctx.Request().Context(), profile.ID)
	if err != nil {
		return err
	}

	return n.Get(ctx)
}

func (n *normalNotifications) Delete(ctx echo.Context) error {
	notificationIDStr := ctx.Param("notification_id")
	notificationID, err := strconv.Atoi(notificationIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid question ID")
	}

	usr := ctx.Get(context.AuthenticatedUserKey).(*ent.User)
	profile := usr.QueryProfile().FirstX(ctx.Request().Context())

	err = n.notifierService.DeleteNotification(ctx.Request().Context(), notificationID, &profile.ID)
	if err != nil {
		return err
	}

	return ctx.String(http.StatusOK, "")
}

type markNormalNotificationRead struct {
	ctr             ui.Controller
	notifierService *notifications.NotifierService
}

func NewMarkNormalNotificationReadRoute(
	ctr ui.Controller, notifierService *notifications.NotifierService,
) *markNormalNotificationRead {
	return &markNormalNotificationRead{
		ctr:             ctr,
		notifierService: notifierService,
	}
}

func (c *markNormalNotificationRead) Post(ctx echo.Context) error {
	notifIDStr := ctx.Param("notification_id")

	notifID, err := strconv.Atoi(notifIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid notification ID")
	}

	usr := ctx.Get(context.AuthenticatedUserKey).(*ent.User)
	profileId := usr.QueryProfile().
		FirstX(ctx.Request().Context()).ID

	err = c.notifierService.MarkNotificationRead(ctx.Request().Context(), notifID, &profileId)
	if err != nil {
		return err
	}

	return ctx.String(http.StatusOK, "")
}

type markNormalNotificationUnread struct {
	ctr             ui.Controller
	notifierService *notifications.NotifierService
}

func NewMarkNormalNotificationUnreadRoute(
	ctr ui.Controller, notifierService *notifications.NotifierService,
) *markNormalNotificationUnread {
	return &markNormalNotificationUnread{
		ctr:             ctr,
		notifierService: notifierService,
	}
}

type SeenEventRequest struct {
	ID int `form:"id" validate:"required"`
}

func (c *markNormalNotificationUnread) Post(ctx echo.Context) error {
	var req SeenEventRequest

	// Bind the request body to the struct
	if err := ctx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request")
	}

	usr := ctx.Get(context.AuthenticatedUserKey).(*ent.User)
	profileId := usr.QueryProfile().
		FirstX(ctx.Request().Context()).ID

	err := c.notifierService.MarkNotificationUnread(ctx.Request().Context(), req.ID, &profileId)
	if err != nil {
		return err
	}

	return c.ctr.Redirect(ctx, "normalNotifications")
}
