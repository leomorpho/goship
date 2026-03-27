package adapters

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/leomorpho/goship/framework/core"
	pubsubrepo "github.com/leomorpho/goship/framework/pubsub"
)

type testPubSubClient struct {
	publishedTopic string
	publishedEvent pubsubrepo.SSEEvent
	ch             chan pubsubrepo.SSEEvent
}

func (c *testPubSubClient) SSESubscribe(context.Context, string) (<-chan pubsubrepo.SSEEvent, error) {
	if c.ch == nil {
		return nil, errors.New("missing channel")
	}
	return c.ch, nil
}

func (c *testPubSubClient) Publish(_ context.Context, topic string, event pubsubrepo.SSEEvent) error {
	c.publishedTopic = topic
	c.publishedEvent = event
	return nil
}

func TestCorePubSubAdapterNilClient(t *testing.T) {
	t.Parallel()

	adapter := NewCorePubSubAdapter(nil)
	if err := adapter.Publish(context.Background(), "topic", []byte("x")); err == nil {
		t.Fatal("expected publish error with nil client")
	}
	if _, err := adapter.Subscribe(context.Background(), "topic", func(context.Context, string, []byte) error { return nil }); err == nil {
		t.Fatal("expected subscribe error with nil client")
	}
}

func TestCorePubSubAdapterPublishAndSubscribe(t *testing.T) {
	t.Parallel()

	client := &testPubSubClient{ch: make(chan pubsubrepo.SSEEvent, 1)}
	adapter := NewCorePubSubAdapter(client)

	if err := adapter.Publish(context.Background(), "topic-a", []byte("hello")); err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	if client.publishedTopic != "topic-a" {
		t.Fatalf("unexpected published topic: %q", client.publishedTopic)
	}
	if client.publishedEvent.Data != "hello" {
		t.Fatalf("unexpected published payload: %q", client.publishedEvent.Data)
	}

	got := make(chan string, 1)
	sub, err := adapter.Subscribe(context.Background(), "topic-b", func(_ context.Context, _ string, payload []byte) error {
		got <- string(payload)
		return nil
	})
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}
	defer func() { _ = sub.Close() }()

	client.ch <- pubsubrepo.SSEEvent{Type: "message", Data: "from-event"}

	select {
	case msg := <-got:
		if msg != "from-event" {
			t.Fatalf("unexpected received message: %q", msg)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for pubsub message")
	}
}

func TestCorePubSubAdapterSubscribeValidation(t *testing.T) {
	t.Parallel()

	client := &testPubSubClient{ch: make(chan pubsubrepo.SSEEvent, 1)}
	adapter := NewCorePubSubAdapter(client)
	_, err := adapter.Subscribe(context.Background(), "topic", nil)
	if err == nil {
		t.Fatal("expected validation error for nil handler")
	}
}

var _ core.PubSub = (*CorePubSubAdapter)(nil)
