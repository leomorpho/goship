package sse

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/a-h/templ"
)

func TestHubPublishDeliversToAllSubscribers(t *testing.T) {
	hub := NewHub()
	ch1, unsub1 := hub.Subscribe("post:42")
	defer unsub1()
	ch2, unsub2 := hub.Subscribe("post:42")
	defer unsub2()

	hub.Publish("post:42", "updated")

	assertReceive(t, ch1, "updated")
	assertReceive(t, ch2, "updated")
}

func TestHubUnsubscribeStopsDelivery(t *testing.T) {
	hub := NewHub()
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
	hub := NewHub()
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
