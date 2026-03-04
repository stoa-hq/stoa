package discount

import (
	"context"

	"github.com/google/uuid"
)

// DiscountFilter controls the result set returned by FindAll.
type DiscountFilter struct {
	Page   int    `json:"page"`
	Limit  int    `json:"limit"`
	Active *bool  `json:"active,omitempty"`
	Type   string `json:"type,omitempty"`
	Code   string `json:"code,omitempty"`
}

// DiscountRepository defines the persistence contract for the discount domain.
type DiscountRepository interface {
	// FindByID retrieves a single discount by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*Discount, error)

	// FindByCode retrieves a single discount by code.
	FindByCode(ctx context.Context, code string) (*Discount, error)

	// FindAll retrieves a paginated, filtered list of discounts.
	FindAll(ctx context.Context, filter DiscountFilter) ([]Discount, int, error)

	// Create persists a new discount.
	Create(ctx context.Context, d *Discount) error

	// Update persists changes to an existing discount.
	Update(ctx context.Context, d *Discount) error

	// Delete removes a discount by ID.
	Delete(ctx context.Context, id uuid.UUID) error

	// IncrementUsedCount atomically increments the used_count of a discount by ID.
	IncrementUsedCount(ctx context.Context, id uuid.UUID) error
}
