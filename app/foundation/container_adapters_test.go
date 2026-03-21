package foundation

import (
	"testing"

	"github.com/leomorpho/goship/config"
	frameworkbootstrap "github.com/leomorpho/goship/framework/bootstrap"
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

			_, err := frameworkbootstrap.ResolveAdapterPlan(tt.cfg)
			gotErr := err != nil
			if gotErr != tt.wantPanic {
				t.Fatalf("error mismatch: got=%v want=%v err=%v", gotErr, tt.wantPanic, err)
			}
		})
	}
}

func TestContainerValidateAdapterPlan_StrictPubSubDependencyContract(t *testing.T) {
	t.Parallel()

	_, err := frameworkbootstrap.ResolveAdapterPlan(&config.Config{
		Adapters: config.AdaptersConfig{
			DB:     "postgres",
			Cache:  "memory",
			Jobs:   "inproc",
			PubSub: "redis",
		},
	})

	if err == nil {
		t.Fatal("expected error when redis pubsub is configured without redis cache backing")
	}
}
