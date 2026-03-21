package main

import (
	"context"
	"testing"

	"github.com/leomorpho/goship/config"
	frameworkbootstrap "github.com/leomorpho/goship/framework/bootstrap"
	"github.com/stretchr/testify/require"
)

func TestWireJobsModuleInProcNoOp(t *testing.T) {
	t.Parallel()

	c := &frameworkbootstrap.Container{
		Config: &config.Config{
			Adapters: config.AdaptersConfig{Jobs: "inproc"},
		},
	}

	require.NoError(t, wireJobsModule(c))
}

func TestWireJobsModuleUnsupportedAdapterFailsFast(t *testing.T) {
	t.Parallel()

	c := &frameworkbootstrap.Container{
		Config: &config.Config{
			Adapters: config.AdaptersConfig{Jobs: "unsupported"},
		},
	}

	err := wireJobsModule(c)
	require.EqualError(t, err, `unsupported jobs adapter "unsupported"`)
}

func TestStartEmbeddedJobsWorkerNoOpOutsideBacklite(t *testing.T) {
	t.Parallel()

	c := &frameworkbootstrap.Container{
		Config: &config.Config{
			Adapters: config.AdaptersConfig{Jobs: "asynq"},
		},
	}

	require.NoError(t, startEmbeddedJobsWorker(context.Background(), c, nil, nil))
}
