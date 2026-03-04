package main

import (
	"github.com/leomorpho/goship/apps/site/foundation"
	"github.com/leomorpho/goship/tools/seeder"
)

func main() {
	c := foundation.NewContainer()
	seeder.SeedUsers(c.Config, c.ORM, true)
	c.Shutdown()
}
