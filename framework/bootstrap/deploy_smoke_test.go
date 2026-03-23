package bootstrap

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/health"
	"github.com/stretchr/testify/require"
)

func TestDeployCriticalRuntimeSmoke(t *testing.T) {
	t.Run("embedded boot applies migrations and reports readiness", func(t *testing.T) {
		t.Setenv("PAGODA_APP_ENVIRONMENT", "test")
		t.Setenv("PAGODA_DB_PATH", filepath.Join(t.TempDir(), "deploy-smoke.db"))

		container := NewContainer(nil)
		t.Cleanup(func() {
			_ = container.Shutdown()
		})

		require.NotNil(t, container.Database)
		require.NotNil(t, container.Health)

		var usersTable int
		require.NoError(t, container.Database.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='users'`,
		).Scan(&usersTable))
		require.Equal(t, 1, usersTable, "expected boot path to provision users table")

		results, ok := container.Health.Run(context.Background())
		require.True(t, ok, "expected readiness checks to pass")
		require.Equal(t, health.StatusOK, results["db"].Status)
	})

	t.Run("worker startup accepts asynq and rejects backlite", func(t *testing.T) {
		asynqRuntime, err := WireJobsRuntime(&config.Config{
			Adapters: config.AdaptersConfig{Jobs: "asynq"},
			Cache: config.CacheConfig{
				Hostname: "localhost",
				Port:     6379,
				Database: 0,
			},
		}, nil, JobsProcessWorker)
		require.NoError(t, err)
		require.NotNil(t, asynqRuntime.Jobs)
		require.NotNil(t, asynqRuntime.Inspector)

		_, err = WireJobsRuntime(&config.Config{
			Adapters: config.AdaptersConfig{Jobs: "backlite"},
		}, nil, JobsProcessWorker)
		require.EqualError(t, err, `jobs adapter "backlite" runs in cmd/web and cannot be started in cmd/worker`)
	})
}
