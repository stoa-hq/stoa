package admin

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	stoamcp "github.com/stoa-hq/stoa/internal/mcp"
)

func adminListWarehouses(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_list_warehouses",
		mcp.WithDescription("List all warehouses with pagination and optional active filter"),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("limit", mcp.Description("Items per page (default: 20)")),
		mcp.WithString("active", mcp.Description("Filter by active status: true or false")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path := "/api/v1/admin/warehouses?" + buildQueryParams(req, "page", "limit", "active")
		data, err := client.Get(path)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminGetWarehouse(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_get_warehouse",
		mcp.WithDescription("Get warehouse details by ID"),
		mcp.WithString("id", mcp.Description("Warehouse UUID"), mcp.Required()),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Get("/api/v1/admin/warehouses/" + req.GetString("id", ""))
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminCreateWarehouse(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_create_warehouse",
		mcp.WithDescription("Create a new warehouse"),
		mcp.WithString("name", mcp.Description("Warehouse name"), mcp.Required()),
		mcp.WithString("code", mcp.Description("Unique warehouse code"), mcp.Required()),
		mcp.WithBoolean("active", mcp.Description("Whether the warehouse is active")),
		mcp.WithNumber("priority", mcp.Description("Priority for stock deduction (lower = higher priority)")),
		mcp.WithString("address_line1", mcp.Description("Address line 1")),
		mcp.WithString("address_line2", mcp.Description("Address line 2")),
		mcp.WithString("city", mcp.Description("City")),
		mcp.WithString("state", mcp.Description("State")),
		mcp.WithString("postal_code", mcp.Description("Postal code")),
		mcp.WithString("country", mcp.Description("Country code (2 letters)")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Post("/api/v1/admin/warehouses", req.GetArguments())
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminUpdateWarehouse(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_update_warehouse",
		mcp.WithDescription("Update an existing warehouse"),
		mcp.WithString("id", mcp.Description("Warehouse UUID"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Warehouse name"), mcp.Required()),
		mcp.WithString("code", mcp.Description("Unique warehouse code"), mcp.Required()),
		mcp.WithBoolean("active", mcp.Description("Whether the warehouse is active")),
		mcp.WithNumber("priority", mcp.Description("Priority for stock deduction")),
		mcp.WithString("address_line1", mcp.Description("Address line 1")),
		mcp.WithString("address_line2", mcp.Description("Address line 2")),
		mcp.WithString("city", mcp.Description("City")),
		mcp.WithString("state", mcp.Description("State")),
		mcp.WithString("postal_code", mcp.Description("Postal code")),
		mcp.WithString("country", mcp.Description("Country code (2 letters)")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		id := req.GetString("id", "")
		delete(args, "id")
		data, err := client.Put("/api/v1/admin/warehouses/"+id, args)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminDeleteWarehouse(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_delete_warehouse",
		mcp.WithDescription("Delete a warehouse permanently"),
		mcp.WithString("id", mcp.Description("Warehouse UUID"), mcp.Required()),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Delete("/api/v1/admin/warehouses/" + req.GetString("id", ""))
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminGetWarehouseStock(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_get_warehouse_stock",
		mcp.WithDescription("Get all stock entries for a warehouse, including product SKU and name"),
		mcp.WithString("id", mcp.Description("Warehouse UUID"), mcp.Required()),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Get("/api/v1/admin/warehouses/" + req.GetString("id", "") + "/stock")
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminSetWarehouseStock(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_set_warehouse_stock",
		mcp.WithDescription("Set stock quantities for products at a warehouse (upsert). Each item needs product_id, quantity, and optionally variant_id and reference."),
		mcp.WithString("id", mcp.Description("Warehouse UUID"), mcp.Required()),
		mcp.WithArray("items", mcp.Description("Array of stock items: [{product_id, variant_id?, quantity, reference?}]"), mcp.Required()),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		id := req.GetString("id", "")
		delete(args, "id")
		data, err := client.Put("/api/v1/admin/warehouses/"+id+"/stock", args)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminGetProductStock(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_get_product_stock",
		mcp.WithDescription("Get stock entries for a product across all warehouses"),
		mcp.WithString("product_id", mcp.Description("Product UUID"), mcp.Required()),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Get("/api/v1/admin/products/" + req.GetString("product_id", "") + "/stock")
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}
