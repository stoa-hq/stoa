package shipping

import (
	"context"

	"github.com/google/uuid"
)

// ShippingMethodFilter controls the result set returned by FindAll.
type ShippingMethodFilter struct {
	Page   int   `json:"page"`
	Limit  int   `json:"limit"`
	Active *bool `json:"active,omitempty"`
}

// ShippingMethodRepository defines the persistence contract for the shipping domain.
type ShippingMethodRepository interface {
	// FindByID retrieves a single shipping method by ID, including its translations.
	FindByID(ctx context.Context, id uuid.UUID) (*ShippingMethod, error)

	// FindAll retrieves a paginated, filtered list of shipping methods with translations.
	FindAll(ctx context.Context, filter ShippingMethodFilter) ([]ShippingMethod, int, error)

	// Create persists a new shipping method and its translations.
	Create(ctx context.Context, m *ShippingMethod) error

	// Update persists changes to an existing shipping method and replaces its translations.
	Update(ctx context.Context, m *ShippingMethod) error

	// Delete removes a shipping method by ID.
	Delete(ctx context.Context, id uuid.UUID) error
}
