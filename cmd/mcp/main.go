package main

import (
	"context"
	"log"
	"os"

	workmcp "github.com/kelvins-io/eino-multi-agent/internal/mcp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	base := getenv("EINO_API_BASE", "http://127.0.0.1:8180")
	token := getenv("EINO_API_TOKEN", "")
	if token == "" {
		log.Fatal("EINO_API_TOKEN is required (user JWT from /api/v1/auth/login)")
	}
	name := getenv("EINO_MCP_NAME", "eino-work-harness")
	version := getenv("EINO_MCP_VERSION", "1.0.0")

	client := workmcp.NewHTTPClient(base, token)
	server := workmcp.NewServer(name, version, client)
	if err := server.Run(context.Background(), &mcpsdk.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
