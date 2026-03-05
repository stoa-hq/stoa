package admin

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	stoamcp "github.com/epoxx-arch/stoa/internal/mcp"
)

func adminListAuditLog(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_list_audit_log",
		mcp.WithDescription("List audit log entries showing all changes made to the shop"),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("limit", mcp.Description("Items per page")),
		mcp.WithString("sort", mcp.Description("Sort field")),
		mcp.WithString("order", mcp.Description("Sort order: asc or desc")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path := "/api/v1/admin/audit-log?" + buildQueryParams(req, "page", "limit", "sort", "order")
		data, err := client.Get(path)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}
