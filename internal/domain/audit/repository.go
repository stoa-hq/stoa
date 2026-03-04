package audit

import (
	"context"

	"github.com/google/uuid"
)

// AuditFilter controls the result set returned by FindAll.
type AuditFilter struct {
	Page       int        `json:"page"`
	Limit      int        `json:"limit"`
	UserID     *uuid.UUID `json:"user_id,omitempty"`
	EntityType string     `json:"entity_type,omitempty"`
	EntityID   *uuid.UUID `json:"entity_id,omitempty"`
	Action     string     `json:"action,omitempty"`
}

// AuditLogRepository defines the persistence contract for the audit domain.
type AuditLogRepository interface {
	// Create persists a new audit log entry.
	Create(ctx context.Context, a *AuditLog) error

	// FindAll retrieves a paginated, filtered list of audit log entries.
	// It returns the matching entries, the total count, and any error.
	FindAll(ctx context.Context, filter AuditFilter) ([]AuditLog, int, error)
}
