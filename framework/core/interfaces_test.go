package core

import (
	"context"
	"testing"
	"time"
)

func TestJobCapabilitiesMissing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		required  JobCapabilities
		available JobCapabilities
		want      []string
	}{
		{
			name:      "no requirements returns empty",
			required:  JobCapabilities{},
			available: JobCapabilities{},
			want:      nil,
		},
		{
			name: "all required capabilities available",
			required: JobCapabilities{
				Delayed:    true,
				Retries:    true,
				Cron:       true,
				Priority:   true,
				DeadLetter: true,
				Dashboard:  true,
			},
			available: JobCapabilities{
				Delayed:    true,
				Retries:    true,
				Cron:       true,
				Priority:   true,
				DeadLetter: true,
				Dashboard:  true,
			},
			want: nil,
		},
		{
			name: "missing subset in stable order",
			required: JobCapabilities{
				Delayed:    true,
				Retries:    true,
				Cron:       true,
				DeadLetter: true,
			},
			available: JobCapabilities{
				Delayed: true,
				Cron:    false,
			},
			want: []string{"retries", "cron", "dead_letter"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.available.Missing(tt.required)
			if len(got) != len(tt.want) {
				t.Fatalf("len mismatch: got=%d want=%d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("index %d mismatch: got=%q want=%q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestValidateJobCapabilities(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when capabilities satisfy requirements", func(t *testing.T) {
		t.Parallel()
		err := ValidateJobCapabilities(
			JobCapabilities{Delayed: true, Retries: true},
			JobCapabilities{Delayed: true, Retries: true, Cron: true},
		)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("returns clear missing-capabilities error", func(t *testing.T) {
		t.Parallel()
		err := ValidateJobCapabilities(
			JobCapabilities{Delayed: true, Retries: true, Priority: true},
			JobCapabilities{Delayed: true},
		)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		want := "missing required jobs capabilities: retries, priority"
		if err.Error() != want {
			t.Fatalf("unexpected error: got=%q want=%q", err.Error(), want)
		}
	})
}

type testStore struct{}

func (testStore) Ping(context.Context) error { return nil }

func (testStore) WithTx(ctx context.Context, fn TxFunc) error { return fn(ctx) }

type testCache struct{}

func (testCache) Get(context.Context, string) ([]byte, bool, error) { return nil, false, nil }

func (testCache) Set(context.Context, string, []byte, time.Duration) error { return nil }

func (testCache) Delete(context.Context, string) error { return nil }

func (testCache) InvalidatePrefix(context.Context, string) error { return nil }

func (testCache) Close() error { return nil }

type testSubscription struct{}

func (testSubscription) Close() error { return nil }

type testPubSub struct{}

func (testPubSub) Publish(context.Context, string, []byte) error { return nil }

func (testPubSub) Subscribe(context.Context, string, MessageHandler) (Subscription, error) {
	return testSubscription{}, nil
}

func (testPubSub) Close() error { return nil }

type testJobs struct{}

func (testJobs) Register(string, JobHandler) error { return nil }

func (testJobs) Enqueue(context.Context, string, []byte, EnqueueOptions) (string, error) {
	return "job-1", nil
}

func (testJobs) StartWorker(context.Context) error { return nil }

func (testJobs) StartScheduler(context.Context) error { return nil }

func (testJobs) Stop(context.Context) error { return nil }

func (testJobs) Capabilities() JobCapabilities { return JobCapabilities{} }

type testBlobStorage struct{}

func (testBlobStorage) Put(context.Context, PutObjectInput) (StoredObject, error) {
	return StoredObject{Bucket: "b", Key: "k"}, nil
}

func (testBlobStorage) Delete(context.Context, string, string) error { return nil }

func (testBlobStorage) PresignGet(context.Context, string, string, time.Duration) (string, error) {
	return "", nil
}

type testMailer struct{}

func (testMailer) Send(context.Context, MailMessage) error { return nil }

func TestInterfaceSatisfactionAndBasicUsage(t *testing.T) {
	t.Parallel()

	var (
		store Store       = testStore{}
		cache Cache       = testCache{}
		ps    PubSub      = testPubSub{}
		jobs  Jobs        = testJobs{}
		blob  BlobStorage = testBlobStorage{}
		mail  Mailer      = testMailer{}
	)

	ctx := context.Background()
	if err := store.Ping(ctx); err != nil {
		t.Fatalf("store ping failed: %v", err)
	}
	if err := cache.Close(); err != nil {
		t.Fatalf("cache close failed: %v", err)
	}
	sub, err := ps.Subscribe(ctx, "topic", func(context.Context, string, []byte) error { return nil })
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}
	if err := sub.Close(); err != nil {
		t.Fatalf("subscription close failed: %v", err)
	}
	if _, err := jobs.Enqueue(ctx, "job", nil, EnqueueOptions{}); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	if _, err := blob.Put(ctx, PutObjectInput{}); err != nil {
		t.Fatalf("blob put failed: %v", err)
	}
	if err := mail.Send(ctx, MailMessage{Subject: "test"}); err != nil {
		t.Fatalf("mail send failed: %v", err)
	}
}
