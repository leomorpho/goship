package main

import (
	"log"

	"github.com/leomorpho/goship"
)

func main() {
	c := goship.NewContainer()
	defer func() {
		_ = c.Shutdown()
	}()
	log.Println("seed command is temporarily disabled; use explicit SQL seed scripts")
}
