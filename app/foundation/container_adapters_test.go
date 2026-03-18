package foundation

import (
	"testing"

	"github.com/leomorpho/goship/config"
)

func TestContainerValidateAdapterPlan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cfg       *config.Config
		wantPanic bool
	}{
		{
			name: "valid default-like selection",
			cfg: &config.Config{
				Adapters: config.AdaptersConfig{
					DB:     "postgres",
					Cache:  "memory",
					Jobs:   "inproc",
					PubSub: "inproc",
				},
				Runtime: config.RuntimeConfig{
					Profile: config.RuntimeProfileServerDB,
				},
			},
			wantPanic: false,
		},
		{
			name: "invalid adapter name panics",
			cfg: &config.Config{
				Adapters: config.AdaptersConfig{
					DB:     "postgres",
					Cache:  "memory",
					Jobs:   "unknown",
					PubSub: "inproc",
				},
			},
			wantPanic: true,
		},
		{
			name: "distributed supported jobs backend",
			cfg: &config.Config{
				Adapters: config.AdaptersConfig{
					DB:     "postgres",
					Cache:  "redis",
					Jobs:   "asynq",
					PubSub: "redis",
				},
				Runtime: config.RuntimeConfig{
					Profile: config.RuntimeProfileDistributed,
				},
			},
			wantPanic: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := &Container{Config: tt.cfg}
			panicked := didPanic(func() {
				c.validateAdapterPlan()
			})
			if panicked != tt.wantPanic {
				t.Fatalf("panic mismatch: got=%v want=%v", panicked, tt.wantPanic)
			}
		})
	}
}

func TestContainerValidateAdapterPlan_StrictPubSubDependencyContract(t *testing.T) {
	t.Parallel()

	c := &Container{
		Config: &config.Config{
			Adapters: config.AdaptersConfig{
				DB:     "postgres",
				Cache:  "memory",
				Jobs:   "inproc",
				PubSub: "redis",
			},
		},
	}

	if !didPanic(func() { c.validateAdapterPlan() }) {
		t.Fatal("expected panic when redis pubsub is configured without redis cache backing")
	}
}

func didPanic(fn func()) (panicked bool) {
	defer func() {
		if recover() != nil {
			panicked = true
		}
	}()
	fn()
	return panicked
}
