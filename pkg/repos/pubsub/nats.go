package pubsub

import (
	"context"
	"encoding/json"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

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
