package store

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	stoamcp "github.com/stoa-hq/stoa/internal/mcp"
)

func getShippingMethodsTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("store_get_shipping_methods",
		mcp.WithDescription("List available shipping methods with prices"),
	)
	return tool, nil
}

func getShippingMethodsHandler(client *stoamcp.StoaClient) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Get("/api/v1/store/shipping-methods")
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
}

func getPaymentMethodsTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("store_get_payment_methods",
		mcp.WithDescription("List available payment methods"),
	)
	return tool, nil
}

func getPaymentMethodsHandler(client *stoamcp.StoaClient) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Get("/api/v1/store/payment-methods")
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
}

func checkoutTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("store_checkout",
		mcp.WithDescription("Complete a checkout. Creates an order from the cart. Requires authentication."),
		mcp.WithString("cart_id", mcp.Description("Cart UUID"), mcp.Required()),
		mcp.WithString("shipping_method_id", mcp.Description("Shipping method UUID"), mcp.Required()),
		mcp.WithString("payment_method_id", mcp.Description("Payment method UUID"), mcp.Required()),
		mcp.WithObject("shipping_address", mcp.Description("Shipping address object with: first_name, last_name, street, city, zip, country"), mcp.Required()),
		mcp.WithObject("billing_address", mcp.Description("Billing address object (same fields as shipping_address). If omitted, shipping address is used.")),
	)
	return tool, nil
}

func checkoutHandler(client *stoamcp.StoaClient) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		body := map[string]interface{}{
			"cart_id":            args["cart_id"],
			"shipping_method_id": args["shipping_method_id"],
			"payment_method_id":  args["payment_method_id"],
			"shipping_address":   args["shipping_address"],
		}
		if ba, ok := args["billing_address"]; ok {
			body["billing_address"] = ba
		}

		data, err := client.Post("/api/v1/store/checkout", body)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
}
