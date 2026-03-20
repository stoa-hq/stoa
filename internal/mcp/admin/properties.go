package admin

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	stoamcp "github.com/stoa-hq/stoa/internal/mcp"
)

func adminListPropertyGroups(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_list_property_groups",
		mcp.WithDescription("List all property groups (e.g. Color, Size) with their options"),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Get("/api/v1/admin/property-groups")
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminGetPropertyGroup(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_get_property_group",
		mcp.WithDescription("Get a property group with its options"),
		mcp.WithString("id", mcp.Description("Property group UUID"), mcp.Required()),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Get("/api/v1/admin/property-groups/" + req.GetString("id", ""))
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminCreatePropertyGroup(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_create_property_group",
		mcp.WithDescription("Create a new property group (e.g. Color, Size)"),
		mcp.WithString("name", mcp.Description("Property group name"), mcp.Required()),
		mcp.WithNumber("position", mcp.Description("Sort position")),
		mcp.WithObject("translations", mcp.Description("Translation object keyed by locale, e.g. {\"de-DE\": {\"name\": \"Farbe\"}}")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Post("/api/v1/admin/property-groups", transformPropertyGroupArgs(req.GetArguments()))
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminUpdatePropertyGroup(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_update_property_group",
		mcp.WithDescription("Update a property group"),
		mcp.WithString("id", mcp.Description("Property group UUID"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Property group name")),
		mcp.WithNumber("position", mcp.Description("Sort position")),
		mcp.WithObject("translations", mcp.Description("Translation object keyed by locale")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		id := req.GetString("id", "")
		delete(args, "id")
		args = transformPropertyGroupArgs(args)
		data, err := client.Put("/api/v1/admin/property-groups/"+id, args)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminDeletePropertyGroup(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_delete_property_group",
		mcp.WithDescription("Delete a property group and all its options"),
		mcp.WithString("id", mcp.Description("Property group UUID"), mcp.Required()),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Delete("/api/v1/admin/property-groups/" + req.GetString("id", ""))
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminCreatePropertyOption(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_create_property_option",
		mcp.WithDescription("Create a property option within a group (e.g. Red, L, XL)"),
		mcp.WithString("group_id", mcp.Description("Parent property group UUID"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Option name"), mcp.Required()),
		mcp.WithNumber("position", mcp.Description("Sort position")),
		mcp.WithString("color_hex", mcp.Description("Hex color code (e.g. #FF0000)")),
		mcp.WithObject("translations", mcp.Description("Translation object keyed by locale, e.g. {\"de-DE\": {\"name\": \"Rot\"}}")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		groupID := req.GetString("group_id", "")
		delete(args, "group_id")
		args = transformPropertyOptionArgs(args)
		data, err := client.Post("/api/v1/admin/property-groups/"+groupID+"/options", args)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminUpdatePropertyOption(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_update_property_option",
		mcp.WithDescription("Update a property option"),
		mcp.WithString("group_id", mcp.Description("Parent property group UUID"), mcp.Required()),
		mcp.WithString("option_id", mcp.Description("Property option UUID"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Option name")),
		mcp.WithNumber("position", mcp.Description("Sort position")),
		mcp.WithString("color_hex", mcp.Description("Hex color code (e.g. #FF0000)")),
		mcp.WithObject("translations", mcp.Description("Translation object keyed by locale")),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		groupID := req.GetString("group_id", "")
		optionID := req.GetString("option_id", "")
		delete(args, "group_id")
		delete(args, "option_id")
		args = transformPropertyOptionArgs(args)
		data, err := client.Put("/api/v1/admin/property-groups/"+groupID+"/options/"+optionID, args)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}

func adminDeletePropertyOption(client *stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("admin_delete_property_option",
		mcp.WithDescription("Delete a property option"),
		mcp.WithString("group_id", mcp.Description("Parent property group UUID"), mcp.Required()),
		mcp.WithString("option_id", mcp.Description("Property option UUID"), mcp.Required()),
	)
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Delete("/api/v1/admin/property-groups/" + req.GetString("group_id", "") + "/options/" + req.GetString("option_id", ""))
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
	return tool, handler
}
