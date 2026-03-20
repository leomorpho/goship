package foundation

import (
	"context"

	notifications "github.com/leomorpho/goship-modules/notifications"
	"github.com/leomorpho/goship/framework/core"
)

type notificationsPubSubAdapter struct {
	inner core.PubSub
}

type notificationsSubscriptionAdapter struct {
	inner core.Subscription
}

func (s notificationsSubscriptionAdapter) Close() error {
	return s.inner.Close()
}

// AdaptNotificationsPubSub converts the app pubsub dependency to the notifications module boundary.
func AdaptNotificationsPubSub(inner core.PubSub) notifications.PubSub {
	if inner == nil {
		return nil
	}
	return notificationsPubSubAdapter{inner: inner}
}

func (a notificationsPubSubAdapter) Publish(ctx context.Context, topic string, payload []byte) error {
	return a.inner.Publish(ctx, topic, payload)
}

func (a notificationsPubSubAdapter) Subscribe(
	ctx context.Context, topic string, handler notifications.MessageHandler,
) (notifications.PubSubSubscription, error) {
	sub, err := a.inner.Subscribe(ctx, topic, func(hctx context.Context, t string, payload []byte) error {
		return handler(hctx, t, payload)
	})
	if err != nil {
		return nil, err
	}
	return notificationsSubscriptionAdapter{inner: sub}, nil
}

func (a notificationsPubSubAdapter) Close() error {
	return a.inner.Close()
}
