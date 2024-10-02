package pubsub

import (
	"context"
	"encoding/json"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
)

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
