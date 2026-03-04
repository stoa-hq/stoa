package category

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/epoxx-arch/stoa/pkg/sdk"
)

// ---------------------------------------------------------------------------
// Mock CategoryRepository
// ---------------------------------------------------------------------------

type mockCategoryRepo struct {
	findByID  func(ctx context.Context, id uuid.UUID) (*Category, error)
	findAll   func(ctx context.Context, f CategoryFilter) ([]Category, int, error)
	findTree  func(ctx context.Context, locale string) ([]Category, error)
	findBySlug func(ctx context.Context, slug, locale string) (*Category, error)
	create    func(ctx context.Context, c *Category) error
	update    func(ctx context.Context, c *Category) error
	delete    func(ctx context.Context, id uuid.UUID) error
}

func (m *mockCategoryRepo) FindByID(ctx context.Context, id uuid.UUID) (*Category, error) {
	if m.findByID != nil {
		return m.findByID(ctx, id)
	}
	return nil, ErrNotFound
}
func (m *mockCategoryRepo) FindAll(ctx context.Context, f CategoryFilter) ([]Category, int, error) {
	if m.findAll != nil {
		return m.findAll(ctx, f)
	}
	return nil, 0, nil
}
func (m *mockCategoryRepo) FindTree(ctx context.Context, locale string) ([]Category, error) {
	if m.findTree != nil {
		return m.findTree(ctx, locale)
	}
	return nil, nil
}
func (m *mockCategoryRepo) FindBySlug(ctx context.Context, slug, locale string) (*Category, error) {
	if m.findBySlug != nil {
		return m.findBySlug(ctx, slug, locale)
	}
	return nil, ErrNotFound
}
func (m *mockCategoryRepo) Create(ctx context.Context, c *Category) error {
	if m.create != nil {
		return m.create(ctx, c)
	}
	return nil
}
func (m *mockCategoryRepo) Update(ctx context.Context, c *Category) error {
	if m.update != nil {
		return m.update(ctx, c)
	}
	return nil
}
func (m *mockCategoryRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.delete != nil {
		return m.delete(ctx, id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func newTestCategoryService(repo CategoryRepository) *Service {
	return NewService(repo, sdk.NewHookRegistry(), zerolog.Nop())
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestCategoryService_GetByID_Found(t *testing.T) {
	id := uuid.New()
	repo := &mockCategoryRepo{
		findByID: func(_ context.Context, got uuid.UUID) (*Category, error) {
			return &Category{ID: got}, nil
		},
	}
	cat, err := newTestCategoryService(repo).GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cat.ID != id {
		t.Errorf("ID: got %s, want %s", cat.ID, id)
	}
}

func TestCategoryService_GetByID_NotFound(t *testing.T) {
	_, err := newTestCategoryService(&mockCategoryRepo{}).GetByID(context.Background(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestCategoryService_List(t *testing.T) {
	cats := []Category{{ID: uuid.New()}, {ID: uuid.New()}}
	repo := &mockCategoryRepo{
		findAll: func(_ context.Context, _ CategoryFilter) ([]Category, int, error) {
			return cats, 10, nil
		},
	}
	got, total, err := newTestCategoryService(repo).List(context.Background(), CategoryFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("items: got %d, want 2", len(got))
	}
	if total != 10 {
		t.Errorf("total: got %d, want 10", total)
	}
}

// ---------------------------------------------------------------------------
// GetTree
// ---------------------------------------------------------------------------

func TestCategoryService_GetTree_DefaultLocale(t *testing.T) {
	var capturedLocale string
	repo := &mockCategoryRepo{
		findTree: func(_ context.Context, locale string) ([]Category, error) {
			capturedLocale = locale
			return nil, nil
		},
	}
	_, err := newTestCategoryService(repo).GetTree(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedLocale != "de-DE" {
		t.Errorf("locale: got %q, want %q", capturedLocale, "de-DE")
	}
}

func TestCategoryService_GetTree_CustomLocale(t *testing.T) {
	var capturedLocale string
	repo := &mockCategoryRepo{
		findTree: func(_ context.Context, locale string) ([]Category, error) {
			capturedLocale = locale
			return []Category{{ID: uuid.New()}}, nil
		},
	}
	got, err := newTestCategoryService(repo).GetTree(context.Background(), "en-US")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedLocale != "en-US" {
		t.Errorf("locale: got %q, want en-US", capturedLocale)
	}
	if len(got) != 1 {
		t.Errorf("tree size: got %d, want 1", len(got))
	}
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestCategoryService_Create_Success(t *testing.T) {
	created := false
	repo := &mockCategoryRepo{
		create: func(_ context.Context, _ *Category) error {
			created = true
			return nil
		},
	}
	err := newTestCategoryService(repo).Create(context.Background(), &Category{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Error("expected repo.Create to be called")
	}
}

func TestCategoryService_Create_BeforeHookCancels(t *testing.T) {
	hooks := sdk.NewHookRegistry()
	hookErr := errors.New("hook rejected")
	hooks.On(sdk.HookBeforeCategoryCreate, func(_ context.Context, _ *sdk.HookEvent) error {
		return hookErr
	})
	svc := NewService(&mockCategoryRepo{}, hooks, zerolog.Nop())
	err := svc.Create(context.Background(), &Category{})
	if !errors.Is(err, hookErr) {
		t.Errorf("expected hookErr, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestCategoryService_Delete_NotFound(t *testing.T) {
	err := newTestCategoryService(&mockCategoryRepo{}).Delete(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error when deleting non-existent category")
	}
}

func TestCategoryService_Delete_Success(t *testing.T) {
	id := uuid.New()
	deleted := false
	repo := &mockCategoryRepo{
		findByID: func(_ context.Context, _ uuid.UUID) (*Category, error) {
			return &Category{ID: id}, nil
		},
		delete: func(_ context.Context, _ uuid.UUID) error {
			deleted = true
			return nil
		},
	}
	if err := newTestCategoryService(repo).Delete(context.Background(), id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Error("expected repo.Delete to be called")
	}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestCategoryService_Update_NotFound(t *testing.T) {
	err := newTestCategoryService(&mockCategoryRepo{}).Update(context.Background(), &Category{ID: uuid.New()})
	if err == nil {
		t.Fatal("expected error for non-existent category")
	}
}

func TestCategoryService_Update_Success(t *testing.T) {
	id := uuid.New()
	updated := false
	repo := &mockCategoryRepo{
		findByID: func(_ context.Context, _ uuid.UUID) (*Category, error) {
			return &Category{ID: id}, nil
		},
		update: func(_ context.Context, _ *Category) error {
			updated = true
			return nil
		},
	}
	if err := newTestCategoryService(repo).Update(context.Background(), &Category{ID: id}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated {
		t.Error("expected repo.Update to be called")
	}
}
