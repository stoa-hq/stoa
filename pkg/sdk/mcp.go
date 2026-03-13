package sdk

// StoreAPIClient is the HTTP client passed to RegisterStoreMCPTools.
// The concrete type *github.com/stoa-hq/stoa/internal/mcp.StoaClient satisfies
// this interface, so no import of internal/mcp is needed in plugin code.
type StoreAPIClient interface {
	Get(path string) ([]byte, error)
	Post(path string, body interface{}) ([]byte, error)
}

// MCPStorePlugin is an optional interface plugins may implement to register
// additional tools on the Store MCP server.
//
// RegisterStoreMCPTools is called once during Store MCP server startup, after
// the built-in tools are registered. The server parameter satisfies the
// toolAdder interface (AddTool method) — use an interface assertion, not a
// concrete type assertion, for forward compatibility:
//
//	import (
//	    "github.com/mark3labs/mcp-go/mcp"
//	    "github.com/mark3labs/mcp-go/server"
//	)
//
//	type toolAdder interface { AddTool(mcp.Tool, server.ToolHandlerFunc) }
//
//	func (p *Plugin) RegisterStoreMCPTools(srv any, client sdk.StoreAPIClient) {
//	    s := srv.(toolAdder)
//	    tool := mcp.NewTool("store_myplugin_action", ...)
//	    s.AddTool(tool, myHandler(client))
//	}
//
// Tool names MUST use the prefix "store_{pluginName}_" to prevent collisions
// with built-in tools or other plugins.
type MCPStorePlugin interface {
	Plugin
	RegisterStoreMCPTools(server any, client StoreAPIClient)
}
