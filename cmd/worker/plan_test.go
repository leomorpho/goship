package main

import (
	"strings"
	"testing"

	"github.com/leomorpho/goship/config"
)

func TestValidateWorkerConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     config.Config
		wantErr string
	}{
		{
			name: "asynq jobs adapter is valid",
			cfg: config.Config{
				Adapters: config.AdaptersConfig{Jobs: "asynq"},
			},
		},
		{
			name: "inproc jobs adapter is rejected",
			cfg: config.Config{
				Adapters: config.AdaptersConfig{Jobs: "inproc"},
			},
			wantErr: `worker requires jobs adapter "asynq"`,
		},
		{
			name: "backlite jobs adapter is rejected",
			cfg: config.Config{
				Adapters: config.AdaptersConfig{Jobs: "backlite"},
			},
			wantErr: `worker requires jobs adapter "asynq"`,
		},
		{
			name: "empty jobs adapter is rejected",
			cfg: config.Config{
				Adapters: config.AdaptersConfig{Jobs: ""},
			},
			wantErr: `worker requires jobs adapter "asynq"`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateWorkerConfig(tt.cfg)
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
