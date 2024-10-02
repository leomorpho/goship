package pubsub

import (
	"context"
)

type SSEEvent struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

// MessageHandler is a function type for handling incoming messages
type MessageHandler func(topic string, event SSEEvent)

// PubSubClient defines the interface for a pub/sub system client
type PubSubClient interface {
	SSESubscribe(ctx context.Context, topic string) (<-chan SSEEvent, error)
	Publish(ctx context.Context, topic string, event SSEEvent) error
}
