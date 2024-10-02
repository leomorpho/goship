package pubsub

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/rs/zerolog/log"
)

// GoroutinePubSubClient is a simple pub/sub client that uses goroutines to handle events internally.
type GoroutinePubSubClient struct {
	subscribers map[string][]chan SSEEvent
	mu          sync.RWMutex
}

// NewGoroutinePubSubClient initializes a new GoroutinePubSubClient
func NewGoroutinePubSubClient() *GoroutinePubSubClient {
	return &GoroutinePubSubClient{
		subscribers: make(map[string][]chan SSEEvent),
	}
}

// SSESubscribe subscribes to a topic and returns a channel to receive events
func (c *GoroutinePubSubClient) SSESubscribe(ctx context.Context, topic string) (<-chan SSEEvent, error) {
	eventCh := make(chan SSEEvent)

	c.mu.Lock()
	c.subscribers[topic] = append(c.subscribers[topic], eventCh)
	c.mu.Unlock()

	// Goroutine to manage context cancellation and cleanup
	go func() {
		<-ctx.Done()

		// Unsubscribe on context cancellation
		c.mu.Lock()
		subs := c.subscribers[topic]
		for i, subCh := range subs {
			if subCh == eventCh {
				c.subscribers[topic] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
		c.mu.Unlock()

		close(eventCh) // Close the channel when unsubscribed
	}()

	return eventCh, nil
}

// Publish publishes an event to all subscribers of a topic
func (c *GoroutinePubSubClient) Publish(_ context.Context, topic string, event SSEEvent) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Marshal the event to JSON (could be optional based on your use case)
	eventData, err := json.Marshal(event)
	if err != nil {
		return err
	}

	log.Info().Msgf("Publishing event: %s", eventData)

	// Send the event to all subscribers of the topic
	for _, subscriberCh := range c.subscribers[topic] {
		go func(ch chan SSEEvent) {
			select {
			case ch <- event:
			default:
				log.Warn().Msg("Subscriber channel is full, dropping event")
			}
		}(subscriberCh)
	}

	return nil
}
