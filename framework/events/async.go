package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/leomorpho/goship/framework/core"
)

const AsyncJobName = "framework.events.publish"

type AsyncEnvelope struct {
	Type  string          `json:"type"`
	Event json.RawMessage `json:"event"`
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
