package cart

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/pkg/sdk"
)

// ---------------------------------------------------------------------------
// Mock CartRepository
// ---------------------------------------------------------------------------

type mockCartRepo struct {
	create         func(ctx context.Context, c *Cart) error
	findByID       func(ctx context.Context, id uuid.UUID) (*Cart, error)
	findItemByID   func(ctx context.Context, itemID uuid.UUID) (*CartItem, error)
	addItem        func(ctx context.Context, item *CartItem) error
	updateItem     func(ctx context.Context, itemID uuid.UUID, quantity int) error
	removeItem     func(ctx context.Context, itemID uuid.UUID) error
}

func (m *mockCartRepo) Create(ctx context.Context, c *Cart) error {
	if m.create != nil {
		return m.create(ctx, c)
	}
	return nil
}
func (m *mockCartRepo) FindByID(ctx context.Context, id uuid.UUID) (*Cart, error) {
	if m.findByID != nil {
		return m.findByID(ctx, id)
	}
	return nil, ErrCartNotFound
}
func (m *mockCartRepo) FindBySessionID(_ context.Context, _ string) (*Cart, error) {
	return nil, ErrCartNotFound
}
func (m *mockCartRepo) FindByCustomerID(_ context.Context, _ uuid.UUID) (*Cart, error) {
	return nil, ErrCartNotFound
}
func (m *mockCartRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockCartRepo) AddItem(ctx context.Context, item *CartItem) error {
	if m.addItem != nil {
		return m.addItem(ctx, item)
	}
	return nil
}
func (m *mockCartRepo) UpdateItem(ctx context.Context, itemID uuid.UUID, qty int) error {
	if m.updateItem != nil {
		return m.updateItem(ctx, itemID, qty)
	}
	return nil
}
func (m *mockCartRepo) RemoveItem(ctx context.Context, itemID uuid.UUID) error {
	if m.removeItem != nil {
		return m.removeItem(ctx, itemID)
	}
	return nil
}
func (m *mockCartRepo) CleanExpired(_ context.Context) error { return nil }
func (m *mockCartRepo) FindItemByID(ctx context.Context, itemID uuid.UUID) (*CartItem, error) {
	if m.findItemByID != nil {
		return m.findItemByID(ctx, itemID)
	}
	return nil, ErrItemNotFound
}

// ---------------------------------------------------------------------------
// Mock stockChecker
// ---------------------------------------------------------------------------

type mockStock struct {
	available bool
	err       error
}

func (m *mockStock) StockAvailable(_ context.Context, _ uuid.UUID, _ *uuid.UUID, _ int) (bool, error) {
	return m.available, m.err
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func newTestCartService(repo CartRepository, stock stockChecker) *CartService {
	return NewCartService(repo, stock, sdk.NewHookRegistry(), zerolog.Nop())
}

// ---------------------------------------------------------------------------
// CreateCart
// ---------------------------------------------------------------------------

func TestCartService_CreateCart(t *testing.T) {
	created := false
	repo := &mockCartRepo{
		create: func(_ context.Context, _ *Cart) error {
			created = true
			return nil
		},
	}

	c, err := newTestCartService(repo, nil).
		CreateCart(context.Background(), "EUR", "session-abc", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Error("expected repo.Create to be called")
	}
	if c.Currency != "EUR" {
		t.Errorf("currency: got %q, want EUR", c.Currency)
	}
	if c.SessionID != "session-abc" {
		t.Errorf("sessionID: got %q, want session-abc", c.SessionID)
	}
	if c.ID == uuid.Nil {
		t.Error("cart ID should be set")
	}
}

func TestCartService_CreateCart_DefaultCurrency(t *testing.T) {
	c, err := newTestCartService(&mockCartRepo{}, nil).
		CreateCart(context.Background(), "", "s", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Currency != "USD" {
		t.Errorf("default currency: got %q, want USD", c.Currency)
	}
}

// ---------------------------------------------------------------------------
// AddItem
// ---------------------------------------------------------------------------

func TestCartService_AddItem_Success(t *testing.T) {
	added := false
	repo := &mockCartRepo{
		addItem: func(_ context.Context, _ *CartItem) error {
			added = true
			return nil
		},
	}

	_, err := newTestCartService(repo, &mockStock{available: true}).
		AddItem(context.Background(), uuid.New(), uuid.New(), nil, 2, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !added {
		t.Error("expected repo.AddItem to be called")
	}
}

func TestCartService_AddItem_InsufficientStock(t *testing.T) {
	_, err := newTestCartService(&mockCartRepo{}, &mockStock{available: false}).
		AddItem(context.Background(), uuid.New(), uuid.New(), nil, 1, nil)
	if !errors.Is(err, ErrInsufficientStock) {
		t.Errorf("expected ErrInsufficientStock, got %v", err)
	}
}

func TestCartService_AddItem_StockCheckError(t *testing.T) {
	stockErr := errors.New("db unreachable")
	_, err := newTestCartService(&mockCartRepo{}, &mockStock{err: stockErr}).
		AddItem(context.Background(), uuid.New(), uuid.New(), nil, 1, nil)
	if err == nil {
		t.Fatal("expected error from stock checker")
	}
}

func TestCartService_AddItem_NilStockChecker(t *testing.T) {
	// nil stock checker → skip validation, always add.
	added := false
	repo := &mockCartRepo{
		addItem: func(_ context.Context, _ *CartItem) error {
			added = true
			return nil
		},
	}

	_, err := newTestCartService(repo, nil).
		AddItem(context.Background(), uuid.New(), uuid.New(), nil, 999, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !added {
		t.Error("item should be added without stock check when checker is nil")
	}
}

func TestCartService_AddItem_ZeroQuantity(t *testing.T) {
	_, err := newTestCartService(&mockCartRepo{}, nil).
		AddItem(context.Background(), uuid.New(), uuid.New(), nil, 0, nil)
	if err == nil {
		t.Fatal("expected error for zero quantity")
	}
}

func TestCartService_AddItem_WithVariant(t *testing.T) {
	variantID := uuid.New()
	var capturedItem *CartItem
	repo := &mockCartRepo{
		addItem: func(_ context.Context, item *CartItem) error {
			capturedItem = item
			return nil
		},
	}

	_, err := newTestCartService(repo, &mockStock{available: true}).
		AddItem(context.Background(), uuid.New(), uuid.New(), &variantID, 1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedItem == nil {
		t.Fatal("expected repo.AddItem to be called")
	}
	if capturedItem.VariantID == nil || *capturedItem.VariantID != variantID {
		t.Errorf("variant ID: got %v, want %s", capturedItem.VariantID, variantID)
	}
}

// ---------------------------------------------------------------------------
// UpdateItemQuantity
// ---------------------------------------------------------------------------

func TestCartService_UpdateItem_ZeroQuantity(t *testing.T) {
	err := newTestCartService(&mockCartRepo{}, nil).
		UpdateItemQuantity(context.Background(), uuid.New(), 0)
	if err == nil {
		t.Fatal("expected error for zero quantity")
	}
}

func TestCartService_UpdateItem_Success(t *testing.T) {
	var updatedQty int
	repo := &mockCartRepo{
		updateItem: func(_ context.Context, _ uuid.UUID, qty int) error {
			updatedQty = qty
			return nil
		},
	}

	if err := newTestCartService(repo, nil).
		UpdateItemQuantity(context.Background(), uuid.New(), 5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updatedQty != 5 {
		t.Errorf("quantity sent to repo: got %d, want 5", updatedQty)
	}
}

func TestCartService_AddItem_ExistingQuantityIncluded(t *testing.T) {
	// Cart already has 3 units of the product; adding 2 more should trigger
	// a stock check for 5 (not just 2).
	productID := uuid.New()
	cartID := uuid.New()
	repo := &mockCartRepo{
		findByID: func(_ context.Context, _ uuid.UUID) (*Cart, error) {
			return &Cart{
				ID: cartID,
				Items: []CartItem{
					{ProductID: productID, Quantity: 3},
				},
			}, nil
		},
		addItem: func(_ context.Context, _ *CartItem) error { return nil },
	}
	stockFn := &capturingStock{available: true}

	svc := NewCartService(repo, stockFn, sdk.NewHookRegistry(), zerolog.Nop())
	_, err := svc.AddItem(context.Background(), cartID, productID, nil, 2, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stockFn.capturedQty != 5 {
		t.Errorf("expected stock check with qty=5 (existing=3 + new=2), got %d", stockFn.capturedQty)
	}
}

func TestCartService_AddItem_ExistingQuantityBlocksAdd(t *testing.T) {
	// Cart already has 8 units; adding 5 more would be 13 which is unavailable.
	productID := uuid.New()
	cartID := uuid.New()
	repo := &mockCartRepo{
		findByID: func(_ context.Context, _ uuid.UUID) (*Cart, error) {
			return &Cart{
				ID:    cartID,
				Items: []CartItem{{ProductID: productID, Quantity: 8}},
			}, nil
		},
	}
	stockFn := &capturingStock{available: false}
	svc := NewCartService(repo, stockFn, sdk.NewHookRegistry(), zerolog.Nop())
	_, err := svc.AddItem(context.Background(), cartID, productID, nil, 5, nil)
	if !errors.Is(err, ErrInsufficientStock) {
		t.Errorf("expected ErrInsufficientStock, got %v", err)
	}
	if stockFn.capturedQty != 13 {
		t.Errorf("expected stock check qty=13, got %d", stockFn.capturedQty)
	}
}

func TestCartService_UpdateItem_InsufficientStock(t *testing.T) {
	itemID := uuid.New()
	productID := uuid.New()
	repo := &mockCartRepo{
		findItemByID: func(_ context.Context, _ uuid.UUID) (*CartItem, error) {
			return &CartItem{ID: itemID, ProductID: productID}, nil
		},
	}
	svc := NewCartService(repo, &mockStock{available: false}, sdk.NewHookRegistry(), zerolog.Nop())
	err := svc.UpdateItemQuantity(context.Background(), itemID, 10)
	if !errors.Is(err, ErrInsufficientStock) {
		t.Errorf("expected ErrInsufficientStock, got %v", err)
	}
}

// capturingStock captures the quantity passed to StockAvailable.
type capturingStock struct {
	capturedQty int
	available   bool
}

func (c *capturingStock) StockAvailable(_ context.Context, _ uuid.UUID, _ *uuid.UUID, qty int) (bool, error) {
	c.capturedQty = qty
	return c.available, nil
}

// ---------------------------------------------------------------------------
// RemoveItem
// ---------------------------------------------------------------------------

func TestCartService_RemoveItem(t *testing.T) {
	removed := false
	repo := &mockCartRepo{
		removeItem: func(_ context.Context, _ uuid.UUID) error {
			removed = true
			return nil
		},
	}

	if err := newTestCartService(repo, nil).
		RemoveItem(context.Background(), uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !removed {
		t.Error("expected repo.RemoveItem to be called")
	}
}
