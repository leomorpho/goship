package controllers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/janberktold/sse"
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship-modules/notifications"
	"github.com/leomorpho/goship/app/web/ui"
	"log/slog"
)

type realtime struct {
	ctr      ui.Controller
	notifier notifications.NotifierService
}

func NewRealtimeRoute(
	ctr ui.Controller,
	notifier notifications.NotifierService,
) *realtime {
	return &realtime{
		ctr:      ctr,
		notifier: notifier,
	}
}

// realtime handles SSE connections to any client desiring real-time data.
func (c *realtime) Get(ctx echo.Context) error {
	profileID, err := authenticatedProfileID(ctx)
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

	// Send initial connection message
	if err := conn.WriteStringEvent("initial-connection", "Welcome!"); err != nil {
		slog.Error("Failed to send initial SSE message", "error", err)
	}

	subCtx, cancel := context.WithCancel(ctx.Request().Context())
	defer cancel()

	// SSESubscribe to a user's channel
	sseEventStream, err := c.notifier.SSESubscribe(subCtx, fmt.Sprint(profileID))
	if err != nil {
		slog.Error("Failed to subscribe to the channel", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to subscribe to the channel")
	}

	// Send periodic comments to keep the connection alive
	ticker := time.NewTicker(c.ctr.Container.Config.HTTP.SseKeepAlive)
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
		// The SSE endpoint is no longer listened to and its ressources can be discarded.
		case <-ctx.Request().Context().Done():
			slog.Info("SSE connection unsubscribed and cleaned up")
			return nil
		}
	}
}
