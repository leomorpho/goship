package main

import (
	"os"

	shipcli "github.com/mikestefanello/pagoda/cli/ship"
)

func main() {
	os.Exit(shipcli.Run(os.Args[1:]))
}
