package main

import (
	"log"

	"github.com/mark3labs/mcp-go/server"

	stoamcp "github.com/epoxx-arch/stoa/internal/mcp"
	"github.com/epoxx-arch/stoa/internal/mcp/admin"
)

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
- Audit log: review all changes

Prices are in cents (1999 = 19.99 EUR). Tax rates are in basis points (1900 = 19%).
Requires an API key with appropriate admin permissions.`),
	)

	admin.RegisterTools(s, client)

	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
