package tag

import "github.com/google/uuid"

// Tag is a label that can be attached to products or other entities.
type Tag struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Slug string    `json:"slug"`
}
