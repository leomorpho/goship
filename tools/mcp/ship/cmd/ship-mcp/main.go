package main

import (
	"context"
	"log"
	"os"

	"github.com/leomorpho/goship/tools/mcp/ship/v2/internal/server"
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
