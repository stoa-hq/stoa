package cart

import (
	"time"

	"github.com/google/uuid"
)

// Cart represents a shopping cart for either a guest session or a logged-in customer.
type Cart struct {
	ID         uuid.UUID  `json:"id"`
	CustomerID *uuid.UUID `json:"customer_id,omitempty"`
	SessionID  string     `json:"session_id"`
	Currency   string     `json:"currency"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`

	// Items contains the line items belonging to this cart.
	Items []CartItem `json:"items,omitempty"`
}

// CartItem represents a single line item within a cart.
type CartItem struct {
	ID           uuid.UUID              `json:"id"`
	CartID       uuid.UUID              `json:"cart_id"`
	ProductID    uuid.UUID              `json:"product_id"`
	VariantID    *uuid.UUID             `json:"variant_id,omitempty"`
	Quantity     int                    `json:"quantity"`
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
}
