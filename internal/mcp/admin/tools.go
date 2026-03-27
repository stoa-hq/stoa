package admin

import (
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	stoamcp "github.com/stoa-hq/stoa/internal/mcp"
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

	// Property Groups & Options (8)
	register(adminListPropertyGroups)
	register(adminGetPropertyGroup)
	register(adminCreatePropertyGroup)
	register(adminUpdatePropertyGroup)
	register(adminDeletePropertyGroup)
	register(adminCreatePropertyOption)
	register(adminUpdatePropertyOption)
	register(adminDeletePropertyOption)

	// Attributes (10)
	register(adminListAttributes)
	register(adminGetAttribute)
	register(adminCreateAttribute)
	register(adminUpdateAttribute)
	register(adminDeleteAttribute)
	register(adminCreateAttributeOption)
	register(adminUpdateAttributeOption)
	register(adminDeleteAttributeOption)
	register(adminSetProductAttributes)
	register(adminSetVariantAttributes)

	// Warehouses (8)
	register(adminListWarehouses)
	register(adminGetWarehouse)
	register(adminCreateWarehouse)
	register(adminUpdateWarehouse)
	register(adminDeleteWarehouse)
	register(adminGetWarehouseStock)
	register(adminSetWarehouseStock)
	register(adminGetProductStock)
}

// buildQueryParams builds a URL query string from request arguments.
// Keys with a "filter_" prefix are mapped to "filter[...]" query parameters.
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
			paramKey := key
			if strings.HasPrefix(key, "filter_") {
				paramKey = "filter[" + strings.TrimPrefix(key, "filter_") + "]"
			}
			query += paramKey + "=" + val
		}
	}
	return query
}
