package main

import (
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

func TestWireJobsModuleBackliteFailsFast(t *testing.T) {
	t.Parallel()

	c := &frameworkbootstrap.Container{
		Config: &config.Config{
			Adapters: config.AdaptersConfig{Jobs: "backlite"},
		},
	}

	err := wireJobsModule(c)
	require.EqualError(t, err, `jobs adapter "backlite" runs in cmd/web and cannot be started in cmd/worker`)
}
