package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mark3labs/mcp-go/server"

	stoamcp "github.com/stoa-hq/stoa/internal/mcp"
	"github.com/stoa-hq/stoa/internal/mcp/admin"
)

func useStdio() bool {
	for _, arg := range os.Args[1:] {
		if arg == "--stdio" {
			return true
		}
	}
	return os.Getenv("STOA_MCP_TRANSPORT") == "stdio"
}

func main() {
	cfg := stoamcp.LoadConfig()
	client := stoamcp.NewStoaClient(cfg)

	s := server.NewMCPServer(
		"stoa-admin",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithRecovery(),
		server.WithInstructions(`Stoa Admin MCP Server — allows AI agents to manage a Stoa e-commerce shop.

Available operations:
- Products: CRUD operations, variant management
- Orders: list, view details, update status
- Discounts: create and manage discount codes
- Customers: view and manage customer accounts
- Categories: organize product categories
- Tags: manage product tags
- Media: list and delete uploaded files
- Shipping/Tax/Payment: view configuration
- Warehouses: CRUD operations, stock management per warehouse/product
- Audit log: review all changes

Prices are in cents (1999 = 19.99 EUR). Tax rates are in basis points (1900 = 19%).
Requires an API key with appropriate admin permissions.`),
	)

	admin.RegisterTools(s, client)

	if useStdio() {
		stdio := server.NewStdioServer(s)
		if err := stdio.Listen(context.Background(), os.Stdin, os.Stdout); err != nil {
			log.Fatalf("stdio server error: %v", err)
		}
		return
	}

	addr := fmt.Sprintf(":%d", cfg.Port)
	sseServer := server.NewSSEServer(s,
		server.WithBaseURL(cfg.BaseURL),
		server.WithSSEEndpoint("/sse"),
		server.WithMessageEndpoint("/message"),
		server.WithKeepAlive(true),
	)

	go func() {
		log.Printf("stoa-admin-mcp listening on %s (SSE)", addr)
		if err := sseServer.Start(addr); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	if err := sseServer.Shutdown(context.Background()); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
