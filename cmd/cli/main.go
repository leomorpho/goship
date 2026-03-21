package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/leomorpho/goship"
	commandfw "github.com/leomorpho/goship/framework/command"
)

func main() {
	container := goship.NewContainer()
	defer func() {
		if err := container.Shutdown(); err != nil {
			log.Printf("container shutdown error: %v", err)
		}
	}()

	registry := commandfw.NewRegistry()
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
