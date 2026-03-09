package store

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	stoamcp "github.com/stoa-hq/stoa/internal/mcp"
)

func registerTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("store_register",
		mcp.WithDescription("Register a new customer account"),
		mcp.WithString("email", mcp.Description("Customer email address"), mcp.Required()),
		mcp.WithString("password", mcp.Description("Password (min 8 characters)"), mcp.Required()),
		mcp.WithString("first_name", mcp.Description("First name"), mcp.Required()),
		mcp.WithString("last_name", mcp.Description("Last name"), mcp.Required()),
	)
	return tool, nil
}

func registerHandler(client *stoamcp.StoaClient) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		body := map[string]interface{}{
			"email":      req.GetString("email", ""),
			"password":   req.GetString("password", ""),
			"first_name": req.GetString("first_name", ""),
			"last_name":  req.GetString("last_name", ""),
		}

		data, err := client.Post("/api/v1/store/register", body)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
}

func loginTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("store_login",
		mcp.WithDescription("Login with email and password. Returns access and refresh tokens."),
		mcp.WithString("email", mcp.Description("Email address"), mcp.Required()),
		mcp.WithString("password", mcp.Description("Password"), mcp.Required()),
	)
	return tool, nil
}

func loginHandler(client *stoamcp.StoaClient) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		body := map[string]interface{}{
			"email":    req.GetString("email", ""),
			"password": req.GetString("password", ""),
		}

		data, err := client.Post("/api/v1/auth/login", body)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
}

func getAccountTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("store_get_account",
		mcp.WithDescription("Get the current customer's account details. Requires authentication."),
	)
	return tool, nil
}

func getAccountHandler(client *stoamcp.StoaClient) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Get("/api/v1/store/account")
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
}

func listOrdersTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("store_list_orders",
		mcp.WithDescription("List the current customer's orders. Requires authentication."),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("limit", mcp.Description("Items per page")),
	)
	return tool, nil
}

func listOrdersHandler(client *stoamcp.StoaClient) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path := "/api/v1/store/account/orders?"
		path += buildQueryParams(req, "page", "limit")

		data, err := client.Get(path)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
}
