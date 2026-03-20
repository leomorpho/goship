package commands

import (
	"context"
	"fmt"

	"github.com/leomorpho/goship/app/foundation"
)

type ExampleCommand struct {
	Container *foundation.Container
}

func (c *ExampleCommand) Name() string {
	return "example:run"
}

func (c *ExampleCommand) Description() string {
	return "Print a simple command-run confirmation."
}

func (c *ExampleCommand) Run(_ context.Context, args []string) error {
	fmt.Printf("example:run executed with %d arg(s)\n", len(args))
	return nil
}
