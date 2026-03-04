package customer

import (
	"context"

	"github.com/google/uuid"
)

// CustomerRepository defines the persistence contract for the customer domain.
type CustomerRepository interface {
	// FindByID retrieves a single customer with their addresses.
	FindByID(ctx context.Context, id uuid.UUID) (*Customer, error)

	// FindByEmail retrieves a single customer by their email address.
	// Returns nil, nil when no customer is found.
	FindByEmail(ctx context.Context, email string) (*Customer, error)

	// FindAll retrieves a paginated, filtered list of customers.
	// It returns the matching customers, the total count (for pagination), and any error.
	FindAll(ctx context.Context, filter CustomerFilter) ([]Customer, int, error)

	// Create persists a new customer record.
	Create(ctx context.Context, c *Customer) error

	// Update persists changes to an existing customer record.
	Update(ctx context.Context, c *Customer) error

	// Delete removes a customer and all dependent rows (via CASCADE).
	Delete(ctx context.Context, id uuid.UUID) error

	// CreateAddress persists a new address for a customer.
	CreateAddress(ctx context.Context, a *CustomerAddress) error

	// UpdateAddress persists changes to an existing customer address.
	UpdateAddress(ctx context.Context, a *CustomerAddress) error

	// DeleteAddress removes a single customer address.
	DeleteAddress(ctx context.Context, id uuid.UUID) error

	// FindAddressesByCustomerID retrieves all addresses for a given customer.
	FindAddressesByCustomerID(ctx context.Context, customerID uuid.UUID) ([]CustomerAddress, error)
}

// CustomerFilter controls the result set returned by FindAll.
type CustomerFilter struct {
	// Pagination
	Page  int `json:"page"`
	Limit int `json:"limit"`

	// Filters
	Search string `json:"search,omitempty"`
	Active *bool  `json:"active,omitempty"`
}
