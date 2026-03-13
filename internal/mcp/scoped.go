package mcp

import (
	"fmt"
	"strings"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ScopedMCPServer wraps an MCPServer and enforces that plugin tools use a
// name prefix of "store_{pluginName}_". This prevents plugins from
// overwriting built-in tools or other plugins' tools.
type ScopedMCPServer struct {
	srv    *server.MCPServer
	prefix string
}

// NewScopedMCPServer creates a scoped wrapper for the given plugin.
func NewScopedMCPServer(srv *server.MCPServer, pluginName string) *ScopedMCPServer {
	return &ScopedMCPServer{
		srv:    srv,
		prefix: "store_" + pluginName + "_",
	}
}

// AddTool validates the tool name prefix and delegates to the underlying server.
func (s *ScopedMCPServer) AddTool(tool mcplib.Tool, handler server.ToolHandlerFunc) {
	if !strings.HasPrefix(tool.Name, s.prefix) {
		panic(fmt.Sprintf("plugin tool %q must use prefix %q", tool.Name, s.prefix))
	}
	s.srv.AddTool(tool, handler)
}
