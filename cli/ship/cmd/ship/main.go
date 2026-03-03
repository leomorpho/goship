package main

import (
	"os"

	shipcli "github.com/leomorpho/goship/cli/ship"
)

func main() {
	os.Exit(shipcli.Run(os.Args[1:]))
}
