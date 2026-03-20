package schedules

import (
	"context"
	"testing"

	"github.com/robfig/cron/v3"

	"github.com/leomorpho/goship/framework/core"
)

type fakeJobs struct {
	names []string
}

func (f *fakeJobs) Register(string, core.JobHandler) error { return nil }
func (f *fakeJobs) StartWorker(context.Context) error      { return nil }
func (f *fakeJobs) StartScheduler(context.Context) error   { return nil }
func (f *fakeJobs) Stop(context.Context) error             { return nil }
func (f *fakeJobs) Capabilities() core.JobCapabilities     { return core.JobCapabilities{} }

func (f *fakeJobs) Enqueue(_ context.Context, name string, _ []byte, _ core.EnqueueOptions) (string, error) {
	f.names = append(f.names, name)
	return "job-id", nil
}

func TestRegisterAddsDefaultSchedulesThatEnqueueJobs(t *testing.T) {
	t.Parallel()

	s := cron.New(cron.WithSeconds())
	j := &fakeJobs{}
	Register(s, func() core.Jobs { return j })

	entries := s.Entries()
	if len(entries) < 2 {
		t.Fatalf("entries len = %d, want at least 2", len(entries))
	}

	for _, entry := range entries {
		entry.Job.Run()
	}

	if len(j.names) < 2 {
		t.Fatalf("enqueued jobs = %d, want at least 2", len(j.names))
	}
}

func TestRegisterNilSafe(t *testing.T) {
	t.Parallel()

	Register(nil, nil)
}
