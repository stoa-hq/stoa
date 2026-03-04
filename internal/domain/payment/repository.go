package payment

import (
	"context"

	"github.com/google/uuid"
)

// PaymentMethodFilter controls the result set returned by FindAll.
type PaymentMethodFilter struct {
	Page   int   `json:"page"`
	Limit  int   `json:"limit"`
	Active *bool `json:"active,omitempty"`
}

// PaymentMethodRepository defines the persistence contract for payment methods.
type PaymentMethodRepository interface {
	// FindByID retrieves a single payment method by ID, including its translations.
	FindByID(ctx context.Context, id uuid.UUID) (*PaymentMethod, error)

	// FindAll retrieves a paginated, filtered list of payment methods with translations.
	FindAll(ctx context.Context, filter PaymentMethodFilter) ([]PaymentMethod, int, error)

	// Create persists a new payment method and its translations.
	Create(ctx context.Context, m *PaymentMethod) error

	// Update persists changes to an existing payment method and replaces its translations.
	Update(ctx context.Context, m *PaymentMethod) error

	// Delete removes a payment method by ID.
	Delete(ctx context.Context, id uuid.UUID) error
}

// PaymentTransactionRepository defines the persistence contract for payment transactions.
type PaymentTransactionRepository interface {
	// Create persists a new payment transaction.
	Create(ctx context.Context, t *PaymentTransaction) error

	// FindByOrderID retrieves all transactions for a given order.
	FindByOrderID(ctx context.Context, orderID uuid.UUID) ([]PaymentTransaction, error)
}
