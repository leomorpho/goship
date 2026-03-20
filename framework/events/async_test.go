package events

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	eventtypes "github.com/leomorpho/goship/framework/events/types"
	"github.com/stretchr/testify/require"
)

func TestDeliverAsyncPublishesEnvelope(t *testing.T) {
	t.Parallel()

	bus := NewBus()
	received := make(chan eventtypes.UserLoggedIn, 1)
	Subscribe(bus, func(_ context.Context, event eventtypes.UserLoggedIn) error {
		received <- event
		return nil
	})

	event := eventtypes.UserLoggedIn{UserID: 7, IP: "203.0.113.8", At: time.Unix(1700000000, 0).UTC()}
	payload, err := json.Marshal(AsyncEnvelope{
		Type:  mustEventTypeName(event),
		Event: mustJSON(t, event),
	})
	require.NoError(t, err)

	require.NoError(t, DeliverAsync(context.Background(), bus, payload))

	select {
	case got := <-received:
		require.Equal(t, event, got)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for delivered event")
	}
}

func TestDeliverAsyncRejectsUnsupportedEventType(t *testing.T) {
	t.Parallel()

	bus := NewBus()
	err := DeliverAsync(context.Background(), bus, mustJSON(t, AsyncEnvelope{
		Type:  "events.UnknownEvent",
		Event: json.RawMessage(`{"id":1}`),
	}))
	require.ErrorContains(t, err, "unsupported async event type")
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()

	payload, err := json.Marshal(v)
	require.NoError(t, err)
	return payload
}
