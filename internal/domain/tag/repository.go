package tag

import (
	"context"

	"github.com/google/uuid"
)

// TagFilter controls the result set returned by FindAll.
type TagFilter struct {
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
	Name  string `json:"name,omitempty"`
}

// TagRepository defines the persistence contract for the tag domain.
type TagRepository interface {
	// FindByID retrieves a single tag by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*Tag, error)

	// FindAll retrieves a paginated, filtered list of tags.
	// It returns the matching tags, the total count, and any error.
	FindAll(ctx context.Context, filter TagFilter) ([]Tag, int, error)

	// Create persists a new tag.
	Create(ctx context.Context, t *Tag) error

	// Update persists changes to an existing tag.
	Update(ctx context.Context, t *Tag) error

	// Delete removes a tag by ID.
	Delete(ctx context.Context, id uuid.UUID) error
}
