package category

import (
	"time"

	"github.com/google/uuid"
)

// Category represents a product category with optional hierarchy and i18n translations.
type Category struct {
	ID           uuid.UUID              `json:"id"`
	ParentID     *uuid.UUID             `json:"parent_id,omitempty"`
	Position     int                    `json:"position"`
	Active       bool                   `json:"active"`
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`

	// Eagerly loaded relations
	Translations []CategoryTranslation `json:"translations,omitempty"`
	Children     []Category            `json:"children,omitempty"`
}

// CategoryTranslation holds locale-specific fields for a category.
type CategoryTranslation struct {
	CategoryID  uuid.UUID `json:"category_id"`
	Locale      string    `json:"locale"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Slug        string    `json:"slug"`
}

// Translation returns the translation for the given locale, falling back to the
// first available translation when the requested locale is not present.
func (c *Category) Translation(locale string) *CategoryTranslation {
	if len(c.Translations) == 0 {
		return nil
	}
	for i := range c.Translations {
		if c.Translations[i].Locale == locale {
			return &c.Translations[i]
		}
	}
	return &c.Translations[0]
}
