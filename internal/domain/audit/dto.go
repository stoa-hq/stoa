package audit

import "github.com/google/uuid"

// ListAuditLogsRequest holds query parameters for the List endpoint.
type ListAuditLogsRequest struct {
	Page       int        `json:"page"`
	Limit      int        `json:"limit"`
	UserID     *uuid.UUID `json:"user_id,omitempty"`
	EntityType string     `json:"entity_type,omitempty"`
	EntityID   *uuid.UUID `json:"entity_id,omitempty"`
	Action     string     `json:"action,omitempty"`
}
