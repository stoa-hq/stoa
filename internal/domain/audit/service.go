package audit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Sentinel errors for the audit domain.
var (
	ErrInvalidInput = errors.New("invalid input")
)

// AuditService defines the business-logic interface for the audit domain.
type AuditService interface {
	// Log persists a new audit log entry.
	Log(ctx context.Context, a *AuditLog) error

	// List retrieves a paginated, filtered list of audit log entries.
	List(ctx context.Context, filter AuditFilter) ([]AuditLog, int, error)
}

type service struct {
	repo   AuditLogRepository
	logger zerolog.Logger
}

// NewService creates a new AuditService with the given repository.
func NewService(repo AuditLogRepository, logger zerolog.Logger) AuditService {
	return &service{repo: repo, logger: logger}
}

func (s *service) Log(ctx context.Context, a *AuditLog) error {
	if a.Action == "" {
		return fmt.Errorf("%w: action is required", ErrInvalidInput)
	}
	if a.EntityType == "" {
		return fmt.Errorf("%w: entity_type is required", ErrInvalidInput)
	}
	a.ID = uuid.New()
	a.CreatedAt = time.Now().UTC()

	if err := s.repo.Create(ctx, a); err != nil {
		s.logger.Error().Err(err).Msg("audit: Log")
		return err
	}
	return nil
}

func (s *service) List(ctx context.Context, filter AuditFilter) ([]AuditLog, int, error) {
	logs, total, err := s.repo.FindAll(ctx, filter)
	if err != nil {
		s.logger.Error().Err(err).Msg("audit: List")
		return nil, 0, err
	}
	return logs, total, nil
}
