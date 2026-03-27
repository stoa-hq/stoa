package admin

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	stoamcp "github.com/stoa-hq/stoa/internal/mcp"
)

func adminListAttributes(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_list_attributes",
		mcp.WithDescription("List all attribute definitions (e.g. Brand, Material, Weight) with their options"),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Get("/api/v1/admin/attributes")
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminGetAttribute(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_get_attribute",
		mcp.WithDescription("Get an attribute definition with its options"),
		mcp.WithString("id", mcp.Description("Attribute UUID"), mcp.Required()),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Get("/api/v1/admin/attributes/" + req.GetString("id", ""))
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminCreateAttribute(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_create_attribute",
		mcp.WithDescription("Create a new attribute definition (e.g. Brand, Material, Weight)"),
		mcp.WithString("identifier", mcp.Description("Unique identifier slug (lowercase, hyphens, underscores, e.g. 'brand' or 'weight')"), mcp.Required()),
		mcp.WithString("type", mcp.Description("Attribute type: text, number, select, multi_select, boolean"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Attribute name"), mcp.Required()),
		mcp.WithString("unit", mcp.Description("Unit of measurement (e.g. 'g', 'mm', 'kg')")),
		mcp.WithNumber("position", mcp.Description("Sort position")),
		mcp.WithBoolean("filterable", mcp.Description("Whether this attribute can be used for filtering")),
		mcp.WithBoolean("required", mcp.Description("Whether this attribute is required on products")),
		mcp.WithString("description", mcp.Description("Attribute description")),
		mcp.WithObject("translations", mcp.Description("Translation object keyed by locale, e.g. {\"de-DE\": {\"name\": \"Marke\", \"description\": \"...\"}}")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Post("/api/v1/admin/attributes", transformAttributeArgs(req.GetArguments()))
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminUpdateAttribute(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_update_attribute",
		mcp.WithDescription("Update an attribute definition"),
		mcp.WithString("id", mcp.Description("Attribute UUID"), mcp.Required()),
		mcp.WithString("identifier", mcp.Description("Unique identifier slug")),
		mcp.WithString("type", mcp.Description("Attribute type: text, number, select, multi_select, boolean")),
		mcp.WithString("name", mcp.Description("Attribute name")),
		mcp.WithString("unit", mcp.Description("Unit of measurement")),
		mcp.WithNumber("position", mcp.Description("Sort position")),
		mcp.WithBoolean("filterable", mcp.Description("Whether this attribute can be used for filtering")),
		mcp.WithBoolean("required", mcp.Description("Whether this attribute is required")),
		mcp.WithString("description", mcp.Description("Attribute description")),
		mcp.WithObject("translations", mcp.Description("Translation object keyed by locale")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		id := req.GetString("id", "")
		delete(args, "id")
		args = transformAttributeArgs(args)
		data, err := client.Put("/api/v1/admin/attributes/"+id, args)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminDeleteAttribute(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_delete_attribute",
		mcp.WithDescription("Delete an attribute and all its values"),
		mcp.WithString("id", mcp.Description("Attribute UUID"), mcp.Required()),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Delete("/api/v1/admin/attributes/" + req.GetString("id", ""))
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminCreateAttributeOption(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_create_attribute_option",
		mcp.WithDescription("Create an option for a select/multi_select attribute (e.g. 'Leather' for Material)"),
		mcp.WithString("attribute_id", mcp.Description("Parent attribute UUID"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Option name"), mcp.Required()),
		mcp.WithNumber("position", mcp.Description("Sort position")),
		mcp.WithObject("translations", mcp.Description("Translation object keyed by locale, e.g. {\"de-DE\": {\"name\": \"Leder\"}}")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		attrID := req.GetString("attribute_id", "")
		delete(args, "attribute_id")
		args = transformAttributeOptionArgs(args)
		data, err := client.Post("/api/v1/admin/attributes/"+attrID+"/options", args)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminUpdateAttributeOption(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_update_attribute_option",
		mcp.WithDescription("Update an attribute option"),
		mcp.WithString("attribute_id", mcp.Description("Parent attribute UUID"), mcp.Required()),
		mcp.WithString("option_id", mcp.Description("Attribute option UUID"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Option name")),
		mcp.WithNumber("position", mcp.Description("Sort position")),
		mcp.WithObject("translations", mcp.Description("Translation object keyed by locale")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		attrID := req.GetString("attribute_id", "")
		optionID := req.GetString("option_id", "")
		delete(args, "attribute_id")
		delete(args, "option_id")
		args = transformAttributeOptionArgs(args)
		data, err := client.Put("/api/v1/admin/attributes/"+attrID+"/options/"+optionID, args)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminDeleteAttributeOption(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_delete_attribute_option",
		mcp.WithDescription("Delete an attribute option"),
		mcp.WithString("attribute_id", mcp.Description("Parent attribute UUID"), mcp.Required()),
		mcp.WithString("option_id", mcp.Description("Attribute option UUID"), mcp.Required()),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Delete("/api/v1/admin/attributes/" + req.GetString("attribute_id", "") + "/options/" + req.GetString("option_id", ""))
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminSetProductAttributes(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_set_product_attributes",
		mcp.WithDescription("Set attribute values on a product. Each attribute value needs attribute_id and value_text/value_numeric/value_boolean/option_id/option_ids depending on type."),
		mcp.WithString("product_id", mcp.Description("Product UUID"), mcp.Required()),
		mcp.WithObject("attributes", mcp.Description("Array of attribute values: [{\"attribute_id\": \"...\", \"value_text\": \"adidas\"}, ...]"), mcp.Required()),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		productID := req.GetString("product_id", "")
		delete(args, "product_id")
		data, err := client.Put("/api/v1/admin/products/"+productID+"/attributes", args)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminSetVariantAttributes(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_set_variant_attributes",
		mcp.WithDescription("Set attribute values on a product variant. Same format as product attributes."),
		mcp.WithString("product_id", mcp.Description("Product UUID"), mcp.Required()),
		mcp.WithString("variant_id", mcp.Description("Variant UUID"), mcp.Required()),
		mcp.WithObject("attributes", mcp.Description("Array of attribute values"), mcp.Required()),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		productID := req.GetString("product_id", "")
		variantID := req.GetString("variant_id", "")
		delete(args, "product_id")
		delete(args, "variant_id")
		data, err := client.Put("/api/v1/admin/products/"+productID+"/variants/"+variantID+"/attributes", args)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}
