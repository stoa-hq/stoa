package cart

import (
	"time"

	"github.com/google/uuid"
)

// CreateCartRequest is the request body for POST /cart.
type CreateCartRequest struct {
	// Currency is the ISO-4217 currency code for the cart (e.g. "USD").
	// Defaults to "USD" when omitted.
	Currency  string     `json:"currency"`
	SessionID string     `json:"session_id"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// AddItemRequest is the request body for POST /cart/:id/items.
type AddItemRequest struct {
	ProductID    uuid.UUID              `json:"product_id"`
	VariantID    *uuid.UUID             `json:"variant_id,omitempty"`
	Quantity     int                    `json:"quantity"`
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
}

// UpdateItemRequest is the request body for PUT /cart/:id/items/:itemId.
type UpdateItemRequest struct {
	Quantity int `json:"quantity"`
}

// CartItemResponse is the API representation of a single cart line item.
type CartItemResponse struct {
	ID           uuid.UUID              `json:"id"`
	CartID       uuid.UUID              `json:"cart_id"`
	ProductID    uuid.UUID              `json:"product_id"`
	VariantID    *uuid.UUID             `json:"variant_id,omitempty"`
	Quantity     int                    `json:"quantity"`
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
}

// CartResponse is the API representation of a cart.
type CartResponse struct {
	ID         uuid.UUID          `json:"id"`
	CustomerID *uuid.UUID         `json:"customer_id,omitempty"`
	SessionID  string             `json:"session_id"`
	Currency   string             `json:"currency"`
	ExpiresAt  *time.Time         `json:"expires_at,omitempty"`
	CreatedAt  time.Time          `json:"created_at"`
	Items      []CartItemResponse `json:"items"`
}

// toCartResponse converts a Cart entity into its API DTO.
func toCartResponse(c *Cart) CartResponse {
	items := make([]CartItemResponse, 0, len(c.Items))
	for _, item := range c.Items {
		items = append(items, toCartItemResponse(item))
	}

	return CartResponse{
		ID:         c.ID,
		CustomerID: c.CustomerID,
		SessionID:  c.SessionID,
		Currency:   c.Currency,
		ExpiresAt:  c.ExpiresAt,
		CreatedAt:  c.CreatedAt,
		Items:      items,
	}
}

// toCartItemResponse converts a CartItem entity into its API DTO.
func toCartItemResponse(item CartItem) CartItemResponse {
	return CartItemResponse(item)
}
