package main

import (
	"context"
	"fmt"
	"log"
	"os"

	appcommands "github.com/leomorpho/goship/app/commands"
	"github.com/leomorpho/goship/app/foundation"
	commandfw "github.com/leomorpho/goship/framework/command"
)

func main() {
	container := foundation.NewContainer()
	defer func() {
		if err := container.Shutdown(); err != nil {
			log.Printf("container shutdown error: %v", err)
		}
	}()

	registry := commandfw.NewRegistry()
	if err := registry.Register(&appcommands.ExampleCommand{Container: container}); err != nil {
		log.Fatalf("failed to register command: %v", err)
	}
	if err := registry.Register(&appcommands.SendTestEmailCommand{Container: container}); err != nil {
		log.Fatalf("failed to register command: %v", err)
	}

	// ship:commands:start
	// ship:commands:end

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/cli/main.go <command> [args...]")
		fmt.Fprintln(os.Stderr, registry.Usage())
		os.Exit(1)
	}

	if err := registry.Run(context.Background(), os.Args[1:]); err != nil {
		log.Fatalf("%v\n%s", err, registry.Usage())
	}
}
