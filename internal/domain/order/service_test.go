package order

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/pkg/sdk"
)

// ---------------------------------------------------------------------------
// Mock OrderRepository
// ---------------------------------------------------------------------------

type mockOrderRepo struct {
	findByID        func(ctx context.Context, id uuid.UUID) (*Order, error)
	findAll         func(ctx context.Context, f OrderFilter) ([]Order, int, error)
	findByCustomerID func(ctx context.Context, customerID uuid.UUID) ([]Order, error)
	create          func(ctx context.Context, o *Order) error
	update          func(ctx context.Context, o *Order) error
	updateStatus    func(ctx context.Context, id uuid.UUID, from, to, comment string) error
}

func (m *mockOrderRepo) FindByID(ctx context.Context, id uuid.UUID) (*Order, error) {
	if m.findByID != nil {
		return m.findByID(ctx, id)
	}
	return nil, errors.New("order not found")
}
func (m *mockOrderRepo) FindAll(ctx context.Context, f OrderFilter) ([]Order, int, error) {
	if m.findAll != nil {
		return m.findAll(ctx, f)
	}
	return nil, 0, nil
}
func (m *mockOrderRepo) FindByCustomerID(ctx context.Context, customerID uuid.UUID) ([]Order, error) {
	if m.findByCustomerID != nil {
		return m.findByCustomerID(ctx, customerID)
	}
	return nil, nil
}
func (m *mockOrderRepo) Create(ctx context.Context, o *Order) error {
	if m.create != nil {
		return m.create(ctx, o)
	}
	return nil
}
func (m *mockOrderRepo) Update(ctx context.Context, o *Order) error {
	if m.update != nil {
		return m.update(ctx, o)
	}
	return nil
}
func (m *mockOrderRepo) UpdateStatus(ctx context.Context, id uuid.UUID, from, to, comment string) error {
	if m.updateStatus != nil {
		return m.updateStatus(ctx, id, from, to, comment)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func newTestOrderService(repo OrderRepository) *Service {
	return NewService(repo, sdk.NewHookRegistry(), zerolog.Nop())
}

// ---------------------------------------------------------------------------
// ValidateStatusTransition
// ---------------------------------------------------------------------------

func TestService_ValidateStatusTransition_Valid(t *testing.T) {
	svc := newTestOrderService(&mockOrderRepo{})
	if err := svc.ValidateStatusTransition(StatusPending, StatusConfirmed); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestService_ValidateStatusTransition_Invalid(t *testing.T) {
	svc := newTestOrderService(&mockOrderRepo{})
	if err := svc.ValidateStatusTransition(StatusPending, StatusShipped); err == nil {
		t.Error("expected error for invalid transition pending→shipped")
	}
}

func TestService_ValidateStatusTransition_UnknownStatus(t *testing.T) {
	svc := newTestOrderService(&mockOrderRepo{})
	if err := svc.ValidateStatusTransition("unknown", StatusConfirmed); err == nil {
		t.Error("expected error for unknown from-status")
	}
}

func TestService_ValidateStatusTransition_TerminalStatuses(t *testing.T) {
	svc := newTestOrderService(&mockOrderRepo{})
	for _, terminal := range []string{StatusCancelled, StatusRefunded} {
		if err := svc.ValidateStatusTransition(terminal, StatusPending); err == nil {
			t.Errorf("expected error: terminal status %q should not allow transitions", terminal)
		}
	}
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestService_Create_SetsIDAndOrderNumber(t *testing.T) {
	var saved *Order
	repo := &mockOrderRepo{
		create: func(_ context.Context, o *Order) error {
			saved = o
			return nil
		},
	}
	o := &Order{Currency: "EUR"}
	if err := newTestOrderService(repo).Create(context.Background(), o); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saved == nil {
		t.Fatal("expected repo.Create to be called")
	}
	if saved.ID == uuid.Nil {
		t.Error("ID should be set")
	}
	if saved.OrderNumber == "" {
		t.Error("OrderNumber should be set")
	}
}

func TestService_Create_DefaultStatusPending(t *testing.T) {
	var saved *Order
	repo := &mockOrderRepo{
		create: func(_ context.Context, o *Order) error {
			saved = o
			return nil
		},
	}
	if err := newTestOrderService(repo).Create(context.Background(), &Order{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saved.Status != StatusPending {
		t.Errorf("status: got %q, want %q", saved.Status, StatusPending)
	}
}

func TestService_Create_PreservesExistingStatus(t *testing.T) {
	var saved *Order
	repo := &mockOrderRepo{
		create: func(_ context.Context, o *Order) error {
			saved = o
			return nil
		},
	}
	if err := newTestOrderService(repo).Create(context.Background(), &Order{Status: StatusConfirmed}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saved.Status != StatusConfirmed {
		t.Errorf("status: got %q, want %q", saved.Status, StatusConfirmed)
	}
}

func TestService_Create_BeforeHookCancels(t *testing.T) {
	hooks := sdk.NewHookRegistry()
	hookErr := errors.New("hook rejected")
	hooks.On(sdk.HookBeforeOrderCreate, func(_ context.Context, _ *sdk.HookEvent) error {
		return hookErr
	})
	svc := NewService(&mockOrderRepo{}, hooks, zerolog.Nop())
	err := svc.Create(context.Background(), &Order{})
	if !errors.Is(err, hookErr) {
		t.Errorf("expected hookErr, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// UpdateStatus
// ---------------------------------------------------------------------------

func TestService_UpdateStatus_NotFound(t *testing.T) {
	err := newTestOrderService(&mockOrderRepo{}).
		UpdateStatus(context.Background(), uuid.New(), StatusConfirmed, "")
	if err == nil {
		t.Fatal("expected error when order not found")
	}
}

func TestService_UpdateStatus_InvalidTransition(t *testing.T) {
	id := uuid.New()
	repo := &mockOrderRepo{
		findByID: func(_ context.Context, _ uuid.UUID) (*Order, error) {
			return &Order{ID: id, Status: StatusPending}, nil
		},
	}
	err := newTestOrderService(repo).UpdateStatus(context.Background(), id, StatusShipped, "")
	if err == nil {
		t.Fatal("expected error for invalid transition pending→shipped")
	}
}

func TestService_UpdateStatus_Success(t *testing.T) {
	id := uuid.New()
	var capturedFrom, capturedTo string
	repo := &mockOrderRepo{
		findByID: func(_ context.Context, _ uuid.UUID) (*Order, error) {
			return &Order{ID: id, Status: StatusPending}, nil
		},
		updateStatus: func(_ context.Context, _ uuid.UUID, from, to, _ string) error {
			capturedFrom = from
			capturedTo = to
			return nil
		},
	}
	if err := newTestOrderService(repo).UpdateStatus(context.Background(), id, StatusConfirmed, "ok"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedFrom != StatusPending {
		t.Errorf("from: got %q, want %q", capturedFrom, StatusPending)
	}
	if capturedTo != StatusConfirmed {
		t.Errorf("to: got %q, want %q", capturedTo, StatusConfirmed)
	}
}

// ---------------------------------------------------------------------------
// GenerateOrderNumber
// ---------------------------------------------------------------------------

func TestService_GenerateOrderNumber_Format(t *testing.T) {
	svc := newTestOrderService(&mockOrderRepo{})
	num := svc.GenerateOrderNumber()
	if !strings.HasPrefix(num, "ORD-") {
		t.Errorf("order number should start with ORD-, got %q", num)
	}
	// Format: ORD-YYYYMMDD-XXXXX → 4+8+1+5 = 18 chars
	if len(num) != 18 {
		t.Errorf("order number length: got %d, want 18 (got %q)", len(num), num)
	}
}

// ---------------------------------------------------------------------------
// List — Search filter
// ---------------------------------------------------------------------------

func TestService_List_PassesSearchFilter(t *testing.T) {
	var capturedFilter OrderFilter
	repo := &mockOrderRepo{
		findAll: func(_ context.Context, f OrderFilter) ([]Order, int, error) {
			capturedFilter = f
			return []Order{{OrderNumber: "ORD-20260315-12345"}}, 1, nil
		},
	}
	svc := newTestOrderService(repo)
	filter := OrderFilter{
		Page:   1,
		Limit:  25,
		Search: "12345",
	}
	orders, total, err := svc.List(context.Background(), filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total=1, got %d", total)
	}
	if len(orders) != 1 {
		t.Errorf("expected 1 order, got %d", len(orders))
	}
	if capturedFilter.Search != "12345" {
		t.Errorf("expected search=%q, got %q", "12345", capturedFilter.Search)
	}
}

// ---------------------------------------------------------------------------
// DispatchHook
// ---------------------------------------------------------------------------

func TestService_DispatchHook_PropagatesError(t *testing.T) {
	hooks := sdk.NewHookRegistry()
	hookErr := errors.New("hook failed")
	hooks.On(sdk.HookBeforeCheckout, func(_ context.Context, _ *sdk.HookEvent) error {
		return hookErr
	})
	svc := NewService(&mockOrderRepo{}, hooks, zerolog.Nop())

	err := svc.DispatchHook(context.Background(), sdk.HookBeforeCheckout, &Order{})
	if !errors.Is(err, hookErr) {
		t.Errorf("expected hookErr, got %v", err)
	}
}

func TestService_DispatchHook_NoHandlers_NoError(t *testing.T) {
	svc := newTestOrderService(&mockOrderRepo{})
	if err := svc.DispatchHook(context.Background(), sdk.HookBeforeCheckout, &Order{}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestService_GenerateOrderNumber_Unique(t *testing.T) {
	svc := newTestOrderService(&mockOrderRepo{})
	seen := make(map[string]struct{})
	for i := 0; i < 20; i++ {
		n := svc.GenerateOrderNumber()
		if _, dup := seen[n]; dup {
			t.Errorf("duplicate order number: %q", n)
		}
		seen[n] = struct{}{}
	}
}
