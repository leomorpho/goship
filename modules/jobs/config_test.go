package jobs

import (
	"testing"

	"github.com/leomorpho/goship/db/ent"
)

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "sql requires ent",
			cfg: Config{
				Backend: BackendSQL,
			},
			wantErr: true,
		},
		{
			name: "sql forbids redis settings",
			cfg: Config{
				Backend: BackendSQL,
				Redis: RedisConfig{
					Addr: "localhost:6379",
				},
			},
			wantErr: true,
		},
		{
			name: "redis requires addr",
			cfg: Config{
				Backend: BackendRedis,
			},
			wantErr: true,
		},
		{
			name: "redis forbids ent settings",
			cfg: Config{
				Backend:   BackendRedis,
				EntClient: &ent.Client{},
				Redis: RedisConfig{
					Addr: "localhost:6379",
				},
			},
			wantErr: true,
		},
		{
			name: "redis with addr passes",
			cfg: Config{
				Backend: BackendRedis,
				Redis: RedisConfig{
					Addr: "localhost:6379",
				},
			},
			wantErr: false,
		},
		{
			name: "unknown backend fails",
			cfg: Config{
				Backend: "unknown",
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.cfg.Validate()
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}
