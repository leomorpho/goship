package adapters

import (
	"strings"
	"testing"

	"github.com/leomorpho/goship/config"
)

func TestResolveFromConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     *config.Config
		wantErr string
	}{
		{
			name: "valid configuration",
			cfg: &config.Config{
				Adapters: config.AdaptersConfig{
					DB:     "postgres",
					Cache:  "memory",
					Jobs:   "inproc",
					PubSub: "inproc",
				},
			},
		},
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: "nil config",
		},
		{
			name: "invalid adapter selection",
			cfg: &config.Config{
				Adapters: config.AdaptersConfig{
					DB:     "postgres",
					Cache:  "memcache",
					Jobs:   "inproc",
					PubSub: "inproc",
				},
			},
			wantErr: "unknown cache adapter",
		},
		{
			name: "capability mismatch",
			cfg: &config.Config{
				Adapters: config.AdaptersConfig{
					DB:     "postgres",
					Cache:  "redis",
					Jobs:   "inproc",
					PubSub: "redis",
				},
				Runtime: config.RuntimeConfig{
					Profile: config.RuntimeProfileDistributed,
				},
				Processes: config.ProcessesConfig{
					Scheduler: true,
				},
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ResolveFromConfig(tt.cfg)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}
			if tt.cfg != nil && got.Selection.Jobs != tt.cfg.Adapters.Jobs {
				t.Fatalf("unexpected selection jobs: got=%q want=%q", got.Selection.Jobs, tt.cfg.Adapters.Jobs)
			}
		})
	}
}
