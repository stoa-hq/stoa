package admin

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	stoamcp "github.com/epoxx-arch/stoa/internal/mcp"
)

func adminListDiscounts(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_list_discounts",
		mcp.WithDescription("List all discount codes and rules"),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("limit", mcp.Description("Items per page")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path := "/api/v1/admin/discounts?" + buildQueryParams(req, "page", "limit")
		data, err := client.Get(path)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminGetDiscount(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_get_discount",
		mcp.WithDescription("Get details of a discount"),
		mcp.WithString("id", mcp.Description("Discount UUID"), mcp.Required()),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Get("/api/v1/admin/discounts/" + req.GetString("id", ""))
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminCreateDiscount(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_create_discount",
		mcp.WithDescription("Create a new discount code. Amount is in cents for fixed, or basis points for percentage (e.g. 2000 = 20%)."),
		mcp.WithString("code", mcp.Description("Discount code (e.g. SUMMER20)"), mcp.Required()),
		mcp.WithString("type", mcp.Description("Discount type: percentage or fixed"), mcp.Required()),
		mcp.WithNumber("amount", mcp.Description("Amount in cents (fixed) or basis points (percentage)"), mcp.Required()),
		mcp.WithNumber("min_order_amount", mcp.Description("Minimum order amount in cents")),
		mcp.WithNumber("max_uses", mcp.Description("Maximum total uses (0 = unlimited)")),
		mcp.WithString("starts_at", mcp.Description("Start date (ISO 8601)")),
		mcp.WithString("expires_at", mcp.Description("Expiration date (ISO 8601)")),
		mcp.WithBoolean("active", mcp.Description("Whether the discount is active")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Post("/api/v1/admin/discounts", req.GetArguments())
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminUpdateDiscount(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_update_discount",
		mcp.WithDescription("Update an existing discount"),
		mcp.WithString("id", mcp.Description("Discount UUID"), mcp.Required()),
		mcp.WithString("code", mcp.Description("Discount code")),
		mcp.WithString("type", mcp.Description("Discount type: percentage or fixed")),
		mcp.WithNumber("amount", mcp.Description("Amount")),
		mcp.WithNumber("min_order_amount", mcp.Description("Minimum order amount in cents")),
		mcp.WithNumber("max_uses", mcp.Description("Maximum total uses")),
		mcp.WithString("starts_at", mcp.Description("Start date (ISO 8601)")),
		mcp.WithString("expires_at", mcp.Description("Expiration date (ISO 8601)")),
		mcp.WithBoolean("active", mcp.Description("Whether the discount is active")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		id := req.GetString("id", "")
		delete(args, "id")
		data, err := client.Put("/api/v1/admin/discounts/"+id, args)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminDeleteDiscount(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_delete_discount",
		mcp.WithDescription("Delete a discount"),
		mcp.WithString("id", mcp.Description("Discount UUID"), mcp.Required()),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Delete("/api/v1/admin/discounts/" + req.GetString("id", ""))
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}
