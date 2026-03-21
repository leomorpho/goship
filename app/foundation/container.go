package foundation

import (
	"github.com/leomorpho/goship/app/schedules"
	frameworkbootstrap "github.com/leomorpho/goship/framework/bootstrap"
	"github.com/leomorpho/goship/framework/core"
	"github.com/robfig/cron/v3"
)

type Container = frameworkbootstrap.Container

// NewContainer keeps the starter-facing entrypoint while delegating generic runtime wiring to framework/bootstrap.
func NewContainer() *Container {
	// ship:container:start
	// ship:container:end
	return frameworkbootstrap.NewContainer(func(scheduler *cron.Cron, jobs func() core.Jobs) {
		schedules.Register(scheduler, jobs)
	})
}
