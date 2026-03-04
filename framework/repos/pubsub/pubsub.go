package pubsub

import (
	"context"
	"encoding/json"

	"github.com/go-redis/redis/v8"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
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

type NATSPubSubClient struct {
	conn *nats.Conn
}

func NewNATSPubSubClient(conn *nats.Conn) *NATSPubSubClient {
	return &NATSPubSubClient{conn: conn}
}

func (c *NATSPubSubClient) SSESubscribe(ctx context.Context, topic string) (<-chan SSEEvent, error) {
	eventCh := make(chan SSEEvent)

	sub, err := c.conn.Subscribe(topic, func(m *nats.Msg) {
		var event SSEEvent
		if err := json.Unmarshal(m.Data, &event); err != nil {
			log.Error().Err(err).Msg("Error unmarshalling message")
			return
		}
		select {
		case eventCh <- event:
		case <-ctx.Done():
			return
		}
	})
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		sub.Unsubscribe()
		close(eventCh)
	}()

	return eventCh, nil
}

func (c *NATSPubSubClient) Publish(_ context.Context, topic string, event SSEEvent) error {
	eventData, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return c.conn.Publish(topic, eventData)
}

type RedisPubSubClient struct {
	client *redis.Client
}

func NewRedisPubSubClient(client *redis.Client) *RedisPubSubClient {
	return &RedisPubSubClient{
		client: client,
	}
}

func (c *RedisPubSubClient) SSESubscribe(ctx context.Context, subject string) (<-chan SSEEvent, error) {
	pubsub := c.client.Subscribe(ctx, subject)
	eventCh := make(chan SSEEvent)

	// Start a new goroutine to listen for messages
	go func() {
		defer pubsub.Close()
		ch := pubsub.Channel()
		for {
			select {
			case msg := <-ch:
				var event SSEEvent
				if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
					log.Error().Err(err).Msg("Error unmarshalling message")
					continue // or consider closing the channel and returning on a fatal error
				}
				select {
				case eventCh <- event:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return eventCh, nil
}

func (c *RedisPubSubClient) Publish(ctx context.Context, subject string, event SSEEvent) error {
	eventData, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return c.client.Publish(ctx, subject, eventData).Err()
}
