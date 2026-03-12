package store

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	stoamcp "github.com/stoa-hq/stoa/internal/mcp"
)

func stripeCreatePaymentIntentTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("store_stripe_create_payment_intent",
		mcp.WithDescription(
			"Create a Stripe PaymentIntent for a pending order. "+
				"Returns a client_secret (for Stripe.js / mobile SDKs) and the publishable_key. "+
				"Requires the Stripe plugin to be installed. "+
				"Call store_checkout first to create the order, then call this tool to initiate payment.",
		),
		mcp.WithString("order_id",
			mcp.Description("UUID of the pending order to pay for"),
			mcp.Required(),
		),
		mcp.WithString("payment_method_id",
			mcp.Description("UUID of the Stoa PaymentMethod configured for Stripe (provider = stripe)"),
			mcp.Required(),
		),
	)
	return tool, nil
}

func stripeCreatePaymentIntentHandler(client *stoamcp.StoaClient) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		body := map[string]interface{}{
			"order_id":          req.GetString("order_id", ""),
			"payment_method_id": req.GetString("payment_method_id", ""),
		}
		data, err := client.Post("/api/v1/store/stripe/payment-intent", body)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
}
