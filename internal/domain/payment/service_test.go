package payment

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
// Mock PaymentMethodRepository
// ---------------------------------------------------------------------------

type mockMethodRepo struct {
	findByID func(ctx context.Context, id uuid.UUID) (*PaymentMethod, error)
	findAll  func(ctx context.Context, f PaymentMethodFilter) ([]PaymentMethod, int, error)
	create   func(ctx context.Context, m *PaymentMethod) error
	update   func(ctx context.Context, m *PaymentMethod) error
	delete   func(ctx context.Context, id uuid.UUID) error
}

func (m *mockMethodRepo) FindByID(ctx context.Context, id uuid.UUID) (*PaymentMethod, error) {
	if m.findByID != nil {
		return m.findByID(ctx, id)
	}
	return nil, ErrMethodNotFound
}
func (m *mockMethodRepo) FindAll(ctx context.Context, f PaymentMethodFilter) ([]PaymentMethod, int, error) {
	if m.findAll != nil {
		return m.findAll(ctx, f)
	}
	return nil, 0, nil
}
func (m *mockMethodRepo) Create(ctx context.Context, pm *PaymentMethod) error {
	if m.create != nil {
		return m.create(ctx, pm)
	}
	return nil
}
func (m *mockMethodRepo) Update(ctx context.Context, pm *PaymentMethod) error {
	if m.update != nil {
		return m.update(ctx, pm)
	}
	return nil
}
func (m *mockMethodRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.delete != nil {
		return m.delete(ctx, id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock PaymentTransactionRepository
// ---------------------------------------------------------------------------

type mockTxRepo struct {
	create        func(ctx context.Context, t *PaymentTransaction) error
	findByOrderID func(ctx context.Context, orderID uuid.UUID) ([]PaymentTransaction, error)
}

func (m *mockTxRepo) Create(ctx context.Context, t *PaymentTransaction) error {
	if m.create != nil {
		return m.create(ctx, t)
	}
	return nil
}
func (m *mockTxRepo) FindByOrderID(ctx context.Context, orderID uuid.UUID) ([]PaymentTransaction, error) {
	if m.findByOrderID != nil {
		return m.findByOrderID(ctx, orderID)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestMethodService(repo PaymentMethodRepository) PaymentMethodService {
	return NewMethodService(repo, sdk.NewHookRegistry(), zerolog.Nop())
}

func newTestTxService(repo PaymentTransactionRepository) PaymentTransactionService {
	return NewTransactionService(repo, sdk.NewHookRegistry(), zerolog.Nop())
}

// ---------------------------------------------------------------------------
// PaymentMethodService – Create
// ---------------------------------------------------------------------------

func TestMethodService_Create_MissingProvider(t *testing.T) {
	err := newTestMethodService(&mockMethodRepo{}).Create(context.Background(), &PaymentMethod{})
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for missing provider, got %v", err)
	}
}

func TestMethodService_Create_SetsIDAndTimestamps(t *testing.T) {
	var saved *PaymentMethod
	repo := &mockMethodRepo{
		create: func(_ context.Context, m *PaymentMethod) error {
			saved = m
			return nil
		},
	}
	before := time.Now()
	err := newTestMethodService(repo).Create(context.Background(), &PaymentMethod{Provider: "stripe"})
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

func TestMethodService_Create_SetsTranslationMethodID(t *testing.T) {
	var saved *PaymentMethod
	repo := &mockMethodRepo{
		create: func(_ context.Context, m *PaymentMethod) error {
			saved = m
			return nil
		},
	}
	m := &PaymentMethod{
		Provider: "paypal",
		Translations: []PaymentMethodTranslation{
			{Locale: "de-DE", Name: "PayPal"},
			{Locale: "en-US", Name: "PayPal"},
		},
	}
	if err := newTestMethodService(repo).Create(context.Background(), m); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, tr := range saved.Translations {
		if tr.PaymentMethodID != saved.ID {
			t.Errorf("translation %q: PaymentMethodID = %s, want %s", tr.Locale, tr.PaymentMethodID, saved.ID)
		}
	}
}

func TestMethodService_Create_BeforeHookCancels(t *testing.T) {
	hooks := sdk.NewHookRegistry()
	hookErr := errors.New("hook rejected")
	hooks.On(HookBeforePaymentMethodCreate, func(_ context.Context, _ *sdk.HookEvent) error {
		return hookErr
	})
	svc := NewMethodService(&mockMethodRepo{}, hooks, zerolog.Nop())
	err := svc.Create(context.Background(), &PaymentMethod{Provider: "stripe"})
	if !errors.Is(err, hookErr) {
		t.Errorf("expected hookErr, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// PaymentMethodService – GetByID / Delete / List
// ---------------------------------------------------------------------------

func TestMethodService_GetByID_NotFound(t *testing.T) {
	_, err := newTestMethodService(&mockMethodRepo{}).GetByID(context.Background(), uuid.New())
	if !errors.Is(err, ErrMethodNotFound) {
		t.Errorf("expected ErrMethodNotFound, got %v", err)
	}
}

func TestMethodService_GetByID_Found(t *testing.T) {
	id := uuid.New()
	repo := &mockMethodRepo{
		findByID: func(_ context.Context, _ uuid.UUID) (*PaymentMethod, error) {
			return &PaymentMethod{ID: id, Provider: "stripe"}, nil
		},
	}
	got, err := newTestMethodService(repo).GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != id {
		t.Errorf("ID: got %s, want %s", got.ID, id)
	}
}

func TestMethodService_Delete_Success(t *testing.T) {
	deleted := false
	repo := &mockMethodRepo{
		delete: func(_ context.Context, _ uuid.UUID) error {
			deleted = true
			return nil
		},
	}
	if err := newTestMethodService(repo).Delete(context.Background(), uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Error("expected repo.Delete to be called")
	}
}

func TestMethodService_Update_MissingProvider(t *testing.T) {
	err := newTestMethodService(&mockMethodRepo{}).Update(context.Background(), &PaymentMethod{ID: uuid.New()})
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for missing provider, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// PaymentTransactionService
// ---------------------------------------------------------------------------

func TestTxService_CreateTransaction_MissingOrderID(t *testing.T) {
	err := newTestTxService(&mockTxRepo{}).CreateTransaction(context.Background(), &PaymentTransaction{
		PaymentMethodID: uuid.New(),
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for missing order_id, got %v", err)
	}
}

func TestTxService_CreateTransaction_MissingPaymentMethodID(t *testing.T) {
	err := newTestTxService(&mockTxRepo{}).CreateTransaction(context.Background(), &PaymentTransaction{
		OrderID: uuid.New(),
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for missing payment_method_id, got %v", err)
	}
}

func TestTxService_CreateTransaction_SetsIDAndTimestamp(t *testing.T) {
	var saved *PaymentTransaction
	repo := &mockTxRepo{
		create: func(_ context.Context, t *PaymentTransaction) error {
			saved = t
			return nil
		},
	}
	before := time.Now()
	tx := &PaymentTransaction{
		OrderID:         uuid.New(),
		PaymentMethodID: uuid.New(),
		Amount:          4999,
		Currency:        "EUR",
		Status:          "pending",
	}
	if err := newTestTxService(repo).CreateTransaction(context.Background(), tx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saved.ID == uuid.Nil {
		t.Error("ID should be set")
	}
	if saved.CreatedAt.Before(before) {
		t.Error("CreatedAt should be set")
	}
}
