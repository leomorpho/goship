package main

import (
	"github.com/leomorpho/goship/apps/goship/foundation"
	"github.com/leomorpho/goship/seeder"
)

func main() {
	c := foundation.NewContainer()
	seeder.SeedUsers(c.Config, c.ORM, true)
	c.Shutdown()
}
