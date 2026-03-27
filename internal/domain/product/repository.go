package product

import (
	"context"

	"github.com/google/uuid"
)

// ProductRepository defines the persistence contract for the product domain.
type ProductRepository interface {
	// FindByID retrieves a single product with all its relations.
	FindByID(ctx context.Context, id uuid.UUID) (*Product, error)

	// FindBySKU retrieves a product by its unique SKU.
	// Returns ErrNotFound when no product has the given SKU.
	FindBySKU(ctx context.Context, sku string) (*Product, error)

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
	FindPropertyGroupByIdentifier(ctx context.Context, identifier string) (*PropertyGroup, error)
	CreatePropertyGroup(ctx context.Context, g *PropertyGroup) error
	UpdatePropertyGroup(ctx context.Context, g *PropertyGroup) error
	DeletePropertyGroup(ctx context.Context, id uuid.UUID) error

	// Property Options
	FindOptionsByGroupID(ctx context.Context, groupID uuid.UUID) ([]PropertyOption, error)
	CreatePropertyOption(ctx context.Context, o *PropertyOption) error
	UpdatePropertyOption(ctx context.Context, o *PropertyOption) error
	DeletePropertyOption(ctx context.Context, id uuid.UUID) error

	// Bulk / Import helpers
	FindOrCreatePropertyGroup(ctx context.Context, locale, name string) (*PropertyGroup, error)
	FindOrCreatePropertyOption(ctx context.Context, groupID uuid.UUID, locale, name string) (*PropertyOption, error)

	// Attributes
	FindAllAttributes(ctx context.Context) ([]Attribute, error)
	FindAttributeByID(ctx context.Context, id uuid.UUID) (*Attribute, error)
	FindAttributeByIdentifier(ctx context.Context, identifier string) (*Attribute, error)
	CreateAttribute(ctx context.Context, a *Attribute) error
	UpdateAttribute(ctx context.Context, a *Attribute) error
	DeleteAttribute(ctx context.Context, id uuid.UUID) error

	// Attribute Options
	FindAttributeOptionsByAttributeID(ctx context.Context, attrID uuid.UUID) ([]AttributeOption, error)
	CreateAttributeOption(ctx context.Context, o *AttributeOption) error
	UpdateAttributeOption(ctx context.Context, o *AttributeOption) error
	DeleteAttributeOption(ctx context.Context, id uuid.UUID) error

	// Product Attribute Values
	FindProductAttributeValues(ctx context.Context, productID uuid.UUID) ([]AttributeValue, error)
	SetProductAttributeValue(ctx context.Context, productID uuid.UUID, val *AttributeValue) error
	DeleteProductAttributeValue(ctx context.Context, productID, attributeID uuid.UUID) error

	// Variant Attribute Values
	FindVariantAttributeValues(ctx context.Context, variantID uuid.UUID) ([]AttributeValue, error)
	SetVariantAttributeValue(ctx context.Context, variantID uuid.UUID, val *AttributeValue) error
	DeleteVariantAttributeValue(ctx context.Context, variantID, attributeID uuid.UUID) error
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
