package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/leomorpho/goship/framework/core"
	"github.com/leomorpho/goship/framework/domain"
)

type fakePlannedNotificationRepo struct {
	createErr error
	ids       []int
	idsErr    error
}

func (f *fakePlannedNotificationRepo) CreateNotificationTimeObjects(context.Context, domain.NotificationType, domain.NotificationPermissionType) error {
	return f.createErr
}

func (f *fakePlannedNotificationRepo) ProfileIDsCanGetPlannedNotificationNow(context.Context, time.Time, domain.NotificationType, *[]int) ([]int, error) {
	return f.ids, f.idsErr
}

type enqueuedJob struct {
	name    string
	payload []byte
	opts    core.EnqueueOptions
}

type fakeJobs struct {
	enqueued []enqueuedJob
	err      error
}

func (f *fakeJobs) Register(string, core.JobHandler) error { return nil }
func (f *fakeJobs) StartWorker(context.Context) error      { return nil }
func (f *fakeJobs) StartScheduler(context.Context) error   { return nil }
func (f *fakeJobs) Stop(context.Context) error             { return nil }
func (f *fakeJobs) Capabilities() core.JobCapabilities     { return core.JobCapabilities{} }

func (f *fakeJobs) Enqueue(_ context.Context, name string, payload []byte, opts core.EnqueueOptions) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	f.enqueued = append(f.enqueued, enqueuedJob{name: name, payload: payload, opts: opts})
	return "job-id", nil
}

func TestAllDailyConvoNotificationsProcessor_ProcessTask(t *testing.T) {
	t.Parallel()

	t.Run("returns create-time error", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("create failed")
		p := NewAllDailyConvoNotificationsProcessor(
			&fakePlannedNotificationRepo{createErr: wantErr},
			&fakeJobs{},
			30,
		)
		if err := p.ProcessTask(context.Background(), nil); !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})

	t.Run("returns lookup error", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("lookup failed")
		p := NewAllDailyConvoNotificationsProcessor(
			&fakePlannedNotificationRepo{idsErr: wantErr},
			&fakeJobs{},
			30,
		)
		if err := p.ProcessTask(context.Background(), nil); !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	})

	t.Run("batches profile IDs and enqueues with expected options", func(t *testing.T) {
		t.Parallel()

		ids := make([]int, 120)
		for i := range ids {
			ids[i] = i + 1
		}
		j := &fakeJobs{}
		p := NewAllDailyConvoNotificationsProcessor(
			&fakePlannedNotificationRepo{ids: ids},
			j,
			30,
		)

		if err := p.ProcessTask(context.Background(), nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(j.enqueued) != 3 {
			t.Fatalf("expected 3 enqueued jobs, got %d", len(j.enqueued))
		}

		expectedBatchSizes := []int{50, 50, 20}
		for i, job := range j.enqueued {
			if job.name != TypeDailyConvoNotification {
				t.Fatalf("job[%d] unexpected type: %s", i, job.name)
			}
			if job.opts.Timeout != 120*time.Second {
				t.Fatalf("job[%d] unexpected timeout: %v", i, job.opts.Timeout)
			}
			if job.opts.Retention != 24*time.Hour {
				t.Fatalf("job[%d] unexpected retention: %v", i, job.opts.Retention)
			}
			var payload DailyConvoNotificationsPayload
			if err := json.Unmarshal(job.payload, &payload); err != nil {
				t.Fatalf("job[%d] invalid payload json: %v", i, err)
			}
			if len(payload.ProfileIDs) != expectedBatchSizes[i] {
				t.Fatalf("job[%d] unexpected batch size: got=%d want=%d", i, len(payload.ProfileIDs), expectedBatchSizes[i])
			}
		}
	})
}
