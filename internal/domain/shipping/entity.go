package shipping

import (
	"time"

	"github.com/google/uuid"
)

// ShippingMethod represents a shipping option available in the store.
// PriceNet and PriceGross are stored as integer cents (e.g. 499 = €4.99).
type ShippingMethod struct {
	ID           uuid.UUID                    `json:"id"`
	Active       bool                         `json:"active"`
	PriceNet     int                          `json:"price_net"`
	PriceGross   int                          `json:"price_gross"`
	TaxRuleID    *uuid.UUID                   `json:"tax_rule_id,omitempty"`
	CustomFields map[string]interface{}       `json:"custom_fields,omitempty"`
	CreatedAt    time.Time                    `json:"created_at"`
	UpdatedAt    time.Time                    `json:"updated_at"`
	Translations []ShippingMethodTranslation  `json:"translations,omitempty"`
}

// ShippingMethodTranslation holds locale-specific content for a ShippingMethod.
type ShippingMethodTranslation struct {
	ShippingMethodID uuid.UUID `json:"shipping_method_id"`
	Locale           string    `json:"locale"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
}
