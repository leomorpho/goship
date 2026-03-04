package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/leomorpho/goship/app/notifications"
	"github.com/leomorpho/goship/app/profiles"
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
		ctr         ui.Controller
		profileRepo profiles.ProfileRepo
	}

	normalNotifications struct {
		ctr          ui.Controller
		notifierRepo *notifications.NotifierRepo
	}
)

func NewNormalNotificationsCountRoute(
	ctr ui.Controller,
	profileRepo profiles.ProfileRepo,
) *normalNotificationsCount {
	return &normalNotificationsCount{
		ctr:         ctr,
		profileRepo: profileRepo,
	}
}

func (c *normalNotificationsCount) Get(ctx echo.Context) error {
	usr := ctx.Get(context.AuthenticatedUserKey).(*ent.User)
	profile := usr.QueryProfile().FirstX(ctx.Request().Context())

	num, err := c.profileRepo.GetCountOfUnseenNotifications(ctx.Request().Context(), profile.ID)
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
	notifierRepo *notifications.NotifierRepo,
) *normalNotifications {
	return &normalNotifications{
		ctr:          ctr,
		notifierRepo: notifierRepo,
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

	notifications, err := n.notifierRepo.GetNotifications(ctx.Request().Context(), profile.ID, false, timestamp, &n.ctr.Container.Config.App.PageSize)
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

	err := n.notifierRepo.MarkAllNotificationRead(ctx.Request().Context(), profile.ID)
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

	err = n.notifierRepo.DeleteNotification(ctx.Request().Context(), notificationID, &profile.ID)
	if err != nil {
		return err
	}

	return ctx.String(http.StatusOK, "")
}

type markNormalNotificationRead struct {
	ctr          ui.Controller
	notifierRepo *notifications.NotifierRepo
}

func NewMarkNormalNotificationReadRoute(
	ctr ui.Controller, notifierRepo *notifications.NotifierRepo,
) *markNormalNotificationRead {
	return &markNormalNotificationRead{
		ctr:          ctr,
		notifierRepo: notifierRepo,
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

	err = c.notifierRepo.MarkNotificationRead(ctx.Request().Context(), notifID, &profileId)
	if err != nil {
		return err
	}

	return ctx.String(http.StatusOK, "")
}

type markNormalNotificationUnread struct {
	ctr          ui.Controller
	notifierRepo *notifications.NotifierRepo
}

func NewMarkNormalNotificationUnreadRoute(
	ctr ui.Controller, notifierRepo *notifications.NotifierRepo,
) *markNormalNotificationUnread {
	return &markNormalNotificationUnread{
		ctr:          ctr,
		notifierRepo: notifierRepo,
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

	err := c.notifierRepo.MarkNotificationUnread(ctx.Request().Context(), req.ID, &profileId)
	if err != nil {
		return err
	}

	return c.ctr.Redirect(ctx, "normalNotifications")
}
