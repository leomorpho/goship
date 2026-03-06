package main

import (
	"log"

	"github.com/leomorpho/goship/app/foundation"
)

func main() {
	c := foundation.NewContainer()
	defer func() {
		_ = c.Shutdown()
	}()
	log.Println("seed command is temporarily disabled; use explicit SQL seed scripts")
}
