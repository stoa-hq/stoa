package order

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/epoxx-arch/stoa/pkg/sdk"
)

// Service implements business logic for the order domain.
type Service struct {
	repo   OrderRepository
	hooks  *sdk.HookRegistry
	logger zerolog.Logger
}

// NewService creates a new order Service.
func NewService(repo OrderRepository, hooks *sdk.HookRegistry, logger zerolog.Logger) *Service {
	return &Service{
		repo:   repo,
		hooks:  hooks,
		logger: logger,
	}
}

// -------------------------------------------------------------------
// Read operations
// -------------------------------------------------------------------

// GetByID returns the order with the given ID, including items and history.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Order, error) {
	o, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("service get order by id: %w", err)
	}
	return o, nil
}

// List returns a paginated, filtered list of orders and the total count.
func (s *Service) List(ctx context.Context, filter OrderFilter) ([]Order, int, error) {
	orders, total, err := s.repo.FindAll(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("service list orders: %w", err)
	}
	return orders, total, nil
}

// GetByCustomerID returns all orders belonging to a customer.
func (s *Service) GetByCustomerID(ctx context.Context, customerID uuid.UUID) ([]Order, error) {
	orders, err := s.repo.FindByCustomerID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("service get orders by customer: %w", err)
	}
	return orders, nil
}

// -------------------------------------------------------------------
// Write operations
// -------------------------------------------------------------------

// Create persists a new order after firing before/after hooks.
// It assigns a UUID and an order number to the entity before saving.
func (s *Service) Create(ctx context.Context, o *Order) error {
	o.ID = uuid.New()
	o.OrderNumber = s.GenerateOrderNumber()
	if o.Status == "" {
		o.Status = StatusPending
	}

	if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
		Name:   sdk.HookBeforeOrderCreate,
		Entity: o,
	}); err != nil {
		return fmt.Errorf("before_create hook: %w", err)
	}

	if err := s.repo.Create(ctx, o); err != nil {
		return fmt.Errorf("creating order: %w", err)
	}

	if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
		Name:   sdk.HookAfterOrderCreate,
		Entity: o,
	}); err != nil {
		// After-hooks should not roll back the write; log and continue.
		s.logger.Warn().Err(err).Str("order_id", o.ID.String()).Msg("after_create hook returned error")
	}

	return nil
}

// UpdateStatus transitions an order to a new status, validates the transition,
// and fires before/after hooks.
func (s *Service) UpdateStatus(ctx context.Context, id uuid.UUID, toStatus, comment string) error {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("fetching order for status update: %w", err)
	}

	if err := s.ValidateStatusTransition(existing.Status, toStatus); err != nil {
		return err
	}

	changes := map[string]interface{}{
		"from_status": existing.Status,
		"to_status":   toStatus,
		"comment":     comment,
	}

	if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
		Name:    sdk.HookBeforeOrderUpdate,
		Entity:  existing,
		Changes: changes,
	}); err != nil {
		return fmt.Errorf("before_update hook: %w", err)
	}

	if err := s.repo.UpdateStatus(ctx, id, existing.Status, toStatus, comment); err != nil {
		return fmt.Errorf("updating order status: %w", err)
	}

	existing.Status = toStatus

	if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
		Name:    sdk.HookAfterOrderUpdate,
		Entity:  existing,
		Changes: changes,
	}); err != nil {
		s.logger.Warn().Err(err).Str("order_id", id.String()).Msg("after_update hook returned error")
	}

	return nil
}

// -------------------------------------------------------------------
// Business logic helpers
// -------------------------------------------------------------------

// GenerateOrderNumber returns a unique order number in the format
// ORD-YYYYMMDD-XXXXX where XXXXX is a random 5-character alphanumeric suffix.
func (s *Service) GenerateOrderNumber() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	date := time.Now().UTC().Format("20060102")
	suffix := make([]byte, 5)
	for i := range suffix {
		suffix[i] = charset[rand.Intn(len(charset))]
	}
	return fmt.Sprintf("ORD-%s-%s", date, suffix)
}

// ValidateStatusTransition returns an error if transitioning from fromStatus
// to toStatus is not permitted by the order state machine.
func (s *Service) ValidateStatusTransition(fromStatus, toStatus string) error {
	allowed, exists := validTransitions[fromStatus]
	if !exists {
		return fmt.Errorf("unknown order status: %q", fromStatus)
	}
	for _, a := range allowed {
		if a == toStatus {
			return nil
		}
	}
	return fmt.Errorf("invalid status transition from %q to %q", fromStatus, toStatus)
}
