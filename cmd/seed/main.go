package main

import (
	"github.com/leomorpho/goship/pkg/services"
	"github.com/leomorpho/goship/seeder"
)

func main() {
	c := services.NewContainer()
	seeder.SeedUsers(c.Config, c.ORM, true)
	c.Shutdown()
}
