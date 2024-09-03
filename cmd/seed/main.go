package main

import (
	"github.com/mikestefanello/pagoda/pkg/services"
	"github.com/mikestefanello/pagoda/seeder"
)

func main() {
	c := services.NewContainer()
	seeder.SeedUsers(c.Config, c.ORM, true)
	c.Shutdown()
}
