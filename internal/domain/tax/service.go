package tax

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/epoxx-arch/stoa/pkg/sdk"
)

// Sentinel errors for the tax domain.
var (
	ErrNotFound     = errors.New("tax rule not found")
	ErrInvalidInput = errors.New("invalid input")
)

// Hook name constants for the tax domain.
const (
	HookBeforeTaxCreate = "tax.before_create"
	HookAfterTaxCreate  = "tax.after_create"
	HookBeforeTaxUpdate = "tax.before_update"
	HookAfterTaxUpdate  = "tax.after_update"
	HookBeforeTaxDelete = "tax.before_delete"
	HookAfterTaxDelete  = "tax.after_delete"
)

// TaxService defines the business-logic interface for the tax domain.
type TaxService interface {
	List(ctx context.Context, filter TaxRuleFilter) ([]TaxRule, int, error)
	Create(ctx context.Context, t *TaxRule) error
	GetByID(ctx context.Context, id uuid.UUID) (*TaxRule, error)
	Update(ctx context.Context, t *TaxRule) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type service struct {
	repo   TaxRuleRepository
	hooks  *sdk.HookRegistry
	logger zerolog.Logger
}

// NewService creates a new TaxService with the given repository and optional hook registry.
func NewService(repo TaxRuleRepository, hooks *sdk.HookRegistry, logger zerolog.Logger) TaxService {
	return &service{repo: repo, hooks: hooks, logger: logger}
}

func (s *service) List(ctx context.Context, filter TaxRuleFilter) ([]TaxRule, int, error) {
	rules, total, err := s.repo.FindAll(ctx, filter)
	if err != nil {
		s.logger.Error().Err(err).Msg("tax: List")
		return nil, 0, err
	}
	return rules, total, nil
}

func (s *service) Create(ctx context.Context, t *TaxRule) error {
	if t.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	t.ID = uuid.New()
	now := time.Now().UTC()
	t.CreatedAt = now
	t.UpdatedAt = now

	if s.hooks != nil {
		if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookBeforeTaxCreate,
			Entity: t,
		}); err != nil {
			return fmt.Errorf("tax: before_create hook: %w", err)
		}
	}

	if err := s.repo.Create(ctx, t); err != nil {
		s.logger.Error().Err(err).Msg("tax: Create")
		return err
	}

	if s.hooks != nil {
		_ = s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookAfterTaxCreate,
			Entity: t,
		})
	}
	return nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*TaxRule, error) {
	t, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			s.logger.Error().Err(err).Msg("tax: GetByID")
		}
		return nil, err
	}
	return t, nil
}

func (s *service) Update(ctx context.Context, t *TaxRule) error {
	if t.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	t.UpdatedAt = time.Now().UTC()

	if s.hooks != nil {
		if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookBeforeTaxUpdate,
			Entity: t,
		}); err != nil {
			return fmt.Errorf("tax: before_update hook: %w", err)
		}
	}

	if err := s.repo.Update(ctx, t); err != nil {
		if !errors.Is(err, ErrNotFound) {
			s.logger.Error().Err(err).Msg("tax: Update")
		}
		return err
	}

	if s.hooks != nil {
		_ = s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookAfterTaxUpdate,
			Entity: t,
		})
	}
	return nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	if s.hooks != nil {
		if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookBeforeTaxDelete,
			Entity: &TaxRule{ID: id},
		}); err != nil {
			return fmt.Errorf("tax: before_delete hook: %w", err)
		}
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		if !errors.Is(err, ErrNotFound) {
			s.logger.Error().Err(err).Msg("tax: Delete")
		}
		return err
	}

	if s.hooks != nil {
		_ = s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookAfterTaxDelete,
			Entity: &TaxRule{ID: id},
		})
	}
	return nil
}
