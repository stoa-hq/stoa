package product

import (
	"time"

	"github.com/google/uuid"
)

// Product is the central aggregate for the product domain.
type Product struct {
	ID           uuid.UUID              `json:"id"`
	SKU          string                 `json:"sku"`
	Active       bool                   `json:"active"`
	PriceNet     int                    `json:"price_net"`
	PriceGross   int                    `json:"price_gross"`
	Currency     string                 `json:"currency"`
	TaxRuleID    *uuid.UUID             `json:"tax_rule_id,omitempty"`
	Stock        int                    `json:"stock"`
	Weight       int                    `json:"weight"`
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`

	// Relations (populated on demand via JOINs / secondary queries)
	Translations []ProductTranslation `json:"translations,omitempty"`
	Categories   []uuid.UUID          `json:"categories,omitempty"`
	Tags         []uuid.UUID          `json:"tags,omitempty"`
	Media        []ProductMedia       `json:"media,omitempty"`
	Variants     []ProductVariant     `json:"variants,omitempty"`
	Attributes   []AttributeValue    `json:"attributes,omitempty"`

	// HasVariants is true if at least one active variant exists.
	// Populated by FindAll (list queries) via a lightweight EXISTS sub-query.
	HasVariants bool `json:"has_variants"`
}

// ProductTranslation holds locale-specific content for a product.
type ProductTranslation struct {
	ProductID       uuid.UUID `json:"product_id"`
	Locale          string    `json:"locale"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Slug            string    `json:"slug"`
	MetaTitle       string    `json:"meta_title"`
	MetaDescription string    `json:"meta_description"`
}

// ProductMedia links a media asset to a product with an ordering position.
type ProductMedia struct {
	MediaID     uuid.UUID `json:"media_id"`
	Position    int       `json:"position"`
	StoragePath string    `json:"-"`            // populated via JOIN with media table
	URL         string    `json:"url,omitempty"` // computed by the service layer
}

// ProductVariant is a purchasable variant of a product (e.g. a specific size/color).
type ProductVariant struct {
	ID           uuid.UUID              `json:"id"`
	ProductID    uuid.UUID              `json:"product_id"`
	SKU          string                 `json:"sku"`
	PriceNet     *int                   `json:"price_net,omitempty"`
	PriceGross   *int                   `json:"price_gross,omitempty"`
	Stock        int                    `json:"stock"`
	Active       bool                   `json:"active"`
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`

	// The property options that define this variant (e.g. Size=L, Color=Red).
	Options    []PropertyOption `json:"options,omitempty"`
	Attributes []AttributeValue `json:"attributes,omitempty"`
}

// PropertyGroup groups related property options together (e.g. "Size", "Color").
type PropertyGroup struct {
	ID           uuid.UUID                  `json:"id"`
	Identifier   string                     `json:"identifier"`
	Position     int                        `json:"position"`
	CreatedAt    time.Time                  `json:"created_at"`
	UpdatedAt    time.Time                  `json:"updated_at"`
	Translations []PropertyGroupTranslation `json:"translations,omitempty"`
	Options      []PropertyOption           `json:"options,omitempty"`
}

// PropertyGroupTranslation provides locale-specific names for a property group.
type PropertyGroupTranslation struct {
	GroupID uuid.UUID `json:"group_id"`
	Locale  string    `json:"locale"`
	Name    string    `json:"name"`
}

// PropertyOption is a single selectable value within a PropertyGroup (e.g. "L", "Red").
type PropertyOption struct {
	ID           uuid.UUID                   `json:"id"`
	GroupID      uuid.UUID                   `json:"group_id"`
	ColorHex     string                      `json:"color_hex,omitempty"`
	Position     int                         `json:"position"`
	CreatedAt    time.Time                   `json:"created_at"`
	UpdatedAt    time.Time                   `json:"updated_at"`
	Translations []PropertyOptionTranslation `json:"translations,omitempty"`
}

// PropertyOptionTranslation provides locale-specific names for a property option.
type PropertyOptionTranslation struct {
	OptionID uuid.UUID `json:"option_id"`
	Locale   string    `json:"locale"`
	Name     string    `json:"name"`
}

// --------------------------------------------------------------------------
// Product Attributes (generic classification, NOT variant-defining)
// --------------------------------------------------------------------------

// Attribute defines a custom attribute that admins can assign to products/variants.
type Attribute struct {
	ID           uuid.UUID              `json:"id"`
	Identifier   string                 `json:"identifier"`
	Type         string                 `json:"type"` // text, number, select, multi_select, boolean
	Unit         string                 `json:"unit,omitempty"`
	Position     int                    `json:"position"`
	Filterable   bool                   `json:"filterable"`
	Required     bool                   `json:"required"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Translations []AttributeTranslation `json:"translations,omitempty"`
	Options      []AttributeOption      `json:"options,omitempty"`
}

// AttributeTranslation provides locale-specific name and description for an attribute.
type AttributeTranslation struct {
	AttributeID uuid.UUID `json:"attribute_id"`
	Locale      string    `json:"locale"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}

// AttributeOption is a predefined value for select / multi_select attributes.
type AttributeOption struct {
	ID           uuid.UUID                    `json:"id"`
	AttributeID  uuid.UUID                    `json:"attribute_id"`
	Position     int                          `json:"position"`
	CreatedAt    time.Time                    `json:"created_at"`
	UpdatedAt    time.Time                    `json:"updated_at"`
	Translations []AttributeOptionTranslation `json:"translations,omitempty"`
}

// AttributeOptionTranslation provides locale-specific names for an attribute option.
type AttributeOptionTranslation struct {
	OptionID uuid.UUID `json:"option_id"`
	Locale   string    `json:"locale"`
	Name     string    `json:"name"`
}

// AttributeValue holds a value assignment for a product or variant attribute.
type AttributeValue struct {
	ID           uuid.UUID   `json:"id"`
	AttributeID  uuid.UUID   `json:"attribute_id"`
	ValueText    *string     `json:"value_text,omitempty"`
	ValueNumeric *float64    `json:"value_numeric,omitempty"`
	ValueBoolean *bool       `json:"value_boolean,omitempty"`
	OptionID     *uuid.UUID  `json:"option_id,omitempty"`
	OptionIDs    []uuid.UUID `json:"option_ids,omitempty"` // for multi_select
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`

	// Populated for responses – the full attribute definition.
	Attribute *Attribute `json:"attribute,omitempty"`
}
