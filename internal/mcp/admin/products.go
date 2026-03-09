package admin

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	stoamcp "github.com/stoa-hq/stoa/internal/mcp"
)

func adminListProducts(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_list_products",
		mcp.WithDescription("List all products with pagination, filtering, and search"),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("limit", mcp.Description("Items per page (default: 25)")),
		mcp.WithString("search", mcp.Description("Search term")),
		mcp.WithString("sort", mcp.Description("Sort field")),
		mcp.WithString("order", mcp.Description("Sort order: asc or desc")),
		mcp.WithString("filter[active]", mcp.Description("Filter by active status: true or false")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path := "/api/v1/admin/products?" + buildQueryParams(req, "page", "limit", "search", "sort", "order", "filter[active]")
		data, err := client.Get(path)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminGetProduct(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_get_product",
		mcp.WithDescription("Get detailed product information including variants and translations"),
		mcp.WithString("id", mcp.Description("Product UUID"), mcp.Required()),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Get("/api/v1/admin/products/" + req.GetString("id", ""))
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminCreateProduct(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_create_product",
		mcp.WithDescription("Create a new product. Prices are in cents (e.g. 1999 = 19.99 EUR)."),
		mcp.WithString("name", mcp.Description("Product name"), mcp.Required()),
		mcp.WithString("slug", mcp.Description("URL-friendly slug"), mcp.Required()),
		mcp.WithString("description", mcp.Description("Product description")),
		mcp.WithNumber("price", mcp.Description("Price in cents"), mcp.Required()),
		mcp.WithString("sku", mcp.Description("Stock keeping unit")),
		mcp.WithNumber("stock", mcp.Description("Stock quantity")),
		mcp.WithBoolean("active", mcp.Description("Whether the product is active")),
		mcp.WithString("tax_rule_id", mcp.Description("Tax rule UUID")),
		mcp.WithArray("category_ids", mcp.Description("Category UUIDs")),
		mcp.WithArray("tag_ids", mcp.Description("Tag UUIDs")),
		mcp.WithObject("translations", mcp.Description("Translation object keyed by locale, e.g. {\"de\": {\"name\": \"...\", \"description\": \"...\"}}")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Post("/api/v1/admin/products", req.GetArguments())
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminUpdateProduct(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_update_product",
		mcp.WithDescription("Update an existing product. Only pass fields you want to change."),
		mcp.WithString("id", mcp.Description("Product UUID"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Product name")),
		mcp.WithString("slug", mcp.Description("URL-friendly slug")),
		mcp.WithString("description", mcp.Description("Product description")),
		mcp.WithNumber("price", mcp.Description("Price in cents")),
		mcp.WithString("sku", mcp.Description("Stock keeping unit")),
		mcp.WithNumber("stock", mcp.Description("Stock quantity")),
		mcp.WithBoolean("active", mcp.Description("Whether the product is active")),
		mcp.WithString("tax_rule_id", mcp.Description("Tax rule UUID")),
		mcp.WithArray("category_ids", mcp.Description("Category UUIDs")),
		mcp.WithArray("tag_ids", mcp.Description("Tag UUIDs")),
		mcp.WithObject("translations", mcp.Description("Translation object")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		id := req.GetString("id", "")
		delete(args, "id")
		data, err := client.Put("/api/v1/admin/products/"+id, args)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminDeleteProduct(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_delete_product",
		mcp.WithDescription("Delete a product permanently"),
		mcp.WithString("id", mcp.Description("Product UUID"), mcp.Required()),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Delete("/api/v1/admin/products/" + req.GetString("id", ""))
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminCreateVariant(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_create_variant",
		mcp.WithDescription("Create a product variant (e.g. size/color combination)"),
		mcp.WithString("product_id", mcp.Description("Parent product UUID"), mcp.Required()),
		mcp.WithString("sku", mcp.Description("Variant SKU"), mcp.Required()),
		mcp.WithNumber("price", mcp.Description("Variant price in cents"), mcp.Required()),
		mcp.WithNumber("stock", mcp.Description("Stock quantity")),
		mcp.WithBoolean("active", mcp.Description("Whether the variant is active")),
		mcp.WithArray("option_ids", mcp.Description("Property option UUIDs for this variant")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		productID := req.GetString("product_id", "")
		delete(args, "product_id")
		data, err := client.Post("/api/v1/admin/products/"+productID+"/variants", args)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminUpdateVariant(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_update_variant",
		mcp.WithDescription("Update a product variant"),
		mcp.WithString("product_id", mcp.Description("Parent product UUID"), mcp.Required()),
		mcp.WithString("variant_id", mcp.Description("Variant UUID"), mcp.Required()),
		mcp.WithString("sku", mcp.Description("Variant SKU")),
		mcp.WithNumber("price", mcp.Description("Variant price in cents")),
		mcp.WithNumber("stock", mcp.Description("Stock quantity")),
		mcp.WithBoolean("active", mcp.Description("Whether the variant is active")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		productID := req.GetString("product_id", "")
		variantID := req.GetString("variant_id", "")
		delete(args, "product_id")
		delete(args, "variant_id")
		data, err := client.Put("/api/v1/admin/products/"+productID+"/variants/"+variantID, args)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminDeleteVariant(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_delete_variant",
		mcp.WithDescription("Delete a product variant"),
		mcp.WithString("product_id", mcp.Description("Parent product UUID"), mcp.Required()),
		mcp.WithString("variant_id", mcp.Description("Variant UUID"), mcp.Required()),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Delete("/api/v1/admin/products/" + req.GetString("product_id", "") + "/variants/" + req.GetString("variant_id", ""))
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}
