package product

import (
	"context"

	"github.com/google/uuid"
)

// ProductRepository defines the persistence contract for the product domain.
type ProductRepository interface {
	// FindByID retrieves a single product with all its relations.
	FindByID(ctx context.Context, id uuid.UUID) (*Product, error)

	// FindAll retrieves a paginated, filtered list of products.
	// It returns the matching products, the total count (for pagination), and any error.
	FindAll(ctx context.Context, filter ProductFilter) ([]Product, int, error)

	// FindBySlug retrieves a product whose translation for the given locale has the given slug.
	FindBySlug(ctx context.Context, slug, locale string) (*Product, error)

	// Create persists a new product and its translations.
	Create(ctx context.Context, p *Product) error

	// Update persists changes to an existing product and its translations.
	Update(ctx context.Context, p *Product) error

	// Delete removes a product and all its dependent rows (via CASCADE).
	Delete(ctx context.Context, id uuid.UUID) error

	// StockAvailable reports whether the requested quantity is in stock.
	// When variantID is non-nil the variant stock is checked; otherwise the
	// product-level stock is used.
	StockAvailable(ctx context.Context, productID uuid.UUID, variantID *uuid.UUID, quantity int) (bool, error)

	// Variants
	CreateVariant(ctx context.Context, v *ProductVariant) error
	FindVariantByID(ctx context.Context, id uuid.UUID) (*ProductVariant, error)
	UpdateVariant(ctx context.Context, v *ProductVariant) error
	DeleteVariant(ctx context.Context, id uuid.UUID) error

	// Property Groups
	FindAllPropertyGroups(ctx context.Context) ([]PropertyGroup, error)
	FindPropertyGroupByID(ctx context.Context, id uuid.UUID) (*PropertyGroup, error)
	CreatePropertyGroup(ctx context.Context, g *PropertyGroup) error
	UpdatePropertyGroup(ctx context.Context, g *PropertyGroup) error
	DeletePropertyGroup(ctx context.Context, id uuid.UUID) error

	// Property Options
	FindOptionsByGroupID(ctx context.Context, groupID uuid.UUID) ([]PropertyOption, error)
	CreatePropertyOption(ctx context.Context, o *PropertyOption) error
	UpdatePropertyOption(ctx context.Context, o *PropertyOption) error
	DeletePropertyOption(ctx context.Context, id uuid.UUID) error
}

// ProductFilter controls the result set returned by FindAll.
type ProductFilter struct {
	// Pagination
	Page  int `json:"page"`
	Limit int `json:"limit"`

	// Filters
	Active     *bool      `json:"active,omitempty"`
	CategoryID *uuid.UUID `json:"category_id,omitempty"`
	Search     string     `json:"search,omitempty"`

	// Sorting – Sort is the column name, Order is "asc" or "desc".
	Sort  string `json:"sort,omitempty"`
	Order string `json:"order,omitempty"`
}
