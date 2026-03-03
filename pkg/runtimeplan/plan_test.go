package runtimeplan

import (
	"testing"

	"github.com/leomorpho/goship/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *config.Config
		wantErr   string
		assertion func(t *testing.T, p Plan)
	}{
		{
			name: "server db defaults and web process",
			cfg: &config.Config{
				Runtime: config.RuntimeConfig{
					Profile: config.RuntimeProfileServerDB,
				},
				Processes: config.ProcessesConfig{
					Web: true,
				},
				Adapters: config.AdaptersConfig{
					DB:    "postgres",
					Jobs:  "inproc",
					Cache: "memory",
				},
			},
			assertion: func(t *testing.T, p Plan) {
				assert.Equal(t, "server-db", p.Profile)
				assert.True(t, p.RunWeb)
				assert.False(t, p.RunWorker)
				assert.Equal(t, "postgres", p.Adapters.DB)
			},
		},
		{
			name: "empty profile normalizes to server db",
			cfg: &config.Config{
				Processes: config.ProcessesConfig{
					Web: true,
				},
				Adapters: config.AdaptersConfig{
					Jobs: "inproc",
				},
			},
			assertion: func(t *testing.T, p Plan) {
				assert.Equal(t, "server-db", p.Profile)
			},
		},
		{
			name: "invalid when no processes enabled",
			cfg: &config.Config{
				Runtime: config.RuntimeConfig{
					Profile: config.RuntimeProfileServerDB,
				},
			},
			wantErr: "at least one of web/worker/scheduler must be enabled",
		},
		{
			name: "invalid profile",
			cfg: &config.Config{
				Runtime: config.RuntimeConfig{
					Profile: "unknown-profile",
				},
				Processes: config.ProcessesConfig{
					Web: true,
				},
			},
			wantErr: "unknown runtime profile",
		},
		{
			name: "distributed rejects inproc jobs",
			cfg: &config.Config{
				Runtime: config.RuntimeConfig{
					Profile: config.RuntimeProfileDistributed,
				},
				Processes: config.ProcessesConfig{
					Web: true,
				},
				Adapters: config.AdaptersConfig{
					Jobs: "inproc",
				},
			},
			wantErr: "invalid distributed jobs backend",
		},
		{
			name: "distributed accepts durable jobs backend",
			cfg: &config.Config{
				Runtime: config.RuntimeConfig{
					Profile: config.RuntimeProfileDistributed,
				},
				Processes: config.ProcessesConfig{
					Web:       true,
					Worker:    true,
					CoLocated: true,
				},
				Adapters: config.AdaptersConfig{
					Jobs: "dbqueue",
				},
			},
			assertion: func(t *testing.T, p Plan) {
				assert.Equal(t, "distributed", p.Profile)
				assert.True(t, p.RunWorker)
				assert.True(t, p.CoLocated)
				assert.Equal(t, "dbqueue", p.Adapters.Jobs)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := Resolve(tt.cfg)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, tt.assertion)
			tt.assertion(t, plan)
		})
	}
}
