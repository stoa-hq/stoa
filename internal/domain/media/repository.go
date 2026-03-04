package media

import (
	"context"

	"github.com/google/uuid"
)

// MediaFilter controls the result set returned by FindAll.
type MediaFilter struct {
	Page     int    `json:"page"`
	Limit    int    `json:"limit"`
	MimeType string `json:"mime_type,omitempty"`
}

// MediaRepository defines the persistence contract for the media domain.
type MediaRepository interface {
	// Create persists a new media record.
	Create(ctx context.Context, m *Media) error

	// FindByID retrieves a single media record by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*Media, error)

	// FindAll retrieves a paginated, filtered list of media records.
	FindAll(ctx context.Context, filter MediaFilter) ([]Media, int, error)

	// Delete removes a media record by ID.
	Delete(ctx context.Context, id uuid.UUID) error
}
