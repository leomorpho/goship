package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/leomorpho/goship/v2/framework/core"
	eventtypes "github.com/leomorpho/goship/v2/framework/events/types"
)

const AsyncJobName = "framework.events.publish"

type AsyncEnvelope struct {
	Type  string          `json:"type"`
	Event json.RawMessage `json:"event"`
}

var asyncEventDecoders = map[string]func([]byte) (any, error){
	mustEventTypeName(eventtypes.UserRegistered{}): func(payload []byte) (any, error) {
		var event eventtypes.UserRegistered
		return event, json.Unmarshal(payload, &event)
	},
	mustEventTypeName(eventtypes.UserLoggedIn{}): func(payload []byte) (any, error) {
		var event eventtypes.UserLoggedIn
		return event, json.Unmarshal(payload, &event)
	},
	mustEventTypeName(eventtypes.UserLoggedOut{}): func(payload []byte) (any, error) {
		var event eventtypes.UserLoggedOut
		return event, json.Unmarshal(payload, &event)
	},
	mustEventTypeName(eventtypes.PasswordChanged{}): func(payload []byte) (any, error) {
		var event eventtypes.PasswordChanged
		return event, json.Unmarshal(payload, &event)
	},
	mustEventTypeName(eventtypes.SubscriptionCreated{}): func(payload []byte) (any, error) {
		var event eventtypes.SubscriptionCreated
		return event, json.Unmarshal(payload, &event)
	},
	mustEventTypeName(eventtypes.SubscriptionCancelled{}): func(payload []byte) (any, error) {
		var event eventtypes.SubscriptionCancelled
		return event, json.Unmarshal(payload, &event)
	},
	mustEventTypeName(eventtypes.ProfileCompletedOnboarding{}): func(payload []byte) (any, error) {
		var event eventtypes.ProfileCompletedOnboarding
		return event, json.Unmarshal(payload, &event)
	},
	mustEventTypeName(eventtypes.ProfileUpdated{}): func(payload []byte) (any, error) {
		var event eventtypes.ProfileUpdated
		return event, json.Unmarshal(payload, &event)
	},
}

func PublishAsync(ctx context.Context, bus *Bus, jobs core.Jobs, event any) error {
	if bus == nil {
		return fmt.Errorf("event bus is nil")
	}
	if jobs == nil {
		return fmt.Errorf("jobs backend is nil")
	}

	typeName, err := eventTypeName(event)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	envelope, err := json.Marshal(AsyncEnvelope{
		Type:  typeName,
		Event: payload,
	})
	if err != nil {
		return err
	}

	_, err = jobs.Enqueue(ctx, AsyncJobName, envelope, core.EnqueueOptions{})
	return err
}

// DeliverAsync decodes the async bridge envelope and republishes the event into the local bus.
func DeliverAsync(ctx context.Context, bus *Bus, payload []byte) error {
	if bus == nil {
		return fmt.Errorf("event bus is nil")
	}

	var envelope AsyncEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return fmt.Errorf("decode async event envelope: %w", err)
	}
	event, err := decodeAsyncEvent(envelope)
	if err != nil {
		return err
	}
	return bus.Publish(ctx, event)
}

func decodeAsyncEvent(envelope AsyncEnvelope) (any, error) {
	if envelope.Type == "" {
		return nil, fmt.Errorf("async event type is required")
	}
	if len(envelope.Event) == 0 {
		return nil, fmt.Errorf("async event payload is required")
	}

	decoder, ok := asyncEventDecoders[envelope.Type]
	if !ok {
		return nil, fmt.Errorf("unsupported async event type %q", envelope.Type)
	}

	event, err := decoder(envelope.Event)
	if err != nil {
		return nil, fmt.Errorf("decode async event %q: %w", envelope.Type, err)
	}
	return event, nil
}

func mustEventTypeName(event any) string {
	name, err := eventTypeName(event)
	if err != nil {
		panic(err)
	}
	return name
}
