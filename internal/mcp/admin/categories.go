package admin

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	stoamcp "github.com/epoxx-arch/stoa/internal/mcp"
)

func adminListCategories(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_list_categories",
		mcp.WithDescription("List all categories"),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("limit", mcp.Description("Items per page")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path := "/api/v1/admin/categories?" + buildQueryParams(req, "page", "limit")
		data, err := client.Get(path)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminGetCategory(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_get_category",
		mcp.WithDescription("Get category details"),
		mcp.WithString("id", mcp.Description("Category UUID"), mcp.Required()),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Get("/api/v1/admin/categories/" + req.GetString("id", ""))
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminCreateCategory(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_create_category",
		mcp.WithDescription("Create a new category"),
		mcp.WithString("name", mcp.Description("Category name"), mcp.Required()),
		mcp.WithString("slug", mcp.Description("URL-friendly slug"), mcp.Required()),
		mcp.WithString("description", mcp.Description("Category description")),
		mcp.WithString("parent_id", mcp.Description("Parent category UUID for subcategories")),
		mcp.WithBoolean("active", mcp.Description("Whether the category is active")),
		mcp.WithObject("translations", mcp.Description("Translation object keyed by locale")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Post("/api/v1/admin/categories", req.GetArguments())
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminUpdateCategory(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_update_category",
		mcp.WithDescription("Update a category"),
		mcp.WithString("id", mcp.Description("Category UUID"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Category name")),
		mcp.WithString("slug", mcp.Description("URL-friendly slug")),
		mcp.WithString("description", mcp.Description("Category description")),
		mcp.WithString("parent_id", mcp.Description("Parent category UUID")),
		mcp.WithBoolean("active", mcp.Description("Whether the category is active")),
		mcp.WithObject("translations", mcp.Description("Translation object")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		id := req.GetString("id", "")
		delete(args, "id")
		data, err := client.Put("/api/v1/admin/categories/"+id, args)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}
