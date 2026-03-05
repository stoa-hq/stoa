package admin

import (
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	stoamcp "github.com/epoxx-arch/stoa/internal/mcp"
)

// RegisterTools adds all admin tools to the MCP server.
func RegisterTools(s *server.MCPServer, client *stoamcp.StoaClient) {
	register := func(fn func(*stoamcp.StoaClient) (mcp.Tool, server.ToolHandlerFunc)) {
		tool, handler := fn(client)
		s.AddTool(tool, handler)
	}

	// Products (8)
	register(adminListProducts)
	register(adminGetProduct)
	register(adminCreateProduct)
	register(adminUpdateProduct)
	register(adminDeleteProduct)
	register(adminCreateVariant)
	register(adminUpdateVariant)
	register(adminDeleteVariant)

	// Orders (3)
	register(adminListOrders)
	register(adminGetOrder)
	register(adminUpdateOrderStatus)

	// Discounts (5)
	register(adminListDiscounts)
	register(adminGetDiscount)
	register(adminCreateDiscount)
	register(adminUpdateDiscount)
	register(adminDeleteDiscount)

	// Customers (4)
	register(adminListCustomers)
	register(adminGetCustomer)
	register(adminUpdateCustomer)
	register(adminDeleteCustomer)

	// Categories (4)
	register(adminListCategories)
	register(adminGetCategory)
	register(adminCreateCategory)
	register(adminUpdateCategory)

	// Tags (3)
	register(adminListTags)
	register(adminCreateTag)
	register(adminDeleteTag)

	// Media (2)
	register(adminListMedia)
	register(adminDeleteMedia)

	// Shipping / Tax / Payment (3)
	register(adminListShippingMethods)
	register(adminListTaxRules)
	register(adminListPaymentMethods)

	// Audit (1)
	register(adminListAuditLog)
}

// buildQueryParams builds a URL query string from request arguments.
func buildQueryParams(req mcp.CallToolRequest, keys ...string) string {
	query := ""
	for _, key := range keys {
		val := req.GetString(key, "")
		if val == "" {
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
