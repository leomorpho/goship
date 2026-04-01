package adapters

import (
	"context"
	"errors"
	"sync"

	"github.com/leomorpho/goship/v2/framework/core"
	pubsubrepo "github.com/leomorpho/goship/v2/framework/pubsub"
)

var _ core.PubSub = (*CorePubSubAdapter)(nil)

// CorePubSubAdapter adapts repo-level pubsub client to the core.PubSub interface.
type CorePubSubAdapter struct {
	client pubsubrepo.PubSubClient
}

type corePubSubSubscription struct {
	cancel context.CancelFunc
	once   sync.Once
}

func NewCorePubSubAdapter(client pubsubrepo.PubSubClient) *CorePubSubAdapter {
	return &CorePubSubAdapter{client: client}
}

func (a *CorePubSubAdapter) Publish(ctx context.Context, topic string, payload []byte) error {
	if a == nil || a.client == nil {
		return errors.New("pubsub client is not initialized")
	}
	return a.client.Publish(ctx, topic, pubsubrepo.SSEEvent{
		Type: "message",
		Data: string(payload),
	})
}

func (a *CorePubSubAdapter) Subscribe(ctx context.Context, topic string, handler core.MessageHandler) (core.Subscription, error) {
	if a == nil || a.client == nil {
		return nil, errors.New("pubsub client is not initialized")
	}
	if handler == nil {
		return nil, errors.New("message handler is required")
	}

	subCtx, cancel := context.WithCancel(ctx)
	ch, err := a.client.SSESubscribe(subCtx, topic)
	if err != nil {
		cancel()
		return nil, err
	}

	go func() {
		for {
			select {
			case <-subCtx.Done():
				return
			case evt, ok := <-ch:
				if !ok {
					return
				}
				if err := handler(subCtx, topic, []byte(evt.Data)); err != nil {
					cancel()
					return
				}
			}
		}
	}()

	return &corePubSubSubscription{cancel: cancel}, nil
}

func (a *CorePubSubAdapter) Close() error {
	// Underlying pubsub clients currently don't expose a common close lifecycle method.
	return nil
}

func (s *corePubSubSubscription) Close() error {
	if s == nil {
		return nil
	}
	s.once.Do(func() {
		if s.cancel != nil {
			s.cancel()
		}
	})
	return nil
}
