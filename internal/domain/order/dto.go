package order

import (
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Request DTOs
// ---------------------------------------------------------------------------

// CheckoutRequest is the store-facing payload for creating a new order.
type CheckoutRequest struct {
	Currency         string                 `json:"currency"          validate:"required,len=3"`
	BillingAddress   map[string]interface{} `json:"billing_address"   validate:"required"`
	ShippingAddress  map[string]interface{} `json:"shipping_address"  validate:"required"`
	PaymentMethodID  *uuid.UUID             `json:"payment_method_id"`
	ShippingMethodID *uuid.UUID             `json:"shipping_method_id"`
	Notes            string                 `json:"notes,omitempty"   validate:"omitempty,max=1000"`
	Items            []CheckoutItemRequest  `json:"items"             validate:"required,min=1,dive"`
	CustomFields     map[string]interface{} `json:"custom_fields,omitempty"`
}

// CheckoutItemRequest describes a single line item in a checkout request.
type CheckoutItemRequest struct {
	ProductID      *uuid.UUID `json:"product_id"`
	VariantID      *uuid.UUID `json:"variant_id"`
	SKU            string     `json:"sku"             validate:"required,max=100"`
	Name           string     `json:"name"            validate:"required,max=255"`
	Quantity       int        `json:"quantity"        validate:"required,min=1"`
	UnitPriceNet   int        `json:"unit_price_net"  validate:"min=0"`
	UnitPriceGross int        `json:"unit_price_gross" validate:"min=0"`
	TaxRate        int        `json:"tax_rate"        validate:"min=0"`
}

// UpdateStatusRequest is the admin-facing payload for transitioning an order's status.
type UpdateStatusRequest struct {
	Status  string `json:"status"            validate:"required,oneof=pending confirmed processing shipped delivered cancelled refunded"`
	Comment string `json:"comment,omitempty" validate:"omitempty,max=1000"`
}

// ---------------------------------------------------------------------------
// Response DTOs
// ---------------------------------------------------------------------------

// OrderItemResponse is the line-item projection in API responses.
type OrderItemResponse struct {
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

// OrderStatusHistoryResponse records a single status change in API responses.
type OrderStatusHistoryResponse struct {
	ID         uuid.UUID `json:"id"`
	OrderID    uuid.UUID `json:"order_id"`
	FromStatus string    `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	Comment    string    `json:"comment,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// OrderResponse is the full order projection returned by the API.
type OrderResponse struct {
	ID               uuid.UUID                    `json:"id"`
	OrderNumber      string                       `json:"order_number"`
	CustomerID       *uuid.UUID                   `json:"customer_id,omitempty"`
	Status           string                       `json:"status"`
	Currency         string                       `json:"currency"`
	SubtotalNet      int                          `json:"subtotal_net"`
	SubtotalGross    int                          `json:"subtotal_gross"`
	ShippingCost     int                          `json:"shipping_cost"`
	TaxTotal         int                          `json:"tax_total"`
	Total            int                          `json:"total"`
	BillingAddress   map[string]interface{}       `json:"billing_address,omitempty"`
	ShippingAddress  map[string]interface{}       `json:"shipping_address,omitempty"`
	PaymentMethodID  *uuid.UUID                   `json:"payment_method_id,omitempty"`
	ShippingMethodID *uuid.UUID                   `json:"shipping_method_id,omitempty"`
	Notes            string                       `json:"notes,omitempty"`
	CustomFields     map[string]interface{}       `json:"custom_fields,omitempty"`
	CreatedAt        time.Time                    `json:"created_at"`
	UpdatedAt        time.Time                    `json:"updated_at"`
	Items            []OrderItemResponse          `json:"items,omitempty"`
	StatusHistory    []OrderStatusHistoryResponse `json:"status_history,omitempty"`
}

// ---------------------------------------------------------------------------
// Mapping helpers — entity → response DTO
// ---------------------------------------------------------------------------

// ToResponse converts an Order entity to its API response representation.
func ToResponse(o *Order) OrderResponse {
	resp := OrderResponse{
		ID:               o.ID,
		OrderNumber:      o.OrderNumber,
		CustomerID:       o.CustomerID,
		Status:           o.Status,
		Currency:         o.Currency,
		SubtotalNet:      o.SubtotalNet,
		SubtotalGross:    o.SubtotalGross,
		ShippingCost:     o.ShippingCost,
		TaxTotal:         o.TaxTotal,
		Total:            o.Total,
		BillingAddress:   o.BillingAddress,
		ShippingAddress:  o.ShippingAddress,
		PaymentMethodID:  o.PaymentMethodID,
		ShippingMethodID: o.ShippingMethodID,
		Notes:            o.Notes,
		CustomFields:     o.CustomFields,
		CreatedAt:        o.CreatedAt,
		UpdatedAt:        o.UpdatedAt,
	}

	for _, item := range o.Items {
		resp.Items = append(resp.Items, OrderItemResponse{
			ID:             item.ID,
			OrderID:        item.OrderID,
			ProductID:      item.ProductID,
			VariantID:      item.VariantID,
			SKU:            item.SKU,
			Name:           item.Name,
			Quantity:       item.Quantity,
			UnitPriceNet:   item.UnitPriceNet,
			UnitPriceGross: item.UnitPriceGross,
			TotalNet:       item.TotalNet,
			TotalGross:     item.TotalGross,
			TaxRate:        item.TaxRate,
		})
	}

	for _, h := range o.StatusHistory {
		resp.StatusHistory = append(resp.StatusHistory, OrderStatusHistoryResponse{
			ID:         h.ID,
			OrderID:    h.OrderID,
			FromStatus: h.FromStatus,
			ToStatus:   h.ToStatus,
			Comment:    h.Comment,
			CreatedAt:  h.CreatedAt,
		})
	}

	return resp
}

// ---------------------------------------------------------------------------
// Mapping helpers — request DTO → entity
// ---------------------------------------------------------------------------

// FromCheckoutRequest builds an Order entity from a CheckoutRequest.
func FromCheckoutRequest(req *CheckoutRequest, customerID *uuid.UUID) *Order {
	o := &Order{
		CustomerID:       customerID,
		Currency:         req.Currency,
		BillingAddress:   req.BillingAddress,
		ShippingAddress:  req.ShippingAddress,
		PaymentMethodID:  req.PaymentMethodID,
		ShippingMethodID: req.ShippingMethodID,
		Notes:            req.Notes,
		CustomFields:     req.CustomFields,
	}

	for _, item := range req.Items {
		i := OrderItem{
			ID:             uuid.New(),
			ProductID:      item.ProductID,
			VariantID:      item.VariantID,
			SKU:            item.SKU,
			Name:           item.Name,
			Quantity:       item.Quantity,
			UnitPriceNet:   item.UnitPriceNet,
			UnitPriceGross: item.UnitPriceGross,
			TaxRate:        item.TaxRate,
		}
		i.TotalNet = i.UnitPriceNet * i.Quantity
		i.TotalGross = i.UnitPriceGross * i.Quantity
		o.Items = append(o.Items, i)
	}

	// Compute order-level totals from line items.
	for _, item := range o.Items {
		o.SubtotalNet += item.TotalNet
		o.SubtotalGross += item.TotalGross
	}
	o.Total = o.SubtotalGross + o.ShippingCost

	return o
}
