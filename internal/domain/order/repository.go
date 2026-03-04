package order

import (
	"context"

	"github.com/google/uuid"
)

// OrderFilter controls pagination, filtering, and sorting for FindAll.
type OrderFilter struct {
	Page       int
	Limit      int
	Status     string
	CustomerID *uuid.UUID
	Sort       string
	Order      string // "asc" or "desc"
}

// OrderRepository defines persistence operations for orders.
type OrderRepository interface {
	// FindByID returns a single order with its items and status history.
	FindByID(ctx context.Context, id uuid.UUID) (*Order, error)

	// FindAll returns a paginated, filtered list of orders together with
	// the total row count.
	FindAll(ctx context.Context, filter OrderFilter) ([]Order, int, error)

	// FindByCustomerID returns all orders that belong to a given customer.
	FindByCustomerID(ctx context.Context, customerID uuid.UUID) ([]Order, error)

	// Create persists a new order together with all its order items.
	Create(ctx context.Context, o *Order) error

	// Update persists changes to an existing order (header fields only).
	Update(ctx context.Context, o *Order) error

	// UpdateStatus transitions an order to a new status and appends an entry to
	// order_status_history in the same transaction.
	UpdateStatus(ctx context.Context, id uuid.UUID, fromStatus, toStatus, comment string) error
}
