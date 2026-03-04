package cart

import (
	"context"

	"github.com/google/uuid"
)

// CartRepository defines the persistence contract for the cart domain.
type CartRepository interface {
	// FindByID retrieves a cart by its primary key, including all its items.
	FindByID(ctx context.Context, id uuid.UUID) (*Cart, error)

	// FindBySessionID retrieves the active cart associated with a guest session.
	FindBySessionID(ctx context.Context, sessionID string) (*Cart, error)

	// FindByCustomerID retrieves the active cart associated with a customer.
	FindByCustomerID(ctx context.Context, customerID uuid.UUID) (*Cart, error)

	// Create persists a new cart record.
	Create(ctx context.Context, c *Cart) error

	// Delete removes a cart and all its items by ID.
	Delete(ctx context.Context, id uuid.UUID) error

	// AddItem inserts a new line item into a cart.
	AddItem(ctx context.Context, item *CartItem) error

	// UpdateItem changes the quantity of an existing cart item.
	UpdateItem(ctx context.Context, itemID uuid.UUID, quantity int) error

	// RemoveItem deletes a single line item from its cart.
	RemoveItem(ctx context.Context, itemID uuid.UUID) error

	// CleanExpired removes all carts whose expiry timestamp is in the past.
	CleanExpired(ctx context.Context) error
}
