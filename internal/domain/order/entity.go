package order

import (
	"time"

	"github.com/google/uuid"
)

// Order status constants define the lifecycle of an order.
const (
	StatusPending    = "pending"
	StatusConfirmed  = "confirmed"
	StatusProcessing = "processing"
	StatusShipped    = "shipped"
	StatusDelivered  = "delivered"
	StatusCancelled  = "cancelled"
	StatusRefunded   = "refunded"
)

// validTransitions maps each status to the set of statuses it may transition to.
var validTransitions = map[string][]string{
	StatusPending:    {StatusConfirmed, StatusCancelled},
	StatusConfirmed:  {StatusProcessing, StatusCancelled},
	StatusProcessing: {StatusShipped, StatusCancelled},
	StatusShipped:    {StatusDelivered},
	StatusDelivered:  {StatusRefunded},
	StatusCancelled:  {},
	StatusRefunded:   {},
}

// Order is the central aggregate for the order domain.
type Order struct {
	ID              uuid.UUID              `json:"id"`
	OrderNumber     string                 `json:"order_number"`
	CustomerID      *uuid.UUID             `json:"customer_id,omitempty"`
	Status          string                 `json:"status"`
	Currency        string                 `json:"currency"`
	SubtotalNet     int                    `json:"subtotal_net"`
	SubtotalGross   int                    `json:"subtotal_gross"`
	ShippingCost    int                    `json:"shipping_cost"`
	TaxTotal        int                    `json:"tax_total"`
	Total           int                    `json:"total"`
	BillingAddress  map[string]interface{} `json:"billing_address,omitempty"`
	ShippingAddress map[string]interface{} `json:"shipping_address,omitempty"`
	PaymentMethodID *uuid.UUID             `json:"payment_method_id,omitempty"`
	ShippingMethodID *uuid.UUID            `json:"shipping_method_id,omitempty"`
	Notes           string                 `json:"notes,omitempty"`
	GuestToken      string                 `json:"-"`
	CustomFields    map[string]interface{} `json:"custom_fields,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`

	// Relations populated on demand.
	Items         []OrderItem          `json:"items,omitempty"`
	StatusHistory []OrderStatusHistory `json:"status_history,omitempty"`
}

// OrderItem represents a single line item within an order.
type OrderItem struct {
	ID             uuid.UUID  `json:"id"`
	OrderID        uuid.UUID  `json:"order_id"`
	ProductID      *uuid.UUID `json:"product_id,omitempty"`
	VariantID      *uuid.UUID `json:"variant_id,omitempty"`
	SKU            string     `json:"sku"`
	Name           string     `json:"name"`
	Quantity       int        `json:"quantity"`
	UnitPriceNet   int        `json:"unit_price_net"`
	UnitPriceGross int        `json:"unit_price_gross"`
	TotalNet       int        `json:"total_net"`
	TotalGross     int        `json:"total_gross"`
	TaxRate        int        `json:"tax_rate"`
}

// OrderStatusHistory records every status change that an order goes through.
type OrderStatusHistory struct {
	ID         uuid.UUID `json:"id"`
	OrderID    uuid.UUID `json:"order_id"`
	FromStatus string    `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	Comment    string    `json:"comment,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}
