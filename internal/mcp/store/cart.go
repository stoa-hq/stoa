package store

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	stoamcp "github.com/stoa-hq/stoa/internal/mcp"
)

func createCartTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("store_create_cart",
		mcp.WithDescription("Create a new shopping cart. Returns a cart ID that must be used for all subsequent cart operations."),
	)
	return tool, nil
}

func createCartHandler(client *stoamcp.StoaClient) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Post("/api/v1/store/cart", map[string]interface{}{})
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
}

func getCartTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("store_get_cart",
		mcp.WithDescription("Get the current contents and totals of a shopping cart"),
		mcp.WithString("cart_id", mcp.Description("Cart UUID"), mcp.Required()),
	)
	return tool, nil
}

func getCartHandler(client *stoamcp.StoaClient) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cartID := req.GetString("cart_id", "")
		if cartID == "" {
			return mcp.NewToolResultError("cart_id is required"), nil
		}

		data, err := client.Get("/api/v1/store/cart/" + cartID)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
}

func addToCartTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("store_add_to_cart",
		mcp.WithDescription("Add a product variant to the shopping cart"),
		mcp.WithString("cart_id", mcp.Description("Cart UUID"), mcp.Required()),
		mcp.WithString("variant_id", mcp.Description("Product variant UUID to add"), mcp.Required()),
		mcp.WithNumber("quantity", mcp.Description("Quantity to add (default: 1)")),
	)
	return tool, nil
}

func addToCartHandler(client *stoamcp.StoaClient) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cartID := req.GetString("cart_id", "")
		if cartID == "" {
			return mcp.NewToolResultError("cart_id is required"), nil
		}

		body := map[string]interface{}{
			"variant_id": req.GetString("variant_id", ""),
		}
		if q := req.GetInt("quantity", 0); q > 0 {
			body["quantity"] = q
		}

		data, err := client.Post("/api/v1/store/cart/"+cartID+"/items", body)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
}

func updateCartItemTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("store_update_cart_item",
		mcp.WithDescription("Update the quantity of an item in the cart"),
		mcp.WithString("cart_id", mcp.Description("Cart UUID"), mcp.Required()),
		mcp.WithString("item_id", mcp.Description("Cart item UUID"), mcp.Required()),
		mcp.WithNumber("quantity", mcp.Description("New quantity"), mcp.Required()),
	)
	return tool, nil
}

func updateCartItemHandler(client *stoamcp.StoaClient) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cartID := req.GetString("cart_id", "")
		itemID := req.GetString("item_id", "")

		body := map[string]interface{}{
			"quantity": req.GetInt("quantity", 1),
		}

		data, err := client.Put("/api/v1/store/cart/"+cartID+"/items/"+itemID, body)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
}

func removeFromCartTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("store_remove_from_cart",
		mcp.WithDescription("Remove an item from the shopping cart"),
		mcp.WithString("cart_id", mcp.Description("Cart UUID"), mcp.Required()),
		mcp.WithString("item_id", mcp.Description("Cart item UUID to remove"), mcp.Required()),
	)
	return tool, nil
}

func removeFromCartHandler(client *stoamcp.StoaClient) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cartID := req.GetString("cart_id", "")
		itemID := req.GetString("item_id", "")

		data, err := client.Delete("/api/v1/store/cart/" + cartID + "/items/" + itemID)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
}
