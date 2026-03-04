package shipping

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/epoxx-arch/stoa/pkg/sdk"
)

// TaxRateFn looks up the integer basis-point tax rate for a given tax rule ID.
type TaxRateFn func(ctx context.Context, id uuid.UUID) (int, error)

func calcNetFromGross(gross, rate int) int {
	return int(math.Round(float64(gross) * 10000 / float64(10000+rate)))
}

func calcGrossFromNet(net, rate int) int {
	return int(math.Round(float64(net) * float64(10000+rate) / 10000))
}

// Sentinel errors for the shipping domain.
var (
	ErrNotFound     = errors.New("shipping method not found")
	ErrInvalidInput = errors.New("invalid input")
)

// Hook name constants for the shipping domain.
const (
	HookBeforeShippingCreate = "shipping.before_create"
	HookAfterShippingCreate  = "shipping.after_create"
	HookBeforeShippingUpdate = "shipping.before_update"
	HookAfterShippingUpdate  = "shipping.after_update"
	HookBeforeShippingDelete = "shipping.before_delete"
	HookAfterShippingDelete  = "shipping.after_delete"
)

// ShippingService defines the business-logic interface for the shipping domain.
type ShippingService interface {
	List(ctx context.Context, filter ShippingMethodFilter) ([]ShippingMethod, int, error)
	Create(ctx context.Context, m *ShippingMethod) error
	GetByID(ctx context.Context, id uuid.UUID) (*ShippingMethod, error)
	Update(ctx context.Context, m *ShippingMethod) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type service struct {
	repo      ShippingMethodRepository
	hooks     *sdk.HookRegistry
	logger    zerolog.Logger
	taxRateFn TaxRateFn
}

// NewService creates a new ShippingService.
// taxRateFn is optional; when non-nil it is used to calculate missing prices from tax rules.
func NewService(repo ShippingMethodRepository, hooks *sdk.HookRegistry, logger zerolog.Logger, taxRateFn TaxRateFn) ShippingService {
	return &service{repo: repo, hooks: hooks, logger: logger, taxRateFn: taxRateFn}
}

func (s *service) List(ctx context.Context, filter ShippingMethodFilter) ([]ShippingMethod, int, error) {
	methods, total, err := s.repo.FindAll(ctx, filter)
	if err != nil {
		s.logger.Error().Err(err).Msg("shipping: List")
		return nil, 0, err
	}
	return methods, total, nil
}

func (s *service) Create(ctx context.Context, m *ShippingMethod) error {
	m.ID = uuid.New()
	now := time.Now().UTC()
	m.CreatedAt = now
	m.UpdatedAt = now

	// Auto-calculate missing price from tax rule.
	if m.TaxRuleID != nil && s.taxRateFn != nil {
		if rate, err := s.taxRateFn(ctx, *m.TaxRuleID); err == nil && rate > 0 {
			if m.PriceGross > 0 && m.PriceNet == 0 {
				m.PriceNet = calcNetFromGross(m.PriceGross, rate)
			} else if m.PriceNet > 0 && m.PriceGross == 0 {
				m.PriceGross = calcGrossFromNet(m.PriceNet, rate)
			}
		}
	}

	// Propagate the method ID to translations.
	for i := range m.Translations {
		m.Translations[i].ShippingMethodID = m.ID
	}

	if s.hooks != nil {
		if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookBeforeShippingCreate,
			Entity: m,
		}); err != nil {
			return fmt.Errorf("shipping: before_create hook: %w", err)
		}
	}

	if err := s.repo.Create(ctx, m); err != nil {
		s.logger.Error().Err(err).Msg("shipping: Create")
		return err
	}

	if s.hooks != nil {
		_ = s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookAfterShippingCreate,
			Entity: m,
		})
	}
	return nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*ShippingMethod, error) {
	m, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			s.logger.Error().Err(err).Msg("shipping: GetByID")
		}
		return nil, err
	}
	return m, nil
}

func (s *service) Update(ctx context.Context, m *ShippingMethod) error {
	m.UpdatedAt = time.Now().UTC()

	// Auto-calculate missing price from tax rule.
	if m.TaxRuleID != nil && s.taxRateFn != nil {
		if rate, err := s.taxRateFn(ctx, *m.TaxRuleID); err == nil && rate > 0 {
			if m.PriceGross > 0 && m.PriceNet == 0 {
				m.PriceNet = calcNetFromGross(m.PriceGross, rate)
			} else if m.PriceNet > 0 && m.PriceGross == 0 {
				m.PriceGross = calcGrossFromNet(m.PriceNet, rate)
			}
		}
	}

	// Propagate the method ID to translations.
	for i := range m.Translations {
		m.Translations[i].ShippingMethodID = m.ID
	}

	if s.hooks != nil {
		if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookBeforeShippingUpdate,
			Entity: m,
		}); err != nil {
			return fmt.Errorf("shipping: before_update hook: %w", err)
		}
	}

	if err := s.repo.Update(ctx, m); err != nil {
		if !errors.Is(err, ErrNotFound) {
			s.logger.Error().Err(err).Msg("shipping: Update")
		}
		return err
	}

	if s.hooks != nil {
		_ = s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookAfterShippingUpdate,
			Entity: m,
		})
	}
	return nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	if s.hooks != nil {
		if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookBeforeShippingDelete,
			Entity: &ShippingMethod{ID: id},
		}); err != nil {
			return fmt.Errorf("shipping: before_delete hook: %w", err)
		}
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		if !errors.Is(err, ErrNotFound) {
			s.logger.Error().Err(err).Msg("shipping: Delete")
		}
		return err
	}

	if s.hooks != nil {
		_ = s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   HookAfterShippingDelete,
			Entity: &ShippingMethod{ID: id},
		})
	}
	return nil
}
