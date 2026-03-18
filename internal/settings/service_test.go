package settings

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/zerolog"
)

// ---------------------------------------------------------------------------
// Mock Repository
// ---------------------------------------------------------------------------

type mockRepo struct {
	get    func(ctx context.Context) (*StoreSettings, error)
	upsert func(ctx context.Context, s *StoreSettings) (*StoreSettings, error)
}

func (m *mockRepo) Get(ctx context.Context) (*StoreSettings, error) {
	if m.get != nil {
		return m.get(ctx)
	}
	return nil, ErrNotFound
}

func (m *mockRepo) Upsert(ctx context.Context, s *StoreSettings) (*StoreSettings, error) {
	if m.upsert != nil {
		return m.upsert(ctx, s)
	}
	return s, nil
}

func newTestService(repo Repository) *Service {
	return NewService(repo, zerolog.Nop())
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestService_Get_ReturnsSettings(t *testing.T) {
	repo := &mockRepo{
		get: func(_ context.Context) (*StoreSettings, error) {
			return &StoreSettings{StoreName: "My Shop", Currency: "USD"}, nil
		},
	}
	svc := newTestService(repo)

	s, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.StoreName != "My Shop" {
		t.Errorf("expected store_name 'My Shop', got %q", s.StoreName)
	}
	if s.Currency != "USD" {
		t.Errorf("expected currency 'USD', got %q", s.Currency)
	}
}

func TestService_Get_FallbackToDefaults(t *testing.T) {
	repo := &mockRepo{} // get returns ErrNotFound by default
	svc := newTestService(repo)

	s, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.StoreName != "Stoa" {
		t.Errorf("expected default store_name 'Stoa', got %q", s.StoreName)
	}
	if s.Currency != "EUR" {
		t.Errorf("expected default currency 'EUR', got %q", s.Currency)
	}
	if s.Timezone != "UTC" {
		t.Errorf("expected default timezone 'UTC', got %q", s.Timezone)
	}
}

func TestService_Get_RepoError(t *testing.T) {
	repo := &mockRepo{
		get: func(_ context.Context) (*StoreSettings, error) {
			return nil, errors.New("db connection failed")
		},
	}
	svc := newTestService(repo)

	_, err := svc.Get(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestService_Update_Success(t *testing.T) {
	repo := &mockRepo{
		upsert: func(_ context.Context, s *StoreSettings) (*StoreSettings, error) {
			return s, nil
		},
	}
	svc := newTestService(repo)

	s := &StoreSettings{StoreName: "Updated Shop", Currency: "USD", Timezone: "Europe/Berlin"}
	result, err := svc.Update(context.Background(), s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.StoreName != "Updated Shop" {
		t.Errorf("expected 'Updated Shop', got %q", result.StoreName)
	}
}

func TestService_Update_EmptyName(t *testing.T) {
	svc := newTestService(&mockRepo{})

	_, err := svc.Update(context.Background(), &StoreSettings{StoreName: "", Currency: "EUR", Timezone: "UTC"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_Update_DefaultCurrency(t *testing.T) {
	repo := &mockRepo{
		upsert: func(_ context.Context, s *StoreSettings) (*StoreSettings, error) {
			return s, nil
		},
	}
	svc := newTestService(repo)

	s := &StoreSettings{StoreName: "Shop", Currency: "", Timezone: ""}
	result, err := svc.Update(context.Background(), s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Currency != "EUR" {
		t.Errorf("expected default currency 'EUR', got %q", result.Currency)
	}
	if result.Timezone != "UTC" {
		t.Errorf("expected default timezone 'UTC', got %q", result.Timezone)
	}
}

func TestService_Update_RepoError(t *testing.T) {
	repo := &mockRepo{
		upsert: func(_ context.Context, s *StoreSettings) (*StoreSettings, error) {
			return nil, errors.New("db error")
		},
	}
	svc := newTestService(repo)

	_, err := svc.Update(context.Background(), &StoreSettings{StoreName: "Shop", Currency: "EUR", Timezone: "UTC"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
