package goship

import (
	frameworkbootstrap "github.com/leomorpho/goship/framework/bootstrap"
	"github.com/leomorpho/goship/framework/core"
	"github.com/robfig/cron/v3"
)

type Container = frameworkbootstrap.Container

// NewContainer builds the canonical GoShip runtime container.
func NewContainer() *Container {
	// ship:container:start
	// ship:container:end
	return frameworkbootstrap.NewContainer(func(scheduler *cron.Cron, jobs func() core.Jobs) {
		RegisterSchedules(scheduler, jobs)
	})
}
