package store

import (
	"github.com/mark3labs/mcp-go/server"

	stoamcp "github.com/stoa-hq/stoa/internal/mcp"
)

// RegisterTools adds all store tools to the MCP server.
func RegisterTools(s *server.MCPServer, client *stoamcp.StoaClient) {
	// Products
	t, _ := listProductsTool()
	s.AddTool(t, listProductsHandler(client))

	t, _ = getProductTool()
	s.AddTool(t, getProductHandler(client))

	t, _ = searchTool()
	s.AddTool(t, searchHandler(client))

	t, _ = getCategoriesTool()
	s.AddTool(t, getCategoriesHandler(client))

	// Cart
	t, _ = createCartTool()
	s.AddTool(t, createCartHandler(client))

	t, _ = getCartTool()
	s.AddTool(t, getCartHandler(client))

	t, _ = addToCartTool()
	s.AddTool(t, addToCartHandler(client))

	t, _ = updateCartItemTool()
	s.AddTool(t, updateCartItemHandler(client))

	t, _ = removeFromCartTool()
	s.AddTool(t, removeFromCartHandler(client))

	// Checkout
	t, _ = getShippingMethodsTool()
	s.AddTool(t, getShippingMethodsHandler(client))

	t, _ = getPaymentMethodsTool()
	s.AddTool(t, getPaymentMethodsHandler(client))

	t, _ = checkoutTool()
	s.AddTool(t, checkoutHandler(client))

	// Account
	t, _ = registerTool()
	s.AddTool(t, registerHandler(client))

	t, _ = loginTool()
	s.AddTool(t, loginHandler(client))

	t, _ = getAccountTool()
	s.AddTool(t, getAccountHandler(client))

	t, _ = listOrdersTool()
	s.AddTool(t, listOrdersHandler(client))

}
