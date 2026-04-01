package main

import (
	"os"

	shipcli "github.com/leomorpho/goship/tools/cli/ship/v2/internal/cli"
)

func main() {
	os.Exit(shipcli.Run(os.Args[1:]))
}
