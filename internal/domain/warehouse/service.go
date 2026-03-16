package warehouse

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/pkg/sdk"
)

// Service implements business logic for the warehouse domain.
type Service struct {
	repo   WarehouseRepository
	hooks  *sdk.HookRegistry
	logger zerolog.Logger
}

// NewService creates a new warehouse Service.
func NewService(repo WarehouseRepository, hooks *sdk.HookRegistry, logger zerolog.Logger) *Service {
	return &Service{
		repo:   repo,
		hooks:  hooks,
		logger: logger,
	}
}

// -------------------------------------------------------------------
// Warehouse CRUD
// -------------------------------------------------------------------

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Warehouse, error) {
	w, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("warehouse service get by id: %w", err)
	}
	return w, nil
}

func (s *Service) List(ctx context.Context, filter WarehouseFilter) ([]Warehouse, int, error) {
	warehouses, total, err := s.repo.FindAll(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("warehouse service list: %w", err)
	}
	return warehouses, total, nil
}

func (s *Service) Create(ctx context.Context, w *Warehouse) error {
	w.ID = uuid.New()
	now := time.Now().UTC()
	w.CreatedAt = now
	w.UpdatedAt = now

	if s.hooks != nil {
		if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   sdk.HookBeforeWarehouseCreate,
			Entity: w,
		}); err != nil {
			return fmt.Errorf("warehouse: before_create hook: %w", err)
		}
	}

	if err := s.repo.Create(ctx, w); err != nil {
		return fmt.Errorf("warehouse: create: %w", err)
	}

	if s.hooks != nil {
		_ = s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   sdk.HookAfterWarehouseCreate,
			Entity: w,
		})
	}
	return nil
}

func (s *Service) Update(ctx context.Context, w *Warehouse) error {
	w.UpdatedAt = time.Now().UTC()

	if s.hooks != nil {
		if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   sdk.HookBeforeWarehouseUpdate,
			Entity: w,
		}); err != nil {
			return fmt.Errorf("warehouse: before_update hook: %w", err)
		}
	}

	if err := s.repo.Update(ctx, w); err != nil {
		return fmt.Errorf("warehouse: update: %w", err)
	}

	if s.hooks != nil {
		_ = s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   sdk.HookAfterWarehouseUpdate,
			Entity: w,
		})
	}
	return nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if s.hooks != nil {
		if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   sdk.HookBeforeWarehouseDelete,
			Entity: &Warehouse{ID: id},
		}); err != nil {
			return fmt.Errorf("warehouse: before_delete hook: %w", err)
		}
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("warehouse: delete: %w", err)
	}

	if s.hooks != nil {
		_ = s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   sdk.HookAfterWarehouseDelete,
			Entity: &Warehouse{ID: id},
		})
	}
	return nil
}

// -------------------------------------------------------------------
// Stock management
// -------------------------------------------------------------------

func (s *Service) SetStock(ctx context.Context, warehouseID, productID uuid.UUID, variantID *uuid.UUID, quantity int, reference string) (*WarehouseStock, error) {
	if s.hooks != nil {
		if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name: sdk.HookBeforeStockUpdate,
			Metadata: map[string]interface{}{
				"warehouse_id": warehouseID,
				"product_id":   productID,
				"variant_id":   variantID,
				"quantity":     quantity,
			},
		}); err != nil {
			return nil, fmt.Errorf("warehouse: before_stock_update hook: %w", err)
		}
	}

	ws, err := s.repo.SetStock(ctx, warehouseID, productID, variantID, quantity, reference)
	if err != nil {
		return nil, fmt.Errorf("warehouse: set stock: %w", err)
	}

	if s.hooks != nil {
		_ = s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name:   sdk.HookAfterStockUpdate,
			Entity: ws,
		})
	}
	return ws, nil
}

func (s *Service) RemoveStock(ctx context.Context, stockID uuid.UUID) error {
	return s.repo.RemoveStock(ctx, stockID)
}

func (s *Service) GetStockByWarehouse(ctx context.Context, warehouseID uuid.UUID) ([]WarehouseStock, error) {
	return s.repo.GetStockByWarehouse(ctx, warehouseID)
}

func (s *Service) GetStockByProduct(ctx context.Context, productID uuid.UUID) ([]WarehouseStock, error) {
	return s.repo.GetStockByProduct(ctx, productID)
}

// -------------------------------------------------------------------
// stockChecker interface (for Cart service)
// -------------------------------------------------------------------

// StockAvailable returns true when the given product (or variant) has at least
// the requested quantity across all active warehouses. Returns true unconditionally
// when any warehouse allows negative stock for this product.
func (s *Service) StockAvailable(ctx context.Context, productID uuid.UUID, variantID *uuid.UUID, quantity int) (bool, error) {
	allowed, err := s.repo.AnyWarehouseAllowsNegative(ctx, productID, variantID)
	if err != nil {
		return false, err
	}
	if allowed {
		return true, nil
	}
	total, err := s.repo.AggregateStock(ctx, productID, variantID)
	if err != nil {
		return false, err
	}
	return total >= quantity, nil
}

// -------------------------------------------------------------------
// stockDeductor interface (for Order service)
// -------------------------------------------------------------------

// DeductStock deducts inventory for order line items using priority-based
// warehouse selection.
func (s *Service) DeductStock(ctx context.Context, items []StockDeductionItem) error {
	if err := s.repo.DeductStock(ctx, items); err != nil {
		return err
	}

	if s.hooks != nil {
		_ = s.hooks.Dispatch(ctx, &sdk.HookEvent{
			Name: sdk.HookAfterStockDeduct,
			Metadata: map[string]interface{}{
				"items": items,
			},
		})
	}
	return nil
}

// RestoreStock reverses all sale movements for a given order.
func (s *Service) RestoreStock(ctx context.Context, orderID uuid.UUID) error {
	return s.repo.RestoreStock(ctx, orderID)
}
