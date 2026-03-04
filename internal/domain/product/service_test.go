package product

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/epoxx-arch/stoa/pkg/sdk"
)

// ---------------------------------------------------------------------------
// Mock repository – shared by service_test.go and handler_test.go
// ---------------------------------------------------------------------------

type mockRepo struct {
	findByID       func(ctx context.Context, id uuid.UUID) (*Product, error)
	findAll        func(ctx context.Context, f ProductFilter) ([]Product, int, error)
	findBySlug     func(ctx context.Context, slug, locale string) (*Product, error)
	create         func(ctx context.Context, p *Product) error
	update         func(ctx context.Context, p *Product) error
	delete         func(ctx context.Context, id uuid.UUID) error
	stockAvailable func(ctx context.Context, productID uuid.UUID, variantID *uuid.UUID, qty int) (bool, error)
}

func (m *mockRepo) FindByID(ctx context.Context, id uuid.UUID) (*Product, error) {
	if m.findByID != nil {
		return m.findByID(ctx, id)
	}
	return nil, ErrNotFound
}
func (m *mockRepo) FindAll(ctx context.Context, f ProductFilter) ([]Product, int, error) {
	if m.findAll != nil {
		return m.findAll(ctx, f)
	}
	return nil, 0, nil
}
func (m *mockRepo) FindBySlug(ctx context.Context, slug, locale string) (*Product, error) {
	if m.findBySlug != nil {
		return m.findBySlug(ctx, slug, locale)
	}
	return nil, ErrNotFound
}
func (m *mockRepo) Create(ctx context.Context, p *Product) error {
	if m.create != nil {
		return m.create(ctx, p)
	}
	return nil
}
func (m *mockRepo) Update(ctx context.Context, p *Product) error {
	if m.update != nil {
		return m.update(ctx, p)
	}
	return nil
}
func (m *mockRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.delete != nil {
		return m.delete(ctx, id)
	}
	return nil
}
func (m *mockRepo) StockAvailable(ctx context.Context, productID uuid.UUID, variantID *uuid.UUID, qty int) (bool, error) {
	if m.stockAvailable != nil {
		return m.stockAvailable(ctx, productID, variantID, qty)
	}
	return true, nil
}

// Variant stubs
func (m *mockRepo) CreateVariant(_ context.Context, _ *ProductVariant) error           { return nil }
func (m *mockRepo) FindVariantByID(_ context.Context, _ uuid.UUID) (*ProductVariant, error) {
	return nil, ErrNotFound
}
func (m *mockRepo) UpdateVariant(_ context.Context, _ *ProductVariant) error { return nil }
func (m *mockRepo) DeleteVariant(_ context.Context, _ uuid.UUID) error       { return nil }

// PropertyGroup stubs
func (m *mockRepo) FindAllPropertyGroups(_ context.Context) ([]PropertyGroup, error)        { return nil, nil }
func (m *mockRepo) FindPropertyGroupByID(_ context.Context, _ uuid.UUID) (*PropertyGroup, error) {
	return nil, ErrNotFound
}
func (m *mockRepo) CreatePropertyGroup(_ context.Context, _ *PropertyGroup) error { return nil }
func (m *mockRepo) UpdatePropertyGroup(_ context.Context, _ *PropertyGroup) error { return nil }
func (m *mockRepo) DeletePropertyGroup(_ context.Context, _ uuid.UUID) error      { return nil }

// PropertyOption stubs
func (m *mockRepo) FindOptionsByGroupID(_ context.Context, _ uuid.UUID) ([]PropertyOption, error) {
	return nil, nil
}
func (m *mockRepo) CreatePropertyOption(_ context.Context, _ *PropertyOption) error { return nil }
func (m *mockRepo) UpdatePropertyOption(_ context.Context, _ *PropertyOption) error { return nil }
func (m *mockRepo) DeletePropertyOption(_ context.Context, _ uuid.UUID) error       { return nil }

// newTestService builds a Service with a no-op HookRegistry and silent logger.
func newTestService(repo ProductRepository) *Service {
	noopURL := func(s string) string { return "/uploads/" + s }
	noopTax := TaxRateFn(func(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil })
	return NewService(repo, sdk.NewHookRegistry(), zerolog.Nop(), noopURL, noopTax)
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestService_GetByID_Found(t *testing.T) {
	id := uuid.New()
	want := &Product{ID: id, SKU: "X"}

	repo := &mockRepo{
		findByID: func(_ context.Context, got uuid.UUID) (*Product, error) {
			if got != id {
				t.Errorf("FindByID: got %s, want %s", got, id)
			}
			return want, nil
		},
	}

	p, err := newTestService(repo).GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ID != id {
		t.Errorf("product ID: got %s, want %s", p.ID, id)
	}
}

func TestService_GetByID_NotFound(t *testing.T) {
	_, err := newTestService(&mockRepo{}).GetByID(context.Background(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestService_Create_Success(t *testing.T) {
	created := false
	repo := &mockRepo{
		create: func(_ context.Context, _ *Product) error {
			created = true
			return nil
		},
	}

	if err := newTestService(repo).Create(context.Background(), &Product{SKU: "NEW"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Error("expected repo.Create to be called")
	}
}

func TestService_Create_BeforeHookCancels(t *testing.T) {
	hooks := sdk.NewHookRegistry()
	hookErr := errors.New("plugin rejected")
	hooks.On(sdk.HookBeforeProductCreate, func(_ context.Context, _ *sdk.HookEvent) error {
		return hookErr
	})

	noopURL := func(s string) string { return "/uploads/" + s }
	noopTax := TaxRateFn(func(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil })
	svc := NewService(&mockRepo{}, hooks, zerolog.Nop(), noopURL, noopTax)
	err := svc.Create(context.Background(), &Product{SKU: "BLOCKED"})
	if !errors.Is(err, hookErr) {
		t.Errorf("expected hookErr to be wrapped in error, got %v", err)
	}
}

func TestService_Create_AfterHookErrorIgnored(t *testing.T) {
	hooks := sdk.NewHookRegistry()
	hooks.On(sdk.HookAfterProductCreate, func(_ context.Context, _ *sdk.HookEvent) error {
		return errors.New("after-hook failure")
	})

	// After-hook errors must not propagate.
	noopURL := func(s string) string { return "/uploads/" + s }
	noopTax := TaxRateFn(func(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil })
	err := NewService(&mockRepo{}, hooks, zerolog.Nop(), noopURL, noopTax).
		Create(context.Background(), &Product{SKU: "OK"})
	if err != nil {
		t.Fatalf("after-hook error should be swallowed, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestService_Delete_NotFound(t *testing.T) {
	// mockRepo.FindByID returns ErrNotFound by default.
	err := newTestService(&mockRepo{}).Delete(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error when deleting non-existent product")
	}
}

func TestService_Delete_Success(t *testing.T) {
	id := uuid.New()
	deleted := false

	repo := &mockRepo{
		findByID: func(_ context.Context, _ uuid.UUID) (*Product, error) {
			return &Product{ID: id}, nil
		},
		delete: func(_ context.Context, _ uuid.UUID) error {
			deleted = true
			return nil
		},
	}

	if err := newTestService(repo).Delete(context.Background(), id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Error("expected repo.Delete to be called")
	}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestService_List_ReturnsPaginatedResults(t *testing.T) {
	products := []Product{{ID: uuid.New()}, {ID: uuid.New()}}

	repo := &mockRepo{
		findAll: func(_ context.Context, _ ProductFilter) ([]Product, int, error) {
			return products, 42, nil
		},
	}

	got, total, err := newTestService(repo).List(context.Background(), ProductFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("items: got %d, want 2", len(got))
	}
	if total != 42 {
		t.Errorf("total: got %d, want 42", total)
	}
}
