package shipping

import "github.com/google/uuid"

// TranslationInput holds locale-specific fields for create/update requests.
type TranslationInput struct {
	Locale      string `json:"locale"      validate:"required,min=2,max=10"`
	Name        string `json:"name"        validate:"required,min=1,max=255"`
	Description string `json:"description"`
}

// CreateShippingMethodRequest is the request body for creating a shipping method.
type CreateShippingMethodRequest struct {
	Active       bool                   `json:"active"`
	PriceNet     int                    `json:"price_net"   validate:"min=0"`
	PriceGross   int                    `json:"price_gross" validate:"min=0"`
	TaxRuleID    *uuid.UUID             `json:"tax_rule_id"`
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
	Translations []TranslationInput     `json:"translations" validate:"dive"`
}

// UpdateShippingMethodRequest is the request body for updating a shipping method.
type UpdateShippingMethodRequest struct {
	Active       bool                   `json:"active"`
	PriceNet     int                    `json:"price_net"   validate:"min=0"`
	PriceGross   int                    `json:"price_gross" validate:"min=0"`
	TaxRuleID    *uuid.UUID             `json:"tax_rule_id"`
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
	Translations []TranslationInput     `json:"translations" validate:"dive"`
}

// ListShippingMethodsRequest holds query parameters for the List endpoint.
type ListShippingMethodsRequest struct {
	Page   int   `json:"page"`
	Limit  int   `json:"limit"`
	Active *bool `json:"active,omitempty"`
}
