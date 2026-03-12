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
// the built-in tools are registered. The server parameter is
// *github.com/mark3labs/mcp-go/server.MCPServer — passed as any to avoid an
// import of mcp-go in pkg/sdk.
//
// Example:
//
//	import "github.com/mark3labs/mcp-go/server"
//
//	func (p *Plugin) RegisterStoreMCPTools(srv any, client sdk.StoreAPIClient) {
//	    s := srv.(*server.MCPServer)
//	    tool := mcp.NewTool("my_tool", ...)
//	    s.AddTool(tool, myHandler(client))
//	}
type MCPStorePlugin interface {
	Plugin
	RegisterStoreMCPTools(server any, client StoreAPIClient)
}
