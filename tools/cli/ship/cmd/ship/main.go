package main

import (
	"os"

	shipcli "github.com/leomorpho/goship/tools/cli/ship/internal/cli"
)

func main() {
	os.Exit(shipcli.Run(os.Args[1:]))
}
