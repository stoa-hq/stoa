# Stoa: Test-Konventionen

## Grundregeln

- **Kein externes Framework** — stdlib `testing` durchgehend
- Tests in `*_test.go` im **gleichen Package** (`package product`, nicht `package product_test`) — interne Typen zugänglich
- Datei liegt im selben Verzeichnis wie der zu testende Code

```bash
make test                                              # Alle Tests
go test ./internal/domain/product/... -v -run TestCreate  # Einzelner Test
make test-race                                         # Race Detector
make test-coverage                                     # → coverage.html
```

## Mock-Pattern

Structs mit optionalen Function-Fields. Default-Return ist Zero-Value oder Sentinel-Error:

```go
type mockRepo struct {
    findByID func(ctx context.Context, id uuid.UUID) (*Product, error)
    create   func(ctx context.Context, p *Product) error
    list     func(ctx context.Context, f Filter) ([]*Product, int, error)
}

func (m *mockRepo) FindByID(ctx context.Context, id uuid.UUID) (*Product, error) {
    if m.findByID != nil {
        return m.findByID(ctx, id)
    }
    return nil, ErrNotFound  // sinnvoller Default
}

func (m *mockRepo) Create(ctx context.Context, p *Product) error {
    if m.create != nil {
        return m.create(ctx, p)
    }
    return nil
}

func (m *mockRepo) List(ctx context.Context, f Filter) ([]*Product, int, error) {
    if m.list != nil {
        return m.list(ctx, f)
    }
    return nil, 0, nil
}
```

Verwendung im Test:

```go
func TestCreate(t *testing.T) {
    repo := &mockRepo{
        create: func(ctx context.Context, p *Product) error {
            p.ID = uuid.New()
            return nil
        },
    }
    svc := NewService(repo, sdk.NewHookRegistry(), zap.NewNop())
    // ...
}
```

## HookRegistry in Tests

```go
// Leere Registry (no-op) für normale Tests
hooks := sdk.NewHookRegistry()

// Für Hook-Tests direkt registrieren
hooks.On("product.before_create", func(ctx context.Context, payload interface{}) error {
    p := payload.(*Product)
    p.Name = "modified"
    return nil
})
```

## Handler-Tests

Direkter Aufruf der Handler-Methode (kein HTTP-Server nötig).
Chi-URL-Params über `chi.NewRouteContext()` injizieren:

```go
func TestGet(t *testing.T) {
    id := uuid.New()

    svc := &mockService{
        findByID: func(ctx context.Context, i uuid.UUID) (*Product, error) {
            return &Product{ID: i, Name: "Test"}, nil
        },
    }
    h := NewHandler(svc, validator.New(), zap.NewNop())

    // Request mit Chi-URL-Params
    r := httptest.NewRequest(http.MethodGet, "/"+id.String(), nil)
    rctx := chi.NewRouteContext()
    rctx.URLParams.Add("id", id.String())
    r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

    w := httptest.NewRecorder()
    h.get(w, r)

    assert.Equal(t, http.StatusOK, w.Code)
}
```

## Private Interfaces testen

Interfaces wie `cart.stockChecker` sind privat → Mock im gleichen Package definieren:

```go
// In package cart (gleiche Package, kein Export nötig)
type mockStockChecker struct {
    stockAvailable func(ctx context.Context, variantID uuid.UUID, qty int) (bool, error)
}

func (m *mockStockChecker) StockAvailable(ctx context.Context, variantID uuid.UUID, qty int) (bool, error) {
    if m.stockAvailable != nil {
        return m.stockAvailable(ctx, variantID, qty)
    }
    return true, nil
}
```

## Auth-Context in Tests

```go
import "github.com/stoa-hq/stoa/internal/auth"

ctx := auth.WithUser(context.Background(), auth.Claims{
    UserID:   uuid.New(),
    UserType: "admin",
    Role:     "super_admin",
})
```
