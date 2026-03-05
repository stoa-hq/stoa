package admin

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	stoamcp "github.com/epoxx-arch/stoa/internal/mcp"
)

func adminListTags(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_list_tags",
		mcp.WithDescription("List all product tags"),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("limit", mcp.Description("Items per page")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path := "/api/v1/admin/tags?" + buildQueryParams(req, "page", "limit")
		data, err := client.Get(path)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminCreateTag(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_create_tag",
		mcp.WithDescription("Create a new tag"),
		mcp.WithString("name", mcp.Description("Tag name"), mcp.Required()),
		mcp.WithString("slug", mcp.Description("URL-friendly slug"), mcp.Required()),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Post("/api/v1/admin/tags", req.GetArguments())
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminDeleteTag(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_delete_tag",
		mcp.WithDescription("Delete a tag"),
		mcp.WithString("id", mcp.Description("Tag UUID"), mcp.Required()),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Delete("/api/v1/admin/tags/" + req.GetString("id", ""))
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}
