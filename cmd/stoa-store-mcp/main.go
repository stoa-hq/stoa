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
	"github.com/stoa-hq/stoa/internal/mcp/store"
	"github.com/stoa-hq/stoa/pkg/sdk"
)

// safeRegisterPluginTools wraps plugin MCP tool registration in a recover
// to prevent a panicking plugin from crashing the server. It passes a
// ScopedMCPServer that enforces tool name prefixes (store_{pluginName}_*).
func safeRegisterPluginTools(mp sdk.MCPStorePlugin, s *server.MCPServer, client sdk.StoreAPIClient) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	scoped := stoamcp.NewScopedMCPServer(s, mp.Name())
	mp.RegisterStoreMCPTools(scoped, client)
	return nil
}

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
	if cfg.APIKey == "" {
		log.Println("WARNING: STOA_MCP_API_KEY is not set — all tool calls will fail with 401")
	}
	client := stoamcp.NewStoaClient(cfg)

	s := server.NewMCPServer(
		"stoa-store",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithRecovery(),
		server.WithInstructions(`Stoa Store MCP Server — allows AI agents to browse products, manage shopping carts, and complete purchases in a Stoa e-commerce store.

Typical workflow:
1. Browse products with store_list_products or store_search
2. Get product details with store_get_product
3. Create a cart with store_create_cart (remember the cart_id!)
4. Add items with store_add_to_cart
5. Check shipping/payment options
6. Complete with store_checkout

Prices are in cents (e.g. 1999 = 19.99 EUR). Tax rates are in basis points (1900 = 19%).`),
	)

	store.RegisterTools(s, client)

	// Let installed plugins register their own Store MCP tools.
	// Plugins receive a store-scoped client (restricted to /api/v1/store/* paths)
	// and a scoped MCP server (enforcing tool name prefixes).
	scopedClient := stoamcp.NewStoreScopedClient(client)
	for _, p := range sdk.RegisteredPlugins() {
		if mp, ok := p.(sdk.MCPStorePlugin); ok {
			if err := safeRegisterPluginTools(mp, s, scopedClient); err != nil {
				log.Printf("WARNING: plugin %s failed to register MCP tools: %v", p.Name(), err)
				continue
			}
			log.Printf("registered store MCP tools from plugin: %s", p.Name())
		}
	}

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
		log.Printf("stoa-store-mcp listening on %s (SSE)", addr)
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
