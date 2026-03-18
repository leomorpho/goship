package bootstrap

import (
	"testing"

	"github.com/leomorpho/goship/config"
	"github.com/stretchr/testify/require"
)

func TestWireJobsRuntime(t *testing.T) {
	t.Parallel()

	t.Run("fails fast on nil config", func(t *testing.T) {
		t.Parallel()
		_, err := WireJobsRuntime(nil, nil, JobsProcessWeb)
		require.EqualError(t, err, "missing runtime config")
	})

	t.Run("inproc returns no wired jobs", func(t *testing.T) {
		t.Parallel()
		cfg := &config.Config{Adapters: config.AdaptersConfig{Jobs: "inproc"}}
		runtime, err := WireJobsRuntime(cfg, nil, JobsProcessWeb)
		require.NoError(t, err)
		require.Nil(t, runtime.Jobs)
		require.Nil(t, runtime.Inspector)
	})

	t.Run("unsupported adapter fails fast", func(t *testing.T) {
		t.Parallel()
		cfg := &config.Config{Adapters: config.AdaptersConfig{Jobs: "nope"}}
		_, err := WireJobsRuntime(cfg, nil, JobsProcessWeb)
		require.EqualError(t, err, `unsupported jobs adapter "nope"`)
	})

	t.Run("worker rejects backlite", func(t *testing.T) {
		t.Parallel()
		cfg := &config.Config{Adapters: config.AdaptersConfig{Jobs: "backlite"}}
		_, err := WireJobsRuntime(cfg, nil, JobsProcessWorker)
		require.EqualError(t, err, `jobs adapter "backlite" runs in cmd/web and cannot be started in cmd/worker`)
	})

	t.Run("asynq wires module bridges", func(t *testing.T) {
		t.Parallel()
		cfg := &config.Config{
			Adapters: config.AdaptersConfig{Jobs: "asynq"},
			Cache: config.CacheConfig{
				Hostname: "localhost",
				Port:     6379,
				Database: 0,
			},
		}
		runtime, err := WireJobsRuntime(cfg, nil, JobsProcessWeb)
		require.NoError(t, err)
		require.NotNil(t, runtime.Jobs)
		require.NotNil(t, runtime.Inspector)
	})
}
