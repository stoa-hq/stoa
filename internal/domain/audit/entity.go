package audit

import (
	"time"

	"github.com/google/uuid"
)

// AuditLog records a user action performed on an entity.
type AuditLog struct {
	ID         uuid.UUID              `json:"id"`
	UserID     *uuid.UUID             `json:"user_id,omitempty"`
	UserType   string                 `json:"user_type"`
	Action     string                 `json:"action"`
	EntityType string                 `json:"entity_type"`
	EntityID   *uuid.UUID             `json:"entity_id,omitempty"`
	Changes    map[string]interface{} `json:"changes,omitempty"`
	IPAddress  string                 `json:"ip_address"`
	CreatedAt  time.Time              `json:"created_at"`
}
