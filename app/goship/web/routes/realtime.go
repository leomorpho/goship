package routes

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/janberktold/sse"
	"github.com/labstack/echo/v4"
	"github.com/mikestefanello/pagoda/ent"
	customContext "github.com/mikestefanello/pagoda/pkg/context"
	"github.com/mikestefanello/pagoda/pkg/controller"
	"github.com/mikestefanello/pagoda/pkg/repos/notifierrepo"
	"github.com/mikestefanello/pagoda/pkg/repos/pubsub"
	"github.com/rs/zerolog/log"
)

type realtime struct {
	ctr      controller.Controller
	notifier notifierrepo.NotifierRepo
}

func NewRealtimeRoute(
	ctr controller.Controller,
	notifier notifierrepo.NotifierRepo,
) *realtime {
	return &realtime{
		ctr:      ctr,
		notifier: notifier,
	}
}

// realtime handles SSE connections to any client desiring real-time data.
func (c *realtime) Get(ctx echo.Context) error {
	usr := ctx.Get(customContext.AuthenticatedUserKey).(*ent.User)
	profileID := usr.QueryProfile().FirstX(ctx.Request().Context()).ID

	w := ctx.Response().Writer
	r := ctx.Request()
	conn, err := sse.Upgrade(w, r)
	if err != nil {
		log.Error().Err(err).Msg("Failed to upgrade to SSE connection")
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to upgrade to SSE connection")
	}

	// Send initial connection message
	if err := conn.WriteStringEvent("initial-connection", "Welcome!"); err != nil {
		log.Error().Err(err).Msg("Failed to send initial SSE message")
	}

	messageChan := make(chan pubsub.SSEEvent)

	subCtx, cancel := context.WithCancel(ctx.Request().Context())
	defer cancel()

	// SSESubscribe to a user's channel
	sseEventStream, err := c.notifier.SSESubscribe(subCtx, fmt.Sprint(profileID))
	if err != nil {
		log.Error().Err(err).Msg("Failed to subscribe to the channel")
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to subscribe to the channel")
	}

	// Send periodic comments to keep the connection alive
	ticker := time.NewTicker(c.ctr.Container.Config.HTTP.SseKeepAlive)
	defer ticker.Stop()

	for {
		select {
		case event := <-sseEventStream:
			if err := conn.WriteStringEvent(event.Type, event.Data); err != nil {
				log.Error().Err(err).Msg("Failed to send SSE message")
			} else {
				log.Info().Str("event", event.Type).Msg("Sent SSE message")
			}
		case <-ticker.C:
			if err := conn.WriteStringEvent(": keep-alive", ""); err != nil {
				log.Error().Err(err).Msg("Failed to send keep-alive SSE message")
			}
		// The SSE endpoint is no longer listened to and its ressources can be discarded.
		case <-ctx.Request().Context().Done():
			close(messageChan)
			log.Info().Msg("SSE connection unsubscribed and cleaned up")
			return nil
		}
	}
}
