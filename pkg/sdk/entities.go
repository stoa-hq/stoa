package sdk

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// JSONB represents a JSONB column value.
type JSONB map[string]interface{}

// Scan implements the sql.Scanner interface.
func (j *JSONB) Scan(src interface{}) error {
	if src == nil {
		*j = nil
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	}
	return json.Unmarshal(data, j)
}

// BaseEntity provides common fields for all entities.
type BaseEntity struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	CustomFields JSONB     `json:"custom_fields,omitempty"`
	Metadata     JSONB     `json:"metadata,omitempty"`
}
