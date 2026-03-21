package controllers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/janberktold/sse"
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship-modules/notifications"
	frameworkauthcontext "github.com/leomorpho/goship/framework/web/authcontext"
	"github.com/leomorpho/goship/framework/web/ui"
	"log/slog"
)

type RealtimeRoute struct {
	Controller ui.Controller
	notifier   notifications.NotifierService
}

func NewRealtimeRoute(
	ctr ui.Controller,
	notifier notifications.NotifierService,
) *RealtimeRoute {
	return &RealtimeRoute{
		Controller: ctr,
		notifier:   notifier,
	}
}

func (c *RealtimeRoute) Get(ctx echo.Context) error {
	profileID, err := frameworkauthcontext.AuthenticatedProfileID(ctx)
	if err != nil {
		return err
	}

	w := ctx.Response().Writer
	r := ctx.Request()
	conn, err := sse.Upgrade(w, r)
	if err != nil {
		slog.Error("Failed to upgrade to SSE connection", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to upgrade to SSE connection")
	}

	if err := conn.WriteStringEvent("initial-connection", "Welcome!"); err != nil {
		slog.Error("Failed to send initial SSE message", "error", err)
	}

	subCtx, cancel := context.WithCancel(ctx.Request().Context())
	defer cancel()

	sseEventStream, err := c.notifier.SSESubscribe(subCtx, fmt.Sprint(profileID))
	if err != nil {
		slog.Error("Failed to subscribe to the channel", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to subscribe to the channel")
	}

	ticker := time.NewTicker(c.Controller.Container.Config.HTTP.SseKeepAlive)
	defer ticker.Stop()

	for {
		select {
		case event := <-sseEventStream:
			if err := conn.WriteStringEvent(event.Type, event.Data); err != nil {
				slog.Error("Failed to send SSE message", "error", err)
			} else {
				slog.Info("Sent SSE message", "event", event.Type)
			}
		case <-ticker.C:
			if err := conn.WriteStringEvent(": keep-alive", ""); err != nil {
				slog.Error("Failed to send keep-alive SSE message", "error", err)
			}
		case <-ctx.Request().Context().Done():
			slog.Info("SSE connection unsubscribed and cleaned up")
			return nil
		}
	}
}
