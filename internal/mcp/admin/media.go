package admin

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	stoamcp "github.com/epoxx-arch/stoa/internal/mcp"
)

func adminListMedia(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_list_media",
		mcp.WithDescription("List all uploaded media files"),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("limit", mcp.Description("Items per page")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path := "/api/v1/admin/media?" + buildQueryParams(req, "page", "limit")
		data, err := client.Get(path)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminDeleteMedia(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_delete_media",
		mcp.WithDescription("Delete a media file"),
		mcp.WithString("id", mcp.Description("Media UUID"), mcp.Required()),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Delete("/api/v1/admin/media/" + req.GetString("id", ""))
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}
