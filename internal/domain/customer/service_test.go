package customer

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/epoxx-arch/stoa/pkg/sdk"
)

// ---------------------------------------------------------------------------
// Mock CustomerRepository
// ---------------------------------------------------------------------------

type mockCustomerRepo struct {
	findByID                func(ctx context.Context, id uuid.UUID) (*Customer, error)
	findByEmail             func(ctx context.Context, email string) (*Customer, error)
	findAll                 func(ctx context.Context, f CustomerFilter) ([]Customer, int, error)
	create                  func(ctx context.Context, c *Customer) error
	update                  func(ctx context.Context, c *Customer) error
	delete                  func(ctx context.Context, id uuid.UUID) error
	createAddress           func(ctx context.Context, a *CustomerAddress) error
	updateAddress           func(ctx context.Context, a *CustomerAddress) error
	deleteAddress           func(ctx context.Context, id uuid.UUID) error
	findAddressesByCustomerID func(ctx context.Context, customerID uuid.UUID) ([]CustomerAddress, error)
}

func (m *mockCustomerRepo) FindByID(ctx context.Context, id uuid.UUID) (*Customer, error) {
	if m.findByID != nil {
		return m.findByID(ctx, id)
	}
	return nil, ErrNotFound
}
func (m *mockCustomerRepo) FindByEmail(ctx context.Context, email string) (*Customer, error) {
	if m.findByEmail != nil {
		return m.findByEmail(ctx, email)
	}
	return nil, nil
}
func (m *mockCustomerRepo) FindAll(ctx context.Context, f CustomerFilter) ([]Customer, int, error) {
	if m.findAll != nil {
		return m.findAll(ctx, f)
	}
	return nil, 0, nil
}
func (m *mockCustomerRepo) Create(ctx context.Context, c *Customer) error {
	if m.create != nil {
		return m.create(ctx, c)
	}
	return nil
}
func (m *mockCustomerRepo) Update(ctx context.Context, c *Customer) error {
	if m.update != nil {
		return m.update(ctx, c)
	}
	return nil
}
func (m *mockCustomerRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.delete != nil {
		return m.delete(ctx, id)
	}
	return nil
}
func (m *mockCustomerRepo) CreateAddress(ctx context.Context, a *CustomerAddress) error {
	if m.createAddress != nil {
		return m.createAddress(ctx, a)
	}
	return nil
}
func (m *mockCustomerRepo) UpdateAddress(ctx context.Context, a *CustomerAddress) error {
	if m.updateAddress != nil {
		return m.updateAddress(ctx, a)
	}
	return nil
}
func (m *mockCustomerRepo) DeleteAddress(ctx context.Context, id uuid.UUID) error {
	if m.deleteAddress != nil {
		return m.deleteAddress(ctx, id)
	}
	return nil
}
func (m *mockCustomerRepo) FindAddressesByCustomerID(ctx context.Context, customerID uuid.UUID) ([]CustomerAddress, error) {
	if m.findAddressesByCustomerID != nil {
		return m.findAddressesByCustomerID(ctx, customerID)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func newTestCustomerService(repo CustomerRepository) *CustomerService {
	return NewCustomerService(repo, sdk.NewHookRegistry(), zerolog.Nop())
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestCustomerService_Create_EmailTaken(t *testing.T) {
	repo := &mockCustomerRepo{
		findByEmail: func(_ context.Context, _ string) (*Customer, error) {
			return &Customer{Email: "taken@example.com"}, nil
		},
	}
	_, err := newTestCustomerService(repo).Create(context.Background(), CreateCustomerInput{
		Email:    "taken@example.com",
		Password: "secret",
	})
	if !errors.Is(err, ErrEmailTaken) {
		t.Errorf("expected ErrEmailTaken, got %v", err)
	}
}

func TestCustomerService_Create_Success(t *testing.T) {
	var saved *Customer
	repo := &mockCustomerRepo{
		findByEmail: func(_ context.Context, _ string) (*Customer, error) {
			return nil, nil // email not taken
		},
		create: func(_ context.Context, c *Customer) error {
			saved = c
			return nil
		},
	}
	got, err := newTestCustomerService(repo).Create(context.Background(), CreateCustomerInput{
		Email:     "new@example.com",
		Password:  "secret123",
		FirstName: "Jane",
		LastName:  "Doe",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saved == nil {
		t.Fatal("expected repo.Create to be called")
	}
	if !got.Active {
		t.Error("new customer should be active")
	}
	if got.PasswordHash == "" {
		t.Error("password should be hashed")
	}
	if got.PasswordHash == "secret123" {
		t.Error("password should not be stored in plain text")
	}
	if got.Email != "new@example.com" {
		t.Errorf("email: got %q, want new@example.com", got.Email)
	}
}

func TestCustomerService_Create_BeforeHookCancels(t *testing.T) {
	hooks := sdk.NewHookRegistry()
	hookErr := errors.New("hook rejected")
	hooks.On(sdk.HookBeforeCustomerCreate, func(_ context.Context, _ *sdk.HookEvent) error {
		return hookErr
	})
	repo := &mockCustomerRepo{
		findByEmail: func(_ context.Context, _ string) (*Customer, error) { return nil, nil },
	}
	svc := NewCustomerService(repo, hooks, zerolog.Nop())
	_, err := svc.Create(context.Background(), CreateCustomerInput{Email: "x@y.com", Password: "pw"})
	if !errors.Is(err, hookErr) {
		t.Errorf("expected hookErr, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// GetByEmail
// ---------------------------------------------------------------------------

func TestCustomerService_GetByEmail_NotFound(t *testing.T) {
	// FindByEmail returning nil, nil → ErrNotFound
	_, err := newTestCustomerService(&mockCustomerRepo{}).
		GetByEmail(context.Background(), "nobody@example.com")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// VerifyCredentials
// ---------------------------------------------------------------------------

func TestCustomerService_VerifyCredentials_UserNotFound(t *testing.T) {
	// FindByEmail returns nil → ErrInvalidCreds
	_, err := newTestCustomerService(&mockCustomerRepo{}).
		VerifyCredentials(context.Background(), "nobody@example.com", "pw")
	if !errors.Is(err, ErrInvalidCreds) {
		t.Errorf("expected ErrInvalidCreds, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestCustomerService_Update_PartialFields(t *testing.T) {
	id := uuid.New()
	existing := &Customer{
		ID:        id,
		Email:     "old@example.com",
		FirstName: "Old",
		LastName:  "Name",
		Active:    true,
	}
	var saved *Customer
	repo := &mockCustomerRepo{
		findByID: func(_ context.Context, _ uuid.UUID) (*Customer, error) {
			return existing, nil
		},
		update: func(_ context.Context, c *Customer) error {
			saved = c
			return nil
		},
	}
	active := false
	got, err := newTestCustomerService(repo).Update(context.Background(), id, UpdateCustomerInput{
		FirstName: "New",
		Active:    &active,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.FirstName != "New" {
		t.Errorf("FirstName: got %q, want New", got.FirstName)
	}
	if got.Active != false {
		t.Error("Active should be updated to false")
	}
	if got.Email != "old@example.com" {
		t.Error("Email should remain unchanged")
	}
	if saved == nil {
		t.Error("expected repo.Update to be called")
	}
}

func TestCustomerService_Update_EmailTaken(t *testing.T) {
	id := uuid.New()
	repo := &mockCustomerRepo{
		findByID: func(_ context.Context, _ uuid.UUID) (*Customer, error) {
			return &Customer{ID: id, Email: "current@example.com"}, nil
		},
		findByEmail: func(_ context.Context, _ string) (*Customer, error) {
			return &Customer{Email: "taken@example.com"}, nil
		},
	}
	_, err := newTestCustomerService(repo).Update(context.Background(), id, UpdateCustomerInput{
		Email: "taken@example.com",
	})
	if !errors.Is(err, ErrEmailTaken) {
		t.Errorf("expected ErrEmailTaken, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Address operations
// ---------------------------------------------------------------------------

func TestCustomerService_AddAddress_CustomerNotFound(t *testing.T) {
	// mockCustomerRepo.FindByID returns ErrNotFound by default.
	_, err := newTestCustomerService(&mockCustomerRepo{}).
		AddAddress(context.Background(), uuid.New(), AddressInput{
			FirstName: "Jane", LastName: "Doe", Street: "Main St 1",
			City: "Berlin", Zip: "10115", CountryCode: "DE",
		})
	if err == nil {
		t.Fatal("expected error when customer not found")
	}
}

func TestCustomerService_AddAddress_Success(t *testing.T) {
	customerID := uuid.New()
	var saved *CustomerAddress
	repo := &mockCustomerRepo{
		findByID: func(_ context.Context, _ uuid.UUID) (*Customer, error) {
			return &Customer{ID: customerID}, nil
		},
		createAddress: func(_ context.Context, a *CustomerAddress) error {
			saved = a
			return nil
		},
	}
	got, err := newTestCustomerService(repo).AddAddress(context.Background(), customerID, AddressInput{
		FirstName:   "Jane",
		LastName:    "Doe",
		Street:      "Main St 1",
		City:        "Berlin",
		Zip:         "10115",
		CountryCode: "DE",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saved == nil {
		t.Fatal("expected repo.CreateAddress to be called")
	}
	if got.CustomerID != customerID {
		t.Errorf("CustomerID: got %s, want %s", got.CustomerID, customerID)
	}
}

func TestCustomerService_DeleteAddress_WrongOwner(t *testing.T) {
	customerID := uuid.New()
	otherAddrID := uuid.New() // not in the customer's address list
	repo := &mockCustomerRepo{
		findAddressesByCustomerID: func(_ context.Context, _ uuid.UUID) ([]CustomerAddress, error) {
			return []CustomerAddress{{ID: uuid.New(), CustomerID: customerID}}, nil
		},
	}
	err := newTestCustomerService(repo).DeleteAddress(context.Background(), customerID, otherAddrID)
	if !errors.Is(err, ErrAddressOwner) {
		t.Errorf("expected ErrAddressOwner, got %v", err)
	}
}

func TestCustomerService_DeleteAddress_Success(t *testing.T) {
	customerID := uuid.New()
	addrID := uuid.New()
	deleted := false
	repo := &mockCustomerRepo{
		findAddressesByCustomerID: func(_ context.Context, _ uuid.UUID) ([]CustomerAddress, error) {
			return []CustomerAddress{{ID: addrID, CustomerID: customerID}}, nil
		},
		deleteAddress: func(_ context.Context, id uuid.UUID) error {
			if id != addrID {
				return errors.New("wrong id")
			}
			deleted = true
			return nil
		},
	}
	if err := newTestCustomerService(repo).DeleteAddress(context.Background(), customerID, addrID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Error("expected repo.DeleteAddress to be called")
	}
}

func TestCustomerService_UpdateAddress_WrongOwner(t *testing.T) {
	customerID := uuid.New()
	repo := &mockCustomerRepo{
		findAddressesByCustomerID: func(_ context.Context, _ uuid.UUID) ([]CustomerAddress, error) {
			return []CustomerAddress{{ID: uuid.New(), CustomerID: customerID}}, nil
		},
	}
	_, err := newTestCustomerService(repo).UpdateAddress(context.Background(), customerID, uuid.New(), AddressInput{
		FirstName: "Jane", LastName: "Doe", Street: "X", City: "Y", Zip: "Z", CountryCode: "DE",
	})
	if !errors.Is(err, ErrAddressOwner) {
		t.Errorf("expected ErrAddressOwner, got %v", err)
	}
}
