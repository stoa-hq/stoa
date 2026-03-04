package discount

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/epoxx-arch/stoa/pkg/sdk"
)

// Sentinel errors for the discount domain.
var (
	ErrNotFound      = errors.New("discount not found")
	ErrInvalidInput  = errors.New("invalid input")
	ErrCodeInvalid   = errors.New("discount code is invalid or expired")
	ErrMaxUsesReached = errors.New("discount maximum uses reached")
)

// Hook name constants for the discount domain.
const (
	HookBeforeDiscountCreate = "discount.before_create"
	HookAfterDiscountCreate  = "discount.after_create"
	HookBeforeDiscountUpdate = "discount.before_update"
	HookAfterDiscountUpdate  = "discount.after_update"
	HookBeforeDiscountDelete = "discount.before_delete"
	HookAfterDiscountDelete  = "discount.after_delete"
)

// DiscountService defines the business-logic interface for the discount domain.
type DiscountService interface {
	List(ctx context.Context, filter DiscountFilter) ([]Discount, int, error)
	Create(ctx context.Context, d *Discount) error
	GetByID(ctx context.Context, id uuid.UUID) (*Discount, error)
	Update(ctx context.Context, d *Discount) error
	Delete(ctx context.Context, id uuid.UUID) error

	// ValidateCode checks whether a discount code is applicable to the given order total.
	// Returns the discount if valid, or an error describing why it is not.
	ValidateCode(ctx context.Context, code string, orderTotal int) (*Discount, error)

	// ApplyDiscount increments the used_count on the given discount.
	ApplyDiscount(ctx context.Context, id uuid.UUID) error
}

type service struct {
	repo   DiscountRepository
	hooks  *sdk.HookRegistry
	logger zerolog.Logger
}

// NewService creates a new DiscountService.
func NewService(repo DiscountRepository, hooks *sdk.HookRegistry, logger zerolog.Logger) DiscountService {
	return &service{repo: repo, hooks: hooks, logger: logger}
}

func (s *service) List(ctx context.Context, filter DiscountFilter) ([]Discount, int, error) {
	discounts, total, err := s.repo.FindAll(ctx, filter)
	if err != nil {
		s.logger.Error().Err(err).Msg("discount: List")
		return nil, 0, err
	}
	return discounts, total, nil
}

func (s *service) Create(ctx context.Context, d *Discount) error {
	if d.Code == "" {
		return fmt.Errorf("%w: code is required", ErrInvalidInput)
	}
	if d.Type != "percentage" && d.Type != "fixed" {
		return fmt.Errorf("%w: type must be percentage or fixed", ErrInvalidInput)
	}
	d.ID = uuid.New()
	now := time.Now().UTC()
	d.CreatedAt = now
	d.UpdatedAt = now

	if s.hooks != nil {
		if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookBeforeDiscountCreate,
			Entity: d,
		}); err != nil {
			return fmt.Errorf("discount: before_create hook: %w", err)
		}
	}

	if err := s.repo.Create(ctx, d); err != nil {
		s.logger.Error().Err(err).Msg("discount: Create")
		return err
	}

	if s.hooks != nil {
		_ = s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookAfterDiscountCreate,
			Entity: d,
		})
	}
	return nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*Discount, error) {
	d, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			s.logger.Error().Err(err).Msg("discount: GetByID")
		}
		return nil, err
	}
	return d, nil
}

func (s *service) Update(ctx context.Context, d *Discount) error {
	if d.Code == "" {
		return fmt.Errorf("%w: code is required", ErrInvalidInput)
	}
	if d.Type != "percentage" && d.Type != "fixed" {
		return fmt.Errorf("%w: type must be percentage or fixed", ErrInvalidInput)
	}
	d.UpdatedAt = time.Now().UTC()

	if s.hooks != nil {
		if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookBeforeDiscountUpdate,
			Entity: d,
		}); err != nil {
			return fmt.Errorf("discount: before_update hook: %w", err)
		}
	}

	if err := s.repo.Update(ctx, d); err != nil {
		if !errors.Is(err, ErrNotFound) {
			s.logger.Error().Err(err).Msg("discount: Update")
		}
		return err
	}

	if s.hooks != nil {
		_ = s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookAfterDiscountUpdate,
			Entity: d,
		})
	}
	return nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	if s.hooks != nil {
		if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookBeforeDiscountDelete,
			Entity: &Discount{ID: id},
		}); err != nil {
			return fmt.Errorf("discount: before_delete hook: %w", err)
		}
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		if !errors.Is(err, ErrNotFound) {
			s.logger.Error().Err(err).Msg("discount: Delete")
		}
		return err
	}

	if s.hooks != nil {
		_ = s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookAfterDiscountDelete,
			Entity: &Discount{ID: id},
		})
	}
	return nil
}

func (s *service) ValidateCode(ctx context.Context, code string, orderTotal int) (*Discount, error) {
	d, err := s.repo.FindByCode(ctx, code)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrCodeInvalid
		}
		s.logger.Error().Err(err).Msg("discount: ValidateCode")
		return nil, err
	}

	if !d.Active {
		return nil, ErrCodeInvalid
	}

	now := time.Now().UTC()
	if d.ValidFrom != nil && now.Before(*d.ValidFrom) {
		return nil, ErrCodeInvalid
	}
	if d.ValidUntil != nil && now.After(*d.ValidUntil) {
		return nil, ErrCodeInvalid
	}
	if d.MaxUses != nil && d.UsedCount >= *d.MaxUses {
		return nil, ErrMaxUsesReached
	}
	if d.MinOrderValue != nil && orderTotal < *d.MinOrderValue {
		return nil, fmt.Errorf("%w: minimum order value not met", ErrInvalidInput)
	}

	return d, nil
}

func (s *service) ApplyDiscount(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.IncrementUsedCount(ctx, id); err != nil {
		if !errors.Is(err, ErrNotFound) {
			s.logger.Error().Err(err).Msg("discount: ApplyDiscount")
		}
		return err
	}
	return nil
}
