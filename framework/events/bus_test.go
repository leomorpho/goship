package events

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/leomorpho/goship/framework/core"
	"github.com/stretchr/testify/require"
)

type testEvent struct {
	ID string `json:"id"`
}

func TestPublishInvokesSubscribedHandler(t *testing.T) {
	bus := NewBus()
	var got testEvent

	Subscribe(bus, func(_ context.Context, event testEvent) error {
		got = event
		return nil
	})

	err := bus.Publish(context.Background(), testEvent{ID: "evt-1"})
	require.NoError(t, err)
	require.Equal(t, testEvent{ID: "evt-1"}, got)
}

func TestPublishAsyncEnqueuesEnvelope(t *testing.T) {
	bus := NewBus()
	jobs := &testJobs{}

	err := PublishAsync(context.Background(), bus, jobs, testEvent{ID: "evt-2"})
	require.NoError(t, err)
	require.Equal(t, AsyncJobName, jobs.name)

	var envelope AsyncEnvelope
	require.NoError(t, json.Unmarshal(jobs.payload, &envelope))
	require.Equal(t, "events.testEvent", envelope.Type)

	var payload testEvent
	require.NoError(t, json.Unmarshal(envelope.Event, &payload))
	require.Equal(t, testEvent{ID: "evt-2"}, payload)
}

type testJobs struct {
	name    string
	payload []byte
}

func (j *testJobs) Register(string, core.JobHandler) error { return nil }
func (j *testJobs) Enqueue(_ context.Context, name string, payload []byte, _ core.EnqueueOptions) (string, error) {
	j.name = name
	j.payload = append([]byte(nil), payload...)
	return "job-1", nil
}
func (j *testJobs) StartWorker(context.Context) error    { return nil }
func (j *testJobs) StartScheduler(context.Context) error { return nil }
func (j *testJobs) Stop(context.Context) error           { return nil }
func (j *testJobs) Capabilities() core.JobCapabilities   { return core.JobCapabilities{} }
