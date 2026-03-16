package warehouse

import "github.com/google/uuid"

// CreateWarehouseRequest is the request body for creating a warehouse.
type CreateWarehouseRequest struct {
	Name               string                 `json:"name"                validate:"required,min=1,max=255"`
	Code               string                 `json:"code"                validate:"required,min=1,max=50"`
	Active             bool                   `json:"active"`
	AllowNegativeStock *bool                  `json:"allow_negative_stock"`
	Priority           int                    `json:"priority"            validate:"min=0"`
	AddressLine1 string                 `json:"address_line1" validate:"max=255"`
	AddressLine2 string                 `json:"address_line2" validate:"max=255"`
	City         string                 `json:"city"         validate:"max=255"`
	State        string                 `json:"state"        validate:"max=255"`
	PostalCode   string                 `json:"postal_code"  validate:"max=50"`
	Country      string                 `json:"country"      validate:"max=2"`
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateWarehouseRequest is the request body for updating a warehouse.
type UpdateWarehouseRequest struct {
	Name               string                 `json:"name"                validate:"required,min=1,max=255"`
	Code               string                 `json:"code"                validate:"required,min=1,max=50"`
	Active             bool                   `json:"active"`
	AllowNegativeStock *bool                  `json:"allow_negative_stock"`
	Priority           int                    `json:"priority"            validate:"min=0"`
	AddressLine1 string                 `json:"address_line1" validate:"max=255"`
	AddressLine2 string                 `json:"address_line2" validate:"max=255"`
	City         string                 `json:"city"         validate:"max=255"`
	State        string                 `json:"state"        validate:"max=255"`
	PostalCode   string                 `json:"postal_code"  validate:"max=50"`
	Country      string                 `json:"country"      validate:"max=2"`
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// SetStockRequest is the request body for setting stock at a warehouse.
type SetStockRequest struct {
	Items []SetStockItem `json:"items" validate:"required,min=1,dive"`
}

// SetStockItem represents a single stock entry to set.
type SetStockItem struct {
	ProductID uuid.UUID  `json:"product_id" validate:"required"`
	VariantID *uuid.UUID `json:"variant_id"`
	Quantity  int        `json:"quantity"   validate:"min=0"`
	Reference string     `json:"reference"`
}
