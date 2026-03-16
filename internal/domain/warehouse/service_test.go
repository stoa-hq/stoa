package warehouse

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/pkg/sdk"
)

// ---------------------------------------------------------------------------
// Mock WarehouseRepository
// ---------------------------------------------------------------------------

type mockWarehouseRepo struct {
	findByID                   func(ctx context.Context, id uuid.UUID) (*Warehouse, error)
	findAll                    func(ctx context.Context, filter WarehouseFilter) ([]Warehouse, int, error)
	create                     func(ctx context.Context, w *Warehouse) error
	update                     func(ctx context.Context, w *Warehouse) error
	delete                     func(ctx context.Context, id uuid.UUID) error
	setStock                   func(ctx context.Context, warehouseID, productID uuid.UUID, variantID *uuid.UUID, quantity int, reference string) (*WarehouseStock, error)
	getStockByWarehouse        func(ctx context.Context, warehouseID uuid.UUID) ([]WarehouseStock, error)
	getStockByProduct          func(ctx context.Context, productID uuid.UUID) ([]WarehouseStock, error)
	removeStock                func(ctx context.Context, stockID uuid.UUID) error
	deductStock                func(ctx context.Context, items []StockDeductionItem) error
	restoreStock               func(ctx context.Context, orderID uuid.UUID) error
	aggregateStock             func(ctx context.Context, productID uuid.UUID, variantID *uuid.UUID) (int, error)
	anyWarehouseAllowsNegative func(ctx context.Context, productID uuid.UUID, variantID *uuid.UUID) (bool, error)
}

func (m *mockWarehouseRepo) FindByID(ctx context.Context, id uuid.UUID) (*Warehouse, error) {
	if m.findByID != nil {
		return m.findByID(ctx, id)
	}
	return nil, ErrNotFound
}

func (m *mockWarehouseRepo) FindAll(ctx context.Context, filter WarehouseFilter) ([]Warehouse, int, error) {
	if m.findAll != nil {
		return m.findAll(ctx, filter)
	}
	return nil, 0, nil
}

func (m *mockWarehouseRepo) Create(ctx context.Context, w *Warehouse) error {
	if m.create != nil {
		return m.create(ctx, w)
	}
	return nil
}

func (m *mockWarehouseRepo) Update(ctx context.Context, w *Warehouse) error {
	if m.update != nil {
		return m.update(ctx, w)
	}
	return nil
}

func (m *mockWarehouseRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.delete != nil {
		return m.delete(ctx, id)
	}
	return nil
}

func (m *mockWarehouseRepo) SetStock(ctx context.Context, warehouseID, productID uuid.UUID, variantID *uuid.UUID, quantity int, reference string) (*WarehouseStock, error) {
	if m.setStock != nil {
		return m.setStock(ctx, warehouseID, productID, variantID, quantity, reference)
	}
	return &WarehouseStock{ID: uuid.New(), Quantity: quantity}, nil
}

func (m *mockWarehouseRepo) GetStockByWarehouse(ctx context.Context, warehouseID uuid.UUID) ([]WarehouseStock, error) {
	if m.getStockByWarehouse != nil {
		return m.getStockByWarehouse(ctx, warehouseID)
	}
	return nil, nil
}

func (m *mockWarehouseRepo) GetStockByProduct(ctx context.Context, productID uuid.UUID) ([]WarehouseStock, error) {
	if m.getStockByProduct != nil {
		return m.getStockByProduct(ctx, productID)
	}
	return nil, nil
}

func (m *mockWarehouseRepo) RemoveStock(ctx context.Context, stockID uuid.UUID) error {
	if m.removeStock != nil {
		return m.removeStock(ctx, stockID)
	}
	return nil
}

func (m *mockWarehouseRepo) DeductStock(ctx context.Context, items []StockDeductionItem) error {
	if m.deductStock != nil {
		return m.deductStock(ctx, items)
	}
	return nil
}

func (m *mockWarehouseRepo) RestoreStock(ctx context.Context, orderID uuid.UUID) error {
	if m.restoreStock != nil {
		return m.restoreStock(ctx, orderID)
	}
	return nil
}

func (m *mockWarehouseRepo) AggregateStock(ctx context.Context, productID uuid.UUID, variantID *uuid.UUID) (int, error) {
	if m.aggregateStock != nil {
		return m.aggregateStock(ctx, productID, variantID)
	}
	return 0, nil
}

func (m *mockWarehouseRepo) AnyWarehouseAllowsNegative(ctx context.Context, productID uuid.UUID, variantID *uuid.UUID) (bool, error) {
	if m.anyWarehouseAllowsNegative != nil {
		return m.anyWarehouseAllowsNegative(ctx, productID, variantID)
	}
	return false, nil
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func newTestService(repo WarehouseRepository) *Service {
	return NewService(repo, sdk.NewHookRegistry(), zerolog.Nop())
}

// ---------------------------------------------------------------------------
// Warehouse CRUD Tests
// ---------------------------------------------------------------------------

func TestService_Create_SetsIDAndTimestamps(t *testing.T) {
	var saved *Warehouse
	repo := &mockWarehouseRepo{
		create: func(_ context.Context, w *Warehouse) error {
			saved = w
			return nil
		},
	}
	before := time.Now()
	w := &Warehouse{Name: "Test WH", Code: "TEST", Active: true}
	if err := newTestService(repo).Create(context.Background(), w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saved == nil {
		t.Fatal("expected repo.Create to be called")
	}
	if saved.ID == uuid.Nil {
		t.Error("ID should be set")
	}
	if saved.CreatedAt.Before(before) {
		t.Error("CreatedAt should be set")
	}
	if saved.UpdatedAt.Before(before) {
		t.Error("UpdatedAt should be set")
	}
}

func TestService_Create_DuplicateCode(t *testing.T) {
	repo := &mockWarehouseRepo{
		create: func(_ context.Context, _ *Warehouse) error {
			return ErrDuplicateCode
		},
	}
	err := newTestService(repo).Create(context.Background(), &Warehouse{Name: "WH", Code: "DUP"})
	if !errors.Is(err, ErrDuplicateCode) {
		t.Errorf("expected ErrDuplicateCode, got %v", err)
	}
}

func TestService_GetByID_NotFound(t *testing.T) {
	_, err := newTestService(&mockWarehouseRepo{}).GetByID(context.Background(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_GetByID_Success(t *testing.T) {
	id := uuid.New()
	repo := &mockWarehouseRepo{
		findByID: func(_ context.Context, _ uuid.UUID) (*Warehouse, error) {
			return &Warehouse{ID: id, Name: "WH1", Code: "WH1"}, nil
		},
	}
	w, err := newTestService(repo).GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.ID != id {
		t.Errorf("ID: got %s, want %s", w.ID, id)
	}
}

func TestService_Update_SetsUpdatedAt(t *testing.T) {
	var saved *Warehouse
	repo := &mockWarehouseRepo{
		update: func(_ context.Context, w *Warehouse) error {
			saved = w
			return nil
		},
	}
	before := time.Now()
	w := &Warehouse{ID: uuid.New(), Name: "Updated", Code: "UPD"}
	if err := newTestService(repo).Update(context.Background(), w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saved.UpdatedAt.Before(before) {
		t.Error("UpdatedAt should be set")
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	repo := &mockWarehouseRepo{
		delete: func(_ context.Context, _ uuid.UUID) error {
			return ErrNotFound
		},
	}
	err := newTestService(repo).Delete(context.Background(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_List_PassesFilter(t *testing.T) {
	var capturedFilter WarehouseFilter
	repo := &mockWarehouseRepo{
		findAll: func(_ context.Context, f WarehouseFilter) ([]Warehouse, int, error) {
			capturedFilter = f
			return []Warehouse{{Name: "WH1"}}, 1, nil
		},
	}
	active := true
	filter := WarehouseFilter{Page: 2, Limit: 10, Active: &active}
	warehouses, total, err := newTestService(repo).List(context.Background(), filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total=1, got %d", total)
	}
	if len(warehouses) != 1 {
		t.Errorf("expected 1 warehouse, got %d", len(warehouses))
	}
	if capturedFilter.Page != 2 {
		t.Errorf("expected page=2, got %d", capturedFilter.Page)
	}
}

// ---------------------------------------------------------------------------
// Hook Tests
// ---------------------------------------------------------------------------

func TestService_Create_BeforeHookCancels(t *testing.T) {
	hooks := sdk.NewHookRegistry()
	hookErr := errors.New("hook rejected")
	hooks.On(sdk.HookBeforeWarehouseCreate, func(_ context.Context, _ *sdk.HookEvent) error {
		return hookErr
	})
	svc := NewService(&mockWarehouseRepo{}, hooks, zerolog.Nop())
	err := svc.Create(context.Background(), &Warehouse{Name: "WH", Code: "WH"})
	if !errors.Is(err, hookErr) {
		t.Errorf("expected hookErr, got %v", err)
	}
}

func TestService_Delete_BeforeHookCancels(t *testing.T) {
	hooks := sdk.NewHookRegistry()
	hookErr := errors.New("hook rejected")
	hooks.On(sdk.HookBeforeWarehouseDelete, func(_ context.Context, _ *sdk.HookEvent) error {
		return hookErr
	})
	svc := NewService(&mockWarehouseRepo{}, hooks, zerolog.Nop())
	err := svc.Delete(context.Background(), uuid.New())
	if !errors.Is(err, hookErr) {
		t.Errorf("expected hookErr, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Stock Management Tests
// ---------------------------------------------------------------------------

func TestService_SetStock_Success(t *testing.T) {
	whID := uuid.New()
	pID := uuid.New()
	var capturedQty int
	repo := &mockWarehouseRepo{
		setStock: func(_ context.Context, _, _ uuid.UUID, _ *uuid.UUID, qty int, _ string) (*WarehouseStock, error) {
			capturedQty = qty
			return &WarehouseStock{ID: uuid.New(), Quantity: qty}, nil
		},
	}
	ws, err := newTestService(repo).SetStock(context.Background(), whID, pID, nil, 50, "initial stock")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ws.Quantity != 50 {
		t.Errorf("quantity: got %d, want 50", ws.Quantity)
	}
	if capturedQty != 50 {
		t.Errorf("captured quantity: got %d, want 50", capturedQty)
	}
}

func TestService_StockAvailable_Sufficient(t *testing.T) {
	pID := uuid.New()
	repo := &mockWarehouseRepo{
		aggregateStock: func(_ context.Context, _ uuid.UUID, _ *uuid.UUID) (int, error) {
			return 100, nil
		},
	}
	ok, err := newTestService(repo).StockAvailable(context.Background(), pID, nil, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected stock to be available")
	}
}

func TestService_StockAvailable_Insufficient(t *testing.T) {
	pID := uuid.New()
	repo := &mockWarehouseRepo{
		aggregateStock: func(_ context.Context, _ uuid.UUID, _ *uuid.UUID) (int, error) {
			return 5, nil
		},
	}
	ok, err := newTestService(repo).StockAvailable(context.Background(), pID, nil, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected stock to be insufficient")
	}
}

func TestService_StockAvailable_WithVariant(t *testing.T) {
	pID := uuid.New()
	vID := uuid.New()
	var capturedVariantID *uuid.UUID
	repo := &mockWarehouseRepo{
		aggregateStock: func(_ context.Context, _ uuid.UUID, variantID *uuid.UUID) (int, error) {
			capturedVariantID = variantID
			return 10, nil
		},
	}
	ok, err := newTestService(repo).StockAvailable(context.Background(), pID, &vID, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected stock to be available")
	}
	if capturedVariantID == nil || *capturedVariantID != vID {
		t.Error("expected variant ID to be passed through")
	}
}

// ---------------------------------------------------------------------------
// DeductStock / RestoreStock
// ---------------------------------------------------------------------------

func TestService_DeductStock_Success(t *testing.T) {
	var capturedItems []StockDeductionItem
	repo := &mockWarehouseRepo{
		deductStock: func(_ context.Context, items []StockDeductionItem) error {
			capturedItems = items
			return nil
		},
	}
	items := []StockDeductionItem{
		{ProductID: uuid.New(), Quantity: 2, OrderID: uuid.New()},
	}
	if err := newTestService(repo).DeductStock(context.Background(), items); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(capturedItems) != 1 {
		t.Fatalf("expected 1 item, got %d", len(capturedItems))
	}
}

func TestService_DeductStock_InsufficientStock(t *testing.T) {
	repo := &mockWarehouseRepo{
		deductStock: func(_ context.Context, _ []StockDeductionItem) error {
			return ErrInsufficientStock
		},
	}
	err := newTestService(repo).DeductStock(context.Background(), []StockDeductionItem{
		{ProductID: uuid.New(), Quantity: 999, OrderID: uuid.New()},
	})
	if !errors.Is(err, ErrInsufficientStock) {
		t.Errorf("expected ErrInsufficientStock, got %v", err)
	}
}

func TestService_RestoreStock_Success(t *testing.T) {
	orderID := uuid.New()
	var capturedOrderID uuid.UUID
	repo := &mockWarehouseRepo{
		restoreStock: func(_ context.Context, id uuid.UUID) error {
			capturedOrderID = id
			return nil
		},
	}
	if err := newTestService(repo).RestoreStock(context.Background(), orderID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedOrderID != orderID {
		t.Errorf("order_id: got %s, want %s", capturedOrderID, orderID)
	}
}

// ---------------------------------------------------------------------------
// StockAvailable with allow_negative_stock
// ---------------------------------------------------------------------------

func TestService_StockAvailable_AllowNegative_SkipsAggregateCheck(t *testing.T) {
	pID := uuid.New()
	aggregateCalled := false
	repo := &mockWarehouseRepo{
		anyWarehouseAllowsNegative: func(_ context.Context, _ uuid.UUID, _ *uuid.UUID) (bool, error) {
			return true, nil
		},
		aggregateStock: func(_ context.Context, _ uuid.UUID, _ *uuid.UUID) (int, error) {
			aggregateCalled = true
			return 0, nil // would be insufficient, but should not be reached
		},
	}
	ok, err := newTestService(repo).StockAvailable(context.Background(), pID, nil, 9999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected stock to be available when allow_negative_stock=true")
	}
	if aggregateCalled {
		t.Error("AggregateStock should not be called when allow_negative_stock=true")
	}
}

func TestService_StockAvailable_AllowNegativeFalse_UsesAggregateCheck(t *testing.T) {
	pID := uuid.New()
	repo := &mockWarehouseRepo{
		anyWarehouseAllowsNegative: func(_ context.Context, _ uuid.UUID, _ *uuid.UUID) (bool, error) {
			return false, nil
		},
		aggregateStock: func(_ context.Context, _ uuid.UUID, _ *uuid.UUID) (int, error) {
			return 5, nil
		},
	}
	ok, err := newTestService(repo).StockAvailable(context.Background(), pID, nil, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected stock to be insufficient when allow_negative_stock=false and total<requested")
	}
}
