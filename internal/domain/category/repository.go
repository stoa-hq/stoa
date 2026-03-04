package category

import (
	"context"

	"github.com/google/uuid"
)

// CategoryFilter controls pagination and filtering for FindAll.
type CategoryFilter struct {
	Page     int
	Limit    int
	ParentID *uuid.UUID
	Active   *bool
}

// CategoryRepository defines persistence operations for categories.
type CategoryRepository interface {
	// FindByID returns a single category with all its translations.
	FindByID(ctx context.Context, id uuid.UUID) (*Category, error)

	// FindAll returns a paginated list of categories matching the filter along
	// with the total row count.
	FindAll(ctx context.Context, filter CategoryFilter) ([]Category, int, error)

	// FindTree returns the full category tree for the given locale.  Each
	// root-level category contains its children recursively.
	FindTree(ctx context.Context, locale string) ([]Category, error)

	// FindBySlug returns the category whose translation for locale matches slug.
	FindBySlug(ctx context.Context, slug, locale string) (*Category, error)

	// Create persists a new category together with its translations.
	Create(ctx context.Context, c *Category) error

	// Update persists changes to an existing category and its translations.
	Update(ctx context.Context, c *Category) error

	// Delete removes a category by ID.
	Delete(ctx context.Context, id uuid.UUID) error
}
