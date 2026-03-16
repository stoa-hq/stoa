package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	stoamcp "github.com/stoa-hq/stoa/internal/mcp"
)

func getShippingMethodsTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("store_get_shipping_methods",
		mcp.WithDescription("List available shipping methods with prices"),
	)
	return tool, nil
}

func getShippingMethodsHandler(client *stoamcp.StoaClient) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Get("/api/v1/store/shipping-methods")
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
}

func getPaymentMethodsTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("store_get_payment_methods",
		mcp.WithDescription("List available payment methods"),
	)
	return tool, nil
}

func getPaymentMethodsHandler(client *stoamcp.StoaClient) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.Get("/api/v1/store/payment-methods")
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
}

func checkoutTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("store_checkout",
		mcp.WithDescription("Complete a checkout. Resolves the cart automatically: fetches cart items and product details, then creates an order."),
		mcp.WithString("cart_id", mcp.Description("Cart UUID"), mcp.Required()),
		mcp.WithString("shipping_method_id", mcp.Description("Shipping method UUID"), mcp.Required()),
		mcp.WithString("payment_method_id", mcp.Description("Payment method UUID"), mcp.Required()),
		mcp.WithObject("shipping_address", mcp.Description("Shipping address object with: first_name, last_name, street, city, zip, country"), mcp.Required()),
		mcp.WithObject("billing_address", mcp.Description("Billing address object (same fields as shipping_address). If omitted, shipping address is used.")),
		mcp.WithString("email", mcp.Description("Guest email address (required for guest checkout)")),
		mcp.WithString("payment_reference", mcp.Description("Payment reference (e.g. Stripe PaymentIntent ID). Required when the selected payment method has a provider.")),
	)
	return tool, nil
}

// cartResponse mirrors the Cart API response for JSON unmarshalling.
type cartResponse struct {
	ID       string             `json:"id"`
	Currency string             `json:"currency"`
	Items    []cartItemResponse `json:"items"`
}

type cartItemResponse struct {
	ProductID string  `json:"product_id"`
	VariantID *string `json:"variant_id,omitempty"`
	Quantity  int     `json:"quantity"`
}

// productResponse mirrors the Product API response fields needed for checkout.
type productResponse struct {
	ID           string                   `json:"id"`
	SKU          string                   `json:"sku"`
	PriceNet     int                      `json:"price_net"`
	PriceGross   int                      `json:"price_gross"`
	Currency     string                   `json:"currency"`
	Translations []productTranslationResp `json:"translations"`
}

type productTranslationResp struct {
	Locale string `json:"locale"`
	Name   string `json:"name"`
}

func checkoutHandler(client *stoamcp.StoaClient) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cartID := req.GetString("cart_id", "")
		if cartID == "" {
			return mcp.NewToolResultError("cart_id is required"), nil
		}

		// 1. Fetch cart
		cartData, err := client.Get("/api/v1/store/cart/" + cartID)
		if err != nil {
			return stoamcp.ErrorResult(fmt.Errorf("fetching cart: %w", err)), nil
		}

		cartResp, err := stoamcp.ParseResponse(cartData)
		if err != nil {
			return stoamcp.ErrorResult(fmt.Errorf("parsing cart response: %w", err)), nil
		}

		var cart cartResponse
		if err := json.Unmarshal(cartResp.Data, &cart); err != nil {
			return stoamcp.ErrorResult(fmt.Errorf("unmarshalling cart: %w", err)), nil
		}

		if len(cart.Items) == 0 {
			return mcp.NewToolResultError("cart is empty"), nil
		}

		// 2. For each cart item, fetch product details
		checkoutItems := make([]map[string]interface{}, 0, len(cart.Items))
		for _, item := range cart.Items {
			prodData, err := client.Get("/api/v1/store/products/id/" + item.ProductID)
			if err != nil {
				return stoamcp.ErrorResult(fmt.Errorf("fetching product %s: %w", item.ProductID, err)), nil
			}

			prodResp, err := stoamcp.ParseResponse(prodData)
			if err != nil {
				return stoamcp.ErrorResult(fmt.Errorf("parsing product response: %w", err)), nil
			}

			var product productResponse
			if err := json.Unmarshal(prodResp.Data, &product); err != nil {
				return stoamcp.ErrorResult(fmt.Errorf("unmarshalling product: %w", err)), nil
			}

			name := product.SKU // fallback
			if len(product.Translations) > 0 {
				name = product.Translations[0].Name
			}

			ci := map[string]interface{}{
				"product_id":       item.ProductID,
				"sku":              product.SKU,
				"name":             name,
				"quantity":         item.Quantity,
				"unit_price_net":   product.PriceNet,
				"unit_price_gross": product.PriceGross,
				"tax_rate":         0,
			}
			if item.VariantID != nil {
				ci["variant_id"] = *item.VariantID
			}

			checkoutItems = append(checkoutItems, ci)
		}

		// 3. Build checkout request
		args := req.GetArguments()
		shippingAddr := args["shipping_address"]
		billingAddr := shippingAddr // default: same as shipping
		if ba, ok := args["billing_address"]; ok && ba != nil {
			billingAddr = ba
		}

		body := map[string]interface{}{
			"currency":           cart.Currency,
			"shipping_method_id": args["shipping_method_id"],
			"payment_method_id":  args["payment_method_id"],
			"shipping_address":   shippingAddr,
			"billing_address":    billingAddr,
			"items":              checkoutItems,
		}

		if email := req.GetString("email", ""); email != "" {
			body["email"] = email
		}
		if paymentRef := req.GetString("payment_reference", ""); paymentRef != "" {
			body["payment_reference"] = paymentRef
		}

		// 4. Submit checkout
		data, err := client.Post("/api/v1/store/checkout", body)
		if err != nil {
			return stoamcp.ErrorResult(err), nil
		}
		return stoamcp.FormatResponse(data)
	}
}
