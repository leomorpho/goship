package adapters

import (
	"strings"
	"testing"

	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/core"
)

func TestRegistryValidateSelection(t *testing.T) {
	t.Parallel()

	reg := NewDefaultRegistry()
	tests := []struct {
		name    string
		sel     Selection
		wantErr string
	}{
		{
			name: "valid selection",
			sel: Selection{
				DB:     "postgres",
				Cache:  "memory",
				Jobs:   "inproc",
				PubSub: "inproc",
			},
		},
		{
			name: "unknown db adapter",
			sel: Selection{
				DB:     "oracle",
				Cache:  "memory",
				Jobs:   "inproc",
				PubSub: "inproc",
			},
			wantErr: "unknown db adapter",
		},
		{
			name: "missing cache adapter",
			sel: Selection{
				DB:     "postgres",
				Cache:  "",
				Jobs:   "inproc",
				PubSub: "inproc",
			},
			wantErr: "missing cache adapter name",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := reg.ValidateSelection(tt.sel)
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
			}
		})
	}
}

func TestRegistryValidateRequirements(t *testing.T) {
	t.Parallel()

	reg := NewDefaultRegistry()
	sel := Selection{
		DB:     "postgres",
		Cache:  "memory",
		Jobs:   "dbqueue",
		PubSub: "inproc",
	}

	err := reg.ValidateRequirements(sel, Requirements{
		Jobs: core.JobCapabilities{
			Cron:       true,
			Retries:    true,
			Delayed:    true,
			DeadLetter: true,
		},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	err = reg.ValidateRequirements(Selection{Jobs: "inproc"}, Requirements{
		Jobs: core.JobCapabilities{Priority: true},
	})
	if err == nil {
		t.Fatal("expected capability error, got nil")
	}
	if !strings.Contains(err.Error(), "missing required jobs capabilities: priority") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequirementsFromConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  *config.Config
		want Requirements
	}{
		{
			name: "nil config",
			cfg:  nil,
			want: Requirements{},
		},
		{
			name: "distributed requires retries and delayed",
			cfg: &config.Config{
				Runtime: config.RuntimeConfig{Profile: config.RuntimeProfileDistributed},
			},
			want: Requirements{
				Jobs: core.JobCapabilities{
					Retries: true,
					Delayed: true,
				},
			},
		},
		{
			name: "scheduler requires cron",
			cfg: &config.Config{
				Processes: config.ProcessesConfig{Scheduler: true},
			},
			want: Requirements{
				Jobs: core.JobCapabilities{
					Cron: true,
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := RequirementsFromConfig(tt.cfg)
			if got != tt.want {
				t.Fatalf("requirements mismatch: got=%+v want=%+v", got, tt.want)
			}
		})
	}
}
