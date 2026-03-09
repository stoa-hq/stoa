package store

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	stoamcp "github.com/stoa-hq/stoa/internal/mcp"
)

func listProductsTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("store_list_products",
		mcp.WithDescription("List products in the store with optional filtering and pagination"),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("limit", mcp.Description("Items per page (default: 25, max: 100)")),
		mcp.WithString("search", mcp.Description("Search term for product name/description")),
		mcp.WithString("category_id", mcp.Description("Filter by category UUID")),
		mcp.WithString("sort", mcp.Description("Sort field: created_at, name, price")),
		mcp.WithString("order", mcp.Description("Sort order: asc or desc")),
	)
	return tool, nil
}

func listProductsHandler(client *stoamcp.StoaClient) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path := "/api/v1/store/products?"
		path += buildQueryParams(req, "page", "limit", "search", "category_id", "sort", "order")

		data, err := client.Get(path)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
}

func getProductTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("store_get_product",
		mcp.WithDescription("Get detailed information about a product by its slug or ID"),
		mcp.WithString("slug", mcp.Description("Product slug (URL-friendly name)")),
		mcp.WithString("id", mcp.Description("Product UUID")),
	)
	return tool, nil
}

func getProductHandler(client *stoamcp.StoaClient) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := req.GetString("id", "")
		slug := req.GetString("slug", "")

		var path string
		switch {
		case id != "":
			path = "/api/v1/store/products/id/" + id
		case slug != "":
			path = "/api/v1/store/products/" + slug
		default:
			return mcp.NewToolResultError("either 'slug' or 'id' is required"), nil
		}

		data, err := client.Get(path)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
}

func searchTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("store_search",
		mcp.WithDescription("Full-text search across products and categories"),
		mcp.WithString("q", mcp.Description("Search query"), mcp.Required()),
		mcp.WithString("locale", mcp.Description("Locale for search (e.g. de, en)")),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("limit", mcp.Description("Results per page")),
		mcp.WithString("type", mcp.Description("Filter by type: product, category")),
	)
	return tool, nil
}

func searchHandler(client *stoamcp.StoaClient) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path := "/api/v1/store/search?"
		path += buildQueryParams(req, "q", "locale", "page", "limit", "type")

		data, err := client.Get(path)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
}

func getCategoriesTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("store_get_categories",
		mcp.WithDescription("Get the full category tree of the store"),
	)
	return tool, nil
}

func getCategoriesHandler(client *stoamcp.StoaClient) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Get("/api/v1/store/categories/tree")
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
}

// buildQueryParams builds a URL query string from request arguments.
func buildQueryParams(req mcp.CallToolRequest, keys ...string) string {
	query := ""
	for _, key := range keys {
		val := req.GetString(key, "")
		if val == "" {
			// Check for number params
			if n := req.GetInt(key, 0); n != 0 {
				val = fmt.Sprintf("%d", n)
			}
		}
		if val != "" {
			if query != "" {
				query += "&"
			}
			query += key + "=" + val
		}
	}
	return query
}
