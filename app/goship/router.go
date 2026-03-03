package goship

import (
	"github.com/mikestefanello/pagoda/app/goship/web/routes"
	"github.com/mikestefanello/pagoda/pkg/services"
)

// BuildRouter is the canonical app-level router entrypoint.
// Detailed domain route composition lives under app/goship/web/routes.
func BuildRouter(c *services.Container) error {
	return routes.BuildRouter(c)
}
