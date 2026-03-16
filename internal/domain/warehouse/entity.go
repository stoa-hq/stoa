package warehouse

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Movement type constants for stock_movements.
const (
	MovementSale       = "sale"
	MovementRestock    = "restock"
	MovementAdjustment = "adjustment"
	MovementReturn     = "return"
)

// Sentinel errors for the warehouse domain.
var (
	ErrNotFound          = errors.New("warehouse: not found")
	ErrInsufficientStock = errors.New("warehouse: insufficient stock")
	ErrDuplicateCode     = errors.New("warehouse: duplicate code")
)

// Warehouse represents a physical storage location.
type Warehouse struct {
	ID           uuid.UUID              `json:"id"`
	Name         string                 `json:"name"`
	Code         string                 `json:"code"`
	Active              bool                   `json:"active"`
	AllowNegativeStock  bool                   `json:"allow_negative_stock"`
	Priority            int                    `json:"priority"`
	AddressLine1 string                 `json:"address_line1"`
	AddressLine2 string                 `json:"address_line2"`
	City         string                 `json:"city"`
	State        string                 `json:"state"`
	PostalCode   string                 `json:"postal_code"`
	Country      string                 `json:"country"`
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// WarehouseStock represents the inventory of a product (or variant) at a warehouse.
type WarehouseStock struct {
	ID          uuid.UUID  `json:"id"`
	WarehouseID uuid.UUID  `json:"warehouse_id"`
	ProductID   uuid.UUID  `json:"product_id"`
	VariantID   *uuid.UUID `json:"variant_id,omitempty"`
	Quantity    int        `json:"quantity"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// Populated on demand for display purposes.
	WarehouseName string `json:"warehouse_name,omitempty"`
	WarehouseCode string `json:"warehouse_code,omitempty"`
	ProductSKU    string `json:"product_sku,omitempty"`
	ProductName   string `json:"product_name,omitempty"`
	VariantSKU    string `json:"variant_sku,omitempty"`
}

// StockMovement records a change in inventory.
type StockMovement struct {
	ID           uuid.UUID  `json:"id"`
	WarehouseID  uuid.UUID  `json:"warehouse_id"`
	ProductID    uuid.UUID  `json:"product_id"`
	VariantID    *uuid.UUID `json:"variant_id,omitempty"`
	OrderID      *uuid.UUID `json:"order_id,omitempty"`
	MovementType string     `json:"movement_type"`
	Quantity     int        `json:"quantity"`
	Reference    string     `json:"reference"`
	CreatedAt    time.Time  `json:"created_at"`
}

// StockDeductionItem describes a single line item for stock deduction.
type StockDeductionItem struct {
	ProductID uuid.UUID
	VariantID *uuid.UUID
	Quantity  int
	OrderID   uuid.UUID
}
