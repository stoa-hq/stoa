package admin

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	stoamcp "github.com/stoa-hq/stoa/internal/mcp"
)

func adminListOrders(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_list_orders",
		mcp.WithDescription("List all orders with pagination and filtering"),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("limit", mcp.Description("Items per page")),
		mcp.WithString("sort", mcp.Description("Sort field")),
		mcp.WithString("order", mcp.Description("Sort order: asc or desc")),
		mcp.WithString("filter[status]", mcp.Description("Filter by status: pending, processing, shipped, delivered, cancelled")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path := "/api/v1/admin/orders?" + buildQueryParams(req, "page", "limit", "sort", "order", "filter[status]")
		data, err := client.Get(path)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminGetOrder(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_get_order",
		mcp.WithDescription("Get detailed order information including line items and addresses"),
		mcp.WithString("id", mcp.Description("Order UUID"), mcp.Required()),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Get("/api/v1/admin/orders/" + req.GetString("id", ""))
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminUpdateOrderStatus(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_update_order_status",
		mcp.WithDescription("Update an order's status"),
		mcp.WithString("id", mcp.Description("Order UUID"), mcp.Required()),
		mcp.WithString("status", mcp.Description("New status: pending, processing, shipped, delivered, cancelled"), mcp.Required()),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		body := map[string]interface{}{
			"status": req.GetString("status", ""),
		}
		data, err := client.Put("/api/v1/admin/orders/"+req.GetString("id", "")+"/status", body)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}
