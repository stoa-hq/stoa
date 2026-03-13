package mcp

import (
	"context"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestScopedMCPServer_ValidPrefix(t *testing.T) {
	srv := server.NewMCPServer("test", "1.0.0", server.WithToolCapabilities(true))
	scoped := NewScopedMCPServer(srv, "stripe")

	tool := mcplib.NewTool("store_stripe_create_payment_intent",
		mcplib.WithDescription("test tool"),
	)

	// Should not panic.
	scoped.AddTool(tool, func(_ context.Context, _ mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		return nil, nil
	})

	if got := srv.GetTool("store_stripe_create_payment_intent"); got == nil {
		t.Error("tool was not registered on the underlying server")
	}
}

func TestScopedMCPServer_InvalidPrefix_Panics(t *testing.T) {
	srv := server.NewMCPServer("test", "1.0.0", server.WithToolCapabilities(true))
	scoped := NewScopedMCPServer(srv, "stripe")

	tool := mcplib.NewTool("store_list_products",
		mcplib.WithDescription("hijack built-in tool"),
	)

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for invalid tool name prefix")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected string panic, got %T", r)
		}
		if msg == "" {
			t.Fatal("panic message should not be empty")
		}
	}()

	scoped.AddTool(tool, func(_ context.Context, _ mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		return nil, nil
	})
}

func TestScopedMCPServer_WrongPluginPrefix_Panics(t *testing.T) {
	srv := server.NewMCPServer("test", "1.0.0", server.WithToolCapabilities(true))
	scoped := NewScopedMCPServer(srv, "stripe")

	// Another plugin's prefix — should be rejected.
	tool := mcplib.NewTool("store_paypal_checkout",
		mcplib.WithDescription("wrong plugin prefix"),
	)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for wrong plugin prefix")
		}
	}()

	scoped.AddTool(tool, func(_ context.Context, _ mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		return nil, nil
	})
}
