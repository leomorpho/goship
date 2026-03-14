package pubsub

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/go-redis/redis/v8"
	"github.com/nats-io/nats.go"
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
			slog.Error("Error unmarshalling message", "error", err)
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
					slog.Error("Error unmarshalling message", "error", err)
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

type InProcPubSubClient struct {
	mu     sync.RWMutex
	topics map[string]map[chan SSEEvent]struct{}
}

func NewInProcPubSubClient() *InProcPubSubClient {
	return &InProcPubSubClient{
		topics: make(map[string]map[chan SSEEvent]struct{}),
	}
}

func (c *InProcPubSubClient) SSESubscribe(ctx context.Context, topic string) (<-chan SSEEvent, error) {
	ch := make(chan SSEEvent, 8)
	c.mu.Lock()
	if c.topics[topic] == nil {
		c.topics[topic] = make(map[chan SSEEvent]struct{})
	}
	c.topics[topic][ch] = struct{}{}
	c.mu.Unlock()

	go func() {
		<-ctx.Done()
		c.mu.Lock()
		delete(c.topics[topic], ch)
		if len(c.topics[topic]) == 0 {
			delete(c.topics, topic)
		}
		c.mu.Unlock()
		close(ch)
	}()

	return ch, nil
}

func (c *InProcPubSubClient) Publish(ctx context.Context, topic string, event SSEEvent) error {
	c.mu.RLock()
	subs := make([]chan SSEEvent, 0, len(c.topics[topic]))
	for ch := range c.topics[topic] {
		subs = append(subs, ch)
	}
	c.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- event:
		default:
		}
	}
	return nil
}
