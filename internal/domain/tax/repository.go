package tax

import (
	"context"

	"github.com/google/uuid"
)

// TaxRuleFilter controls the result set returned by FindAll.
type TaxRuleFilter struct {
	Page        int    `json:"page"`
	Limit       int    `json:"limit"`
	CountryCode string `json:"country_code,omitempty"`
	Type        string `json:"type,omitempty"`
}

// TaxRuleRepository defines the persistence contract for the tax domain.
type TaxRuleRepository interface {
	// FindByID retrieves a single tax rule by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*TaxRule, error)

	// FindAll retrieves a paginated, filtered list of tax rules.
	// It returns the matching rules, the total count (for pagination), and any error.
	FindAll(ctx context.Context, filter TaxRuleFilter) ([]TaxRule, int, error)

	// Create persists a new tax rule.
	Create(ctx context.Context, t *TaxRule) error

	// Update persists changes to an existing tax rule.
	Update(ctx context.Context, t *TaxRule) error

	// Delete removes a tax rule by ID.
	Delete(ctx context.Context, id uuid.UUID) error
}
