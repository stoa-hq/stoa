package shipping

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
// Mock ShippingMethodRepository
// ---------------------------------------------------------------------------

type mockShippingRepo struct {
	findByID func(ctx context.Context, id uuid.UUID) (*ShippingMethod, error)
	findAll  func(ctx context.Context, f ShippingMethodFilter) ([]ShippingMethod, int, error)
	create   func(ctx context.Context, m *ShippingMethod) error
	update   func(ctx context.Context, m *ShippingMethod) error
	delete   func(ctx context.Context, id uuid.UUID) error
}

func (m *mockShippingRepo) FindByID(ctx context.Context, id uuid.UUID) (*ShippingMethod, error) {
	if m.findByID != nil {
		return m.findByID(ctx, id)
	}
	return nil, ErrNotFound
}
func (m *mockShippingRepo) FindAll(ctx context.Context, f ShippingMethodFilter) ([]ShippingMethod, int, error) {
	if m.findAll != nil {
		return m.findAll(ctx, f)
	}
	return nil, 0, nil
}
func (m *mockShippingRepo) Create(ctx context.Context, sm *ShippingMethod) error {
	if m.create != nil {
		return m.create(ctx, sm)
	}
	return nil
}
func (m *mockShippingRepo) Update(ctx context.Context, sm *ShippingMethod) error {
	if m.update != nil {
		return m.update(ctx, sm)
	}
	return nil
}
func (m *mockShippingRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.delete != nil {
		return m.delete(ctx, id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func newTestShippingService(repo ShippingMethodRepository) ShippingService {
	noopTaxRate := TaxRateFn(func(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil })
	return NewService(repo, sdk.NewHookRegistry(), zerolog.Nop(), noopTaxRate)
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestShippingService_Create_SetsIDAndTimestamps(t *testing.T) {
	var saved *ShippingMethod
	repo := &mockShippingRepo{
		create: func(_ context.Context, m *ShippingMethod) error {
			saved = m
			return nil
		},
	}
	before := time.Now()
	err := newTestShippingService(repo).Create(context.Background(), &ShippingMethod{PriceNet: 499, PriceGross: 595})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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

func TestShippingService_Create_PropagatesTranslationID(t *testing.T) {
	var saved *ShippingMethod
	repo := &mockShippingRepo{
		create: func(_ context.Context, m *ShippingMethod) error {
			saved = m
			return nil
		},
	}
	m := &ShippingMethod{
		Translations: []ShippingMethodTranslation{
			{Locale: "de-DE", Name: "Standardversand"},
			{Locale: "en-US", Name: "Standard Shipping"},
		},
	}
	if err := newTestShippingService(repo).Create(context.Background(), m); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, tr := range saved.Translations {
		if tr.ShippingMethodID != saved.ID {
			t.Errorf("translation %q: ShippingMethodID = %s, want %s", tr.Locale, tr.ShippingMethodID, saved.ID)
		}
	}
}

func TestShippingService_Create_BeforeHookCancels(t *testing.T) {
	hooks := sdk.NewHookRegistry()
	hookErr := errors.New("hook rejected")
	hooks.On(HookBeforeShippingCreate, func(_ context.Context, _ *sdk.HookEvent) error {
		return hookErr
	})
	noopTaxRate := TaxRateFn(func(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil })
	svc := NewService(&mockShippingRepo{}, hooks, zerolog.Nop(), noopTaxRate)
	err := svc.Create(context.Background(), &ShippingMethod{})
	if !errors.Is(err, hookErr) {
		t.Errorf("expected hookErr, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestShippingService_GetByID_NotFound(t *testing.T) {
	_, err := newTestShippingService(&mockShippingRepo{}).GetByID(context.Background(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestShippingService_GetByID_Found(t *testing.T) {
	id := uuid.New()
	repo := &mockShippingRepo{
		findByID: func(_ context.Context, _ uuid.UUID) (*ShippingMethod, error) {
			return &ShippingMethod{ID: id, PriceGross: 595}, nil
		},
	}
	got, err := newTestShippingService(repo).GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != id {
		t.Errorf("ID: got %s, want %s", got.ID, id)
	}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestShippingService_Update_SetsUpdatedAt(t *testing.T) {
	var saved *ShippingMethod
	repo := &mockShippingRepo{
		update: func(_ context.Context, m *ShippingMethod) error {
			saved = m
			return nil
		},
	}
	before := time.Now()
	m := &ShippingMethod{ID: uuid.New(), PriceNet: 100}
	if err := newTestShippingService(repo).Update(context.Background(), m); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saved.UpdatedAt.Before(before) {
		t.Error("UpdatedAt should be refreshed on update")
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestShippingService_Delete_Success(t *testing.T) {
	deleted := false
	repo := &mockShippingRepo{
		delete: func(_ context.Context, _ uuid.UUID) error {
			deleted = true
			return nil
		},
	}
	if err := newTestShippingService(repo).Delete(context.Background(), uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Error("expected repo.Delete to be called")
	}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestShippingService_List(t *testing.T) {
	methods := []ShippingMethod{{ID: uuid.New()}, {ID: uuid.New()}}
	repo := &mockShippingRepo{
		findAll: func(_ context.Context, _ ShippingMethodFilter) ([]ShippingMethod, int, error) {
			return methods, 2, nil
		},
	}
	got, total, err := newTestShippingService(repo).List(context.Background(), ShippingMethodFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("items: got %d, want 2", len(got))
	}
	if total != 2 {
		t.Errorf("total: got %d, want 2", total)
	}
}
