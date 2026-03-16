package warehouse

import (
	"context"

	"github.com/google/uuid"
)

// WarehouseFilter controls the result set returned by FindAll.
type WarehouseFilter struct {
	Page   int   `json:"page"`
	Limit  int   `json:"limit"`
	Active *bool `json:"active,omitempty"`
}

// WarehouseRepository defines the persistence contract for the warehouse domain.
type WarehouseRepository interface {
	// FindByID retrieves a single warehouse by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*Warehouse, error)

	// FindAll retrieves a paginated, filtered list of warehouses.
	FindAll(ctx context.Context, filter WarehouseFilter) ([]Warehouse, int, error)

	// Create persists a new warehouse.
	Create(ctx context.Context, w *Warehouse) error

	// Update persists changes to an existing warehouse.
	Update(ctx context.Context, w *Warehouse) error

	// Delete removes a warehouse by ID.
	Delete(ctx context.Context, id uuid.UUID) error

	// SetStock upserts the stock quantity for a product/variant at a warehouse
	// and records an adjustment movement. Returns the updated WarehouseStock.
	SetStock(ctx context.Context, warehouseID, productID uuid.UUID, variantID *uuid.UUID, quantity int, reference string) (*WarehouseStock, error)

	// GetStockByWarehouse returns all stock entries for a given warehouse.
	GetStockByWarehouse(ctx context.Context, warehouseID uuid.UUID) ([]WarehouseStock, error)

	// GetStockByProduct returns all stock entries across warehouses for a product.
	GetStockByProduct(ctx context.Context, productID uuid.UUID) ([]WarehouseStock, error)

	// DeductStock deducts inventory for order items using priority-based warehouse
	// selection. Records movements and updates denormalized product/variant stock.
	// Returns ErrInsufficientStock if any item cannot be fulfilled.
	DeductStock(ctx context.Context, items []StockDeductionItem) error

	// RestoreStock reverses all sale movements for the given order, restoring
	// inventory and updating denormalized stock fields.
	RestoreStock(ctx context.Context, orderID uuid.UUID) error

	// RemoveStock deletes a stock entry and records an adjustment movement.
	// Also updates the denormalized stock on the product/variant.
	RemoveStock(ctx context.Context, stockID uuid.UUID) error

	// AggregateStock returns the total stock across all active warehouses for
	// a product or variant. Used for stock availability checks.
	AggregateStock(ctx context.Context, productID uuid.UUID, variantID *uuid.UUID) (int, error)

	// AnyWarehouseAllowsNegative returns true when at least one active warehouse
	// that holds stock for the given product/variant has allow_negative_stock=true.
	AnyWarehouseAllowsNegative(ctx context.Context, productID uuid.UUID, variantID *uuid.UUID) (bool, error)
}
