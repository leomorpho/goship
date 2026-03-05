package main

import (
	"github.com/leomorpho/goship/app/foundation"
	"github.com/leomorpho/goship/tools/seeder"
)

func main() {
	c := foundation.NewContainer()
	seeder.SeedUsers(c.Config, c.ORM, c.Database, c.Config.Adapters.DB, true)
	c.Shutdown()
}
