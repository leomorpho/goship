package sse

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/a-h/templ"
	"github.com/leomorpho/goship/framework/core"
)

func TestHubPublishDeliversToAllSubscribers(t *testing.T) {
	hub := NewHub(nil)
	ch1, unsub1 := hub.Subscribe("post:42")
	defer unsub1()
	ch2, unsub2 := hub.Subscribe("post:42")
	defer unsub2()

	hub.Publish("post:42", "updated")

	assertReceive(t, ch1, "updated")
	assertReceive(t, ch2, "updated")
}

func TestHubUnsubscribeStopsDelivery(t *testing.T) {
	hub := NewHub(nil)
	ch, unsubscribe := hub.Subscribe("post:42")
	unsubscribe()

	hub.Publish("post:42", "updated")

	select {
	case msg := <-ch:
		t.Fatalf("received %q after unsubscribe", msg)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestHubPublishHTMLRendersComponent(t *testing.T) {
	hub := NewHub(nil)
	ch, unsubscribe := hub.Subscribe("post:42")
	defer unsubscribe()

	err := hub.PublishHTML("post:42", templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, "<div>hello</div>")
		return err
	}))
	if err != nil {
		t.Fatalf("PublishHTML() error = %v", err)
	}

	assertReceive(t, ch, "<div>hello</div>")
}

type mockPubSub struct {
	core.PubSub
	publishedTopic string
	publishedData  []byte
	handler        core.MessageHandler
}

func (m *mockPubSub) Publish(ctx context.Context, topic string, payload []byte) error {
	m.publishedTopic = topic
	m.publishedData = payload
	return nil
}

func (m *mockPubSub) Subscribe(ctx context.Context, topic string, handler core.MessageHandler) (core.Subscription, error) {
	m.handler = handler
	return &mockSubscription{}, nil
}

type mockSubscription struct{}

func (s *mockSubscription) Close() error { return nil }

func TestHubWithPubSub(t *testing.T) {
	ps := &mockPubSub{}
	hub := NewHub(ps)

	ch, unsub := hub.Subscribe("topic:1")
	defer unsub()

	if ps.handler == nil {
		t.Fatal("expected Hub to subscribe to pubsub")
	}

	// Test publishing via Hub
	hub.Publish("topic:1", "data:1")
	if ps.publishedTopic != "topic:1" {
		t.Errorf("published topic = %q, want topic:1", ps.publishedTopic)
	}
	if string(ps.publishedData) != "data:1" {
		t.Errorf("published data = %q, want data:1", string(ps.publishedData))
	}

	// Test receiving via pubsub
	go ps.handler(context.Background(), "topic:1", []byte("data:remote"))
	assertReceive(t, ch, "data:remote")
}

func assertReceive(t *testing.T, ch <-chan string, want string) {
	t.Helper()
	select {
	case got := <-ch:
		if got != want {
			t.Fatalf("received %q, want %q", got, want)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %q", want)
	}
}
