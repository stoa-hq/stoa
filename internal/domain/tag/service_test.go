package tag

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/pkg/sdk"
)

// ---------------------------------------------------------------------------
// Mock TagRepository
// ---------------------------------------------------------------------------

type mockTagRepo struct {
	findByID func(ctx context.Context, id uuid.UUID) (*Tag, error)
	findAll  func(ctx context.Context, f TagFilter) ([]Tag, int, error)
	create   func(ctx context.Context, t *Tag) error
	update   func(ctx context.Context, t *Tag) error
	delete   func(ctx context.Context, id uuid.UUID) error
}

func (m *mockTagRepo) FindByID(ctx context.Context, id uuid.UUID) (*Tag, error) {
	if m.findByID != nil {
		return m.findByID(ctx, id)
	}
	return nil, ErrNotFound
}
func (m *mockTagRepo) FindAll(ctx context.Context, f TagFilter) ([]Tag, int, error) {
	if m.findAll != nil {
		return m.findAll(ctx, f)
	}
	return nil, 0, nil
}
func (m *mockTagRepo) Create(ctx context.Context, t *Tag) error {
	if m.create != nil {
		return m.create(ctx, t)
	}
	return nil
}
func (m *mockTagRepo) Update(ctx context.Context, t *Tag) error {
	if m.update != nil {
		return m.update(ctx, t)
	}
	return nil
}
func (m *mockTagRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.delete != nil {
		return m.delete(ctx, id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func newTestTagService(repo TagRepository) TagService {
	return NewService(repo, sdk.NewHookRegistry(), zerolog.Nop())
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestTagService_Create_MissingName(t *testing.T) {
	err := newTestTagService(&mockTagRepo{}).Create(context.Background(), &Tag{Slug: "my-tag"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for missing name, got %v", err)
	}
}

func TestTagService_Create_MissingSlug(t *testing.T) {
	err := newTestTagService(&mockTagRepo{}).Create(context.Background(), &Tag{Name: "My Tag"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for missing slug, got %v", err)
	}
}

func TestTagService_Create_SetsID(t *testing.T) {
	var saved *Tag
	repo := &mockTagRepo{
		create: func(_ context.Context, t *Tag) error {
			saved = t
			return nil
		},
	}
	err := newTestTagService(repo).Create(context.Background(), &Tag{Name: "Sale", Slug: "sale"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saved == nil {
		t.Fatal("expected repo.Create to be called")
	}
	if saved.ID == uuid.Nil {
		t.Error("ID should be set")
	}
}

func TestTagService_Create_BeforeHookCancels(t *testing.T) {
	hooks := sdk.NewHookRegistry()
	hookErr := errors.New("hook rejected")
	hooks.On(HookBeforeTagCreate, func(_ context.Context, _ *sdk.HookEvent) error {
		return hookErr
	})
	svc := NewService(&mockTagRepo{}, hooks, zerolog.Nop())
	err := svc.Create(context.Background(), &Tag{Name: "X", Slug: "x"})
	if !errors.Is(err, hookErr) {
		t.Errorf("expected hookErr, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestTagService_GetByID_NotFound(t *testing.T) {
	_, err := newTestTagService(&mockTagRepo{}).GetByID(context.Background(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestTagService_GetByID_Found(t *testing.T) {
	id := uuid.New()
	repo := &mockTagRepo{
		findByID: func(_ context.Context, _ uuid.UUID) (*Tag, error) {
			return &Tag{ID: id, Name: "Sale", Slug: "sale"}, nil
		},
	}
	got, err := newTestTagService(repo).GetByID(context.Background(), id)
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

func TestTagService_Update_MissingName(t *testing.T) {
	err := newTestTagService(&mockTagRepo{}).Update(context.Background(), &Tag{ID: uuid.New(), Slug: "x"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestTagService_Update_MissingSlug(t *testing.T) {
	err := newTestTagService(&mockTagRepo{}).Update(context.Background(), &Tag{ID: uuid.New(), Name: "X"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestTagService_Update_Success(t *testing.T) {
	updated := false
	repo := &mockTagRepo{
		update: func(_ context.Context, _ *Tag) error {
			updated = true
			return nil
		},
	}
	if err := newTestTagService(repo).Update(context.Background(), &Tag{ID: uuid.New(), Name: "Sale", Slug: "sale"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated {
		t.Error("expected repo.Update to be called")
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestTagService_Delete_Success(t *testing.T) {
	deleted := false
	repo := &mockTagRepo{
		delete: func(_ context.Context, _ uuid.UUID) error {
			deleted = true
			return nil
		},
	}
	if err := newTestTagService(repo).Delete(context.Background(), uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Error("expected repo.Delete to be called")
	}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestTagService_List(t *testing.T) {
	tags := []Tag{{ID: uuid.New()}, {ID: uuid.New()}, {ID: uuid.New()}}
	repo := &mockTagRepo{
		findAll: func(_ context.Context, _ TagFilter) ([]Tag, int, error) {
			return tags, 3, nil
		},
	}
	got, total, err := newTestTagService(repo).List(context.Background(), TagFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("items: got %d, want 3", len(got))
	}
	if total != 3 {
		t.Errorf("total: got %d, want 3", total)
	}
}
