package order

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/pkg/sdk"
)

// stockDeductor is a narrow interface for deducting/restoring stock at order time.
// Implemented by warehouse.Service.
type stockDeductor interface {
	DeductStock(ctx context.Context, items []StockDeductionItem) error
	RestoreStock(ctx context.Context, orderID uuid.UUID) error
}

// StockDeductionItem describes a single line item for stock deduction.
// The warehouse package has its own identical type; the adapter in app.go bridges them.
type StockDeductionItem struct {
	ProductID uuid.UUID
	VariantID *uuid.UUID
	Quantity  int
	OrderID   uuid.UUID
}

// Service implements business logic for the order domain.
type Service struct {
	repo   OrderRepository
	stock  stockDeductor
	hooks  *sdk.HookRegistry
	logger zerolog.Logger
}

// NewService creates a new order Service.
// stock may be nil; when nil, stock deduction is skipped.
func NewService(repo OrderRepository, stock stockDeductor, hooks *sdk.HookRegistry, logger zerolog.Logger) *Service {
	return &Service{
		repo:   repo,
		stock:  stock,
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

	// Deduct stock for order items.
	if s.stock != nil && len(o.Items) > 0 {
		items := make([]StockDeductionItem, 0, len(o.Items))
		for _, item := range o.Items {
			if item.ProductID == nil {
				continue
			}
			items = append(items, StockDeductionItem{
				ProductID: *item.ProductID,
				VariantID: item.VariantID,
				Quantity:  item.Quantity,
				OrderID:   o.ID,
			})
		}
		if len(items) > 0 {
			if err := s.stock.DeductStock(ctx, items); err != nil {
				// Mark order as cancelled on stock failure.
				_ = s.repo.UpdateStatus(ctx, o.ID, o.Status, StatusCancelled, "insufficient stock")
				return fmt.Errorf("deducting stock: %w", err)
			}
		}
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
		"from_status":       existing.Status,
		"to_status":         toStatus,
		"comment":           comment,
		"payment_reference": existing.PaymentReference,
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

	// Restore stock when order is cancelled or refunded.
	if s.stock != nil && (toStatus == StatusCancelled || toStatus == StatusRefunded) {
		if err := s.stock.RestoreStock(ctx, id); err != nil {
			s.logger.Error().Err(err).Str("order_id", id.String()).Msg("failed to restore stock")
		}
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

// DispatchHook dispatches a named hook event through the hook registry.
func (s *Service) DispatchHook(ctx context.Context, event string, entity interface{}) error {
	return s.hooks.Dispatch(ctx, &sdk.HookEvent{
		Name:   event,
		Entity: entity,
	})
}

// DispatchHookWithMetadata dispatches a hook event with additional metadata.
func (s *Service) DispatchHookWithMetadata(ctx context.Context, event string, entity interface{}, metadata map[string]interface{}) error {
	return s.hooks.Dispatch(ctx, &sdk.HookEvent{
		Name:     event,
		Entity:   entity,
		Metadata: metadata,
	})
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
