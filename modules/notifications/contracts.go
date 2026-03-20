package notifications

import "context"

// MessageHandler processes pubsub payloads.
type MessageHandler func(ctx context.Context, topic string, payload []byte) error

// PubSubSubscription represents an active pubsub subscription.
type PubSubSubscription interface {
	Close() error
}

// PubSub is the module-local realtime boundary.
type PubSub interface {
	Publish(ctx context.Context, topic string, payload []byte) error
	Subscribe(ctx context.Context, topic string, handler MessageHandler) (PubSubSubscription, error)
	Close() error
}
