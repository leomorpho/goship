package main

import (
	"context"
	"log"
	"os"

	"github.com/mikestefanello/pagoda/mcp/ship/internal/server"
)

func main() {
	docsRoot := os.Getenv("SHIP_MCP_DOCS_ROOT")
	if docsRoot == "" {
		docsRoot = "docs"
	}

	if err := server.Run(context.Background(), os.Stdin, os.Stdout, os.Stderr, docsRoot); err != nil {
		log.Fatal(err)
	}
}
