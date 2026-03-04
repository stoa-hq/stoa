package discount

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/epoxx-arch/stoa/pkg/sdk"
)

// ---------------------------------------------------------------------------
// Mock DiscountRepository
// ---------------------------------------------------------------------------

type mockDiscountRepo struct {
	findByID          func(ctx context.Context, id uuid.UUID) (*Discount, error)
	findByCode        func(ctx context.Context, code string) (*Discount, error)
	findAll           func(ctx context.Context, f DiscountFilter) ([]Discount, int, error)
	create            func(ctx context.Context, d *Discount) error
	update            func(ctx context.Context, d *Discount) error
	delete            func(ctx context.Context, id uuid.UUID) error
	incrementUsedCount func(ctx context.Context, id uuid.UUID) error
}

func (m *mockDiscountRepo) FindByID(ctx context.Context, id uuid.UUID) (*Discount, error) {
	if m.findByID != nil {
		return m.findByID(ctx, id)
	}
	return nil, ErrNotFound
}
func (m *mockDiscountRepo) FindByCode(ctx context.Context, code string) (*Discount, error) {
	if m.findByCode != nil {
		return m.findByCode(ctx, code)
	}
	return nil, ErrNotFound
}
func (m *mockDiscountRepo) FindAll(ctx context.Context, f DiscountFilter) ([]Discount, int, error) {
	if m.findAll != nil {
		return m.findAll(ctx, f)
	}
	return nil, 0, nil
}
func (m *mockDiscountRepo) Create(ctx context.Context, d *Discount) error {
	if m.create != nil {
		return m.create(ctx, d)
	}
	return nil
}
func (m *mockDiscountRepo) Update(ctx context.Context, d *Discount) error {
	if m.update != nil {
		return m.update(ctx, d)
	}
	return nil
}
func (m *mockDiscountRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.delete != nil {
		return m.delete(ctx, id)
	}
	return nil
}
func (m *mockDiscountRepo) IncrementUsedCount(ctx context.Context, id uuid.UUID) error {
	if m.incrementUsedCount != nil {
		return m.incrementUsedCount(ctx, id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func newTestDiscountService(repo DiscountRepository) DiscountService {
	return NewService(repo, sdk.NewHookRegistry(), zerolog.Nop())
}

// activeDiscount returns a minimal valid, active discount with no restrictions.
func activeDiscount(code string) *Discount {
	return &Discount{
		ID:     uuid.New(),
		Code:   code,
		Type:   "percentage",
		Value:  1000,
		Active: true,
	}
}

// ---------------------------------------------------------------------------
// ValidateCode
// ---------------------------------------------------------------------------

func TestValidateCode_Valid(t *testing.T) {
	d := activeDiscount("SUMMER10")
	repo := &mockDiscountRepo{
		findByCode: func(_ context.Context, _ string) (*Discount, error) {
			return d, nil
		},
	}

	got, err := newTestDiscountService(repo).ValidateCode(context.Background(), "SUMMER10", 5000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Code != "SUMMER10" {
		t.Errorf("code: got %q, want SUMMER10", got.Code)
	}
}

func TestValidateCode_CodeNotFound(t *testing.T) {
	// mockDiscountRepo.findByCode returns ErrNotFound by default.
	_, err := newTestDiscountService(&mockDiscountRepo{}).
		ValidateCode(context.Background(), "UNKNOWN", 1000)
	if !errors.Is(err, ErrCodeInvalid) {
		t.Errorf("expected ErrCodeInvalid, got %v", err)
	}
}

func TestValidateCode_Inactive(t *testing.T) {
	d := activeDiscount("OFF")
	d.Active = false

	repo := &mockDiscountRepo{
		findByCode: func(_ context.Context, _ string) (*Discount, error) { return d, nil },
	}

	_, err := newTestDiscountService(repo).ValidateCode(context.Background(), "OFF", 1000)
	if !errors.Is(err, ErrCodeInvalid) {
		t.Errorf("expected ErrCodeInvalid for inactive discount, got %v", err)
	}
}

func TestValidateCode_Expired(t *testing.T) {
	d := activeDiscount("EXPIRED")
	past := time.Now().Add(-24 * time.Hour)
	d.ValidUntil = &past

	repo := &mockDiscountRepo{
		findByCode: func(_ context.Context, _ string) (*Discount, error) { return d, nil },
	}

	_, err := newTestDiscountService(repo).ValidateCode(context.Background(), "EXPIRED", 1000)
	if !errors.Is(err, ErrCodeInvalid) {
		t.Errorf("expected ErrCodeInvalid for expired discount, got %v", err)
	}
}

func TestValidateCode_NotYetValid(t *testing.T) {
	d := activeDiscount("FUTURE")
	future := time.Now().Add(24 * time.Hour)
	d.ValidFrom = &future

	repo := &mockDiscountRepo{
		findByCode: func(_ context.Context, _ string) (*Discount, error) { return d, nil },
	}

	_, err := newTestDiscountService(repo).ValidateCode(context.Background(), "FUTURE", 1000)
	if !errors.Is(err, ErrCodeInvalid) {
		t.Errorf("expected ErrCodeInvalid for not-yet-valid discount, got %v", err)
	}
}

func TestValidateCode_MaxUsesReached(t *testing.T) {
	d := activeDiscount("MAXED")
	max := 5
	d.MaxUses = &max
	d.UsedCount = 5

	repo := &mockDiscountRepo{
		findByCode: func(_ context.Context, _ string) (*Discount, error) { return d, nil },
	}

	_, err := newTestDiscountService(repo).ValidateCode(context.Background(), "MAXED", 1000)
	if !errors.Is(err, ErrMaxUsesReached) {
		t.Errorf("expected ErrMaxUsesReached, got %v", err)
	}
}

func TestValidateCode_MinOrderValueNotMet(t *testing.T) {
	d := activeDiscount("MIN100")
	min := 10000 // 100.00 in cents
	d.MinOrderValue = &min

	repo := &mockDiscountRepo{
		findByCode: func(_ context.Context, _ string) (*Discount, error) { return d, nil },
	}

	_, err := newTestDiscountService(repo).ValidateCode(context.Background(), "MIN100", 500)
	if err == nil {
		t.Fatal("expected error when order total is below minimum")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput (min order value), got %v", err)
	}
}

func TestValidateCode_MaxUsesNotYetReached(t *testing.T) {
	d := activeDiscount("ALMOSTFULL")
	max := 10
	d.MaxUses = &max
	d.UsedCount = 9 // one slot left

	repo := &mockDiscountRepo{
		findByCode: func(_ context.Context, _ string) (*Discount, error) { return d, nil },
	}

	got, err := newTestDiscountService(repo).ValidateCode(context.Background(), "ALMOSTFULL", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Error("expected valid discount returned")
	}
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestCreate_EmptyCode(t *testing.T) {
	err := newTestDiscountService(&mockDiscountRepo{}).
		Create(context.Background(), &Discount{Type: "percentage"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for empty code, got %v", err)
	}
}

func TestCreate_InvalidType(t *testing.T) {
	err := newTestDiscountService(&mockDiscountRepo{}).
		Create(context.Background(), &Discount{Code: "CODE", Type: "invalid"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for invalid type, got %v", err)
	}
}

func TestCreate_SetsIDAndTimestamps(t *testing.T) {
	var saved *Discount
	repo := &mockDiscountRepo{
		create: func(_ context.Context, d *Discount) error {
			saved = d
			return nil
		},
	}

	d := &Discount{Code: "NEW", Type: "fixed", Value: 500}
	if err := newTestDiscountService(repo).Create(context.Background(), d); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saved == nil {
		t.Fatal("expected repo.Create to be called")
	}
	if saved.ID == uuid.Nil {
		t.Error("ID should be set")
	}
	if saved.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if saved.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
}

func TestCreate_PercentageType(t *testing.T) {
	d := &Discount{Code: "PCT", Type: "percentage", Value: 1500}
	if err := newTestDiscountService(&mockDiscountRepo{}).Create(context.Background(), d); err != nil {
		t.Fatalf("percentage type should be accepted: %v", err)
	}
}

func TestCreate_FixedType(t *testing.T) {
	d := &Discount{Code: "FIX", Type: "fixed", Value: 500}
	if err := newTestDiscountService(&mockDiscountRepo{}).Create(context.Background(), d); err != nil {
		t.Fatalf("fixed type should be accepted: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ApplyDiscount
// ---------------------------------------------------------------------------

func TestApplyDiscount_CallsIncrement(t *testing.T) {
	incremented := false
	id := uuid.New()
	repo := &mockDiscountRepo{
		incrementUsedCount: func(_ context.Context, got uuid.UUID) error {
			if got != id {
				t.Errorf("IncrementUsedCount called with wrong ID: %s", got)
			}
			incremented = true
			return nil
		},
	}

	if err := newTestDiscountService(repo).ApplyDiscount(context.Background(), id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !incremented {
		t.Error("expected IncrementUsedCount to be called")
	}
}
