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
)

func main() {
	cfg := stoamcp.LoadConfig()
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
