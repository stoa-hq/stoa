package discount

import "time"

// CreateDiscountRequest is the request body for creating a discount.
type CreateDiscountRequest struct {
	Code          string                 `json:"code"            validate:"required,min=1,max=100"`
	Type          string                 `json:"type"            validate:"required,oneof=percentage fixed"`
	Value         int                    `json:"value"           validate:"required,min=1"`
	MinOrderValue *int                   `json:"min_order_value" validate:"omitempty,min=0"`
	MaxUses       *int                   `json:"max_uses"        validate:"omitempty,min=1"`
	ValidFrom     *time.Time             `json:"valid_from"      validate:"omitempty"`
	ValidUntil    *time.Time             `json:"valid_until"     validate:"omitempty"`
	Active        bool                   `json:"active"`
	Conditions    map[string]interface{} `json:"conditions,omitempty"`
}

// UpdateDiscountRequest is the request body for updating a discount.
type UpdateDiscountRequest struct {
	Code          string                 `json:"code"            validate:"required,min=1,max=100"`
	Type          string                 `json:"type"            validate:"required,oneof=percentage fixed"`
	Value         int                    `json:"value"           validate:"required,min=1"`
	MinOrderValue *int                   `json:"min_order_value" validate:"omitempty,min=0"`
	MaxUses       *int                   `json:"max_uses"        validate:"omitempty,min=1"`
	ValidFrom     *time.Time             `json:"valid_from"      validate:"omitempty"`
	ValidUntil    *time.Time             `json:"valid_until"     validate:"omitempty"`
	Active        bool                   `json:"active"`
	Conditions    map[string]interface{} `json:"conditions,omitempty"`
}

// ValidateCodeRequest is the request body for validating a discount code.
type ValidateCodeRequest struct {
	Code       string `json:"code"        validate:"required"`
	OrderTotal int    `json:"order_total" validate:"required,min=0"`
}

// ListDiscountsRequest holds query parameters for the List endpoint.
type ListDiscountsRequest struct {
	Page   int    `json:"page"`
	Limit  int    `json:"limit"`
	Active *bool  `json:"active,omitempty"`
	Type   string `json:"type,omitempty"`
	Code   string `json:"code,omitempty"`
}
