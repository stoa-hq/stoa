package tax

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
// Mock TaxRuleRepository
// ---------------------------------------------------------------------------

type mockTaxRepo struct {
	findByID func(ctx context.Context, id uuid.UUID) (*TaxRule, error)
	findAll  func(ctx context.Context, f TaxRuleFilter) ([]TaxRule, int, error)
	create   func(ctx context.Context, t *TaxRule) error
	update   func(ctx context.Context, t *TaxRule) error
	delete   func(ctx context.Context, id uuid.UUID) error
}

func (m *mockTaxRepo) FindByID(ctx context.Context, id uuid.UUID) (*TaxRule, error) {
	if m.findByID != nil {
		return m.findByID(ctx, id)
	}
	return nil, ErrNotFound
}
func (m *mockTaxRepo) FindAll(ctx context.Context, f TaxRuleFilter) ([]TaxRule, int, error) {
	if m.findAll != nil {
		return m.findAll(ctx, f)
	}
	return nil, 0, nil
}
func (m *mockTaxRepo) Create(ctx context.Context, t *TaxRule) error {
	if m.create != nil {
		return m.create(ctx, t)
	}
	return nil
}
func (m *mockTaxRepo) Update(ctx context.Context, t *TaxRule) error {
	if m.update != nil {
		return m.update(ctx, t)
	}
	return nil
}
func (m *mockTaxRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.delete != nil {
		return m.delete(ctx, id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func newTestTaxService(repo TaxRuleRepository) TaxService {
	return NewService(repo, sdk.NewHookRegistry(), zerolog.Nop())
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestTaxService_Create_MissingName(t *testing.T) {
	err := newTestTaxService(&mockTaxRepo{}).Create(context.Background(), &TaxRule{Rate: 1900})
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for missing name, got %v", err)
	}
}

func TestTaxService_Create_SetsIDAndTimestamps(t *testing.T) {
	var saved *TaxRule
	repo := &mockTaxRepo{
		create: func(_ context.Context, t *TaxRule) error {
			saved = t
			return nil
		},
	}
	before := time.Now()
	err := newTestTaxService(repo).Create(context.Background(), &TaxRule{Name: "VAT 19%", Rate: 1900})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saved.ID == uuid.Nil {
		t.Error("ID should be set")
	}
	if saved.CreatedAt.Before(before) {
		t.Error("CreatedAt should be set to current time")
	}
	if saved.UpdatedAt.Before(before) {
		t.Error("UpdatedAt should be set to current time")
	}
}

func TestTaxService_Create_BeforeHookCancels(t *testing.T) {
	hooks := sdk.NewHookRegistry()
	hookErr := errors.New("hook rejected")
	hooks.On(HookBeforeTaxCreate, func(_ context.Context, _ *sdk.HookEvent) error {
		return hookErr
	})
	svc := NewService(&mockTaxRepo{}, hooks, zerolog.Nop())
	err := svc.Create(context.Background(), &TaxRule{Name: "VAT"})
	if !errors.Is(err, hookErr) {
		t.Errorf("expected hookErr, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestTaxService_GetByID_NotFound(t *testing.T) {
	_, err := newTestTaxService(&mockTaxRepo{}).GetByID(context.Background(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestTaxService_GetByID_Found(t *testing.T) {
	id := uuid.New()
	repo := &mockTaxRepo{
		findByID: func(_ context.Context, _ uuid.UUID) (*TaxRule, error) {
			return &TaxRule{ID: id, Name: "VAT", Rate: 1900}, nil
		},
	}
	got, err := newTestTaxService(repo).GetByID(context.Background(), id)
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

func TestTaxService_Update_MissingName(t *testing.T) {
	err := newTestTaxService(&mockTaxRepo{}).Update(context.Background(), &TaxRule{ID: uuid.New(), Rate: 1900})
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestTaxService_Update_SetsUpdatedAt(t *testing.T) {
	var saved *TaxRule
	repo := &mockTaxRepo{
		update: func(_ context.Context, t *TaxRule) error {
			saved = t
			return nil
		},
	}
	before := time.Now()
	if err := newTestTaxService(repo).Update(context.Background(), &TaxRule{ID: uuid.New(), Name: "VAT"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saved.UpdatedAt.Before(before) {
		t.Error("UpdatedAt should be refreshed on update")
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestTaxService_Delete_Success(t *testing.T) {
	deleted := false
	repo := &mockTaxRepo{
		delete: func(_ context.Context, _ uuid.UUID) error {
			deleted = true
			return nil
		},
	}
	if err := newTestTaxService(repo).Delete(context.Background(), uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Error("expected repo.Delete to be called")
	}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestTaxService_List(t *testing.T) {
	rules := []TaxRule{{ID: uuid.New(), Name: "VAT"}, {ID: uuid.New(), Name: "Reduced VAT"}}
	repo := &mockTaxRepo{
		findAll: func(_ context.Context, _ TaxRuleFilter) ([]TaxRule, int, error) {
			return rules, 2, nil
		},
	}
	got, total, err := newTestTaxService(repo).List(context.Background(), TaxRuleFilter{})
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

// ---------------------------------------------------------------------------
// RatePercent (entity helper)
// ---------------------------------------------------------------------------

func TestTaxRule_RatePercent(t *testing.T) {
	tests := []struct {
		rate int
		want float64
	}{
		{1900, 19.0},
		{700, 7.0},
		{0, 0.0},
	}
	for _, tt := range tests {
		rule := &TaxRule{Rate: tt.rate}
		if got := rule.RatePercent(); got != tt.want {
			t.Errorf("RatePercent(%d): got %v, want %v", tt.rate, got, tt.want)
		}
	}
}
