# Stoa: Neues Domain-Package erstellen

Vollständige Anleitung für ein neues Domain-Package in `internal/domain/<entity>/`.

## 6-Datei-Pattern

Jedes Domain-Package besteht aus exakt diesen 6 Dateien:

| Datei | Inhalt |
|-------|--------|
| `entity.go` | Struct-Definitionen |
| `repository.go` | Interface + Filter-Struct |
| `postgres.go` | PostgreSQL-Implementierung (pgxpool) |
| `service.go` | Business-Logik, Hook-Dispatch, Sentinel Errors |
| `handler.go` | HTTP Handler mit `RegisterAdminRoutes` / `RegisterStoreRoutes` |
| `dto.go` | Request/Response DTOs mit `validator` Tags |

## entity.go

```go
package <entity>

import (
    "time"
    "github.com/google/uuid"
)

type <Entity> struct {
    ID          uuid.UUID              `json:"id"`
    // ... Felder
    CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
    Metadata     map[string]interface{} `json:"metadata,omitempty"`
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
}
```

## repository.go

```go
package <entity>

import (
    "context"
    "github.com/google/uuid"
)

type Repository interface {
    Create(ctx context.Context, e *<Entity>) error
    FindByID(ctx context.Context, id uuid.UUID) (*<Entity>, error)
    List(ctx context.Context, filter Filter) ([]*<Entity>, int, error)
    Update(ctx context.Context, e *<Entity>) error
    Delete(ctx context.Context, id uuid.UUID) error
}

type Filter struct {
    Page   int
    Limit  int
    Sort   string
    Order  string
}
```

## service.go

```go
package <entity>

import (
    "context"
    "errors"
    "github.com/stoa-hq/stoa/pkg/sdk"
    "go.uber.org/zap"
    "github.com/google/uuid"
)

var (
    ErrNotFound = errors.New("<entity>: not found")
    ErrInvalid  = errors.New("<entity>: invalid")
)

type Service struct {
    repo   Repository
    hooks  *sdk.HookRegistry
    logger *zap.Logger
}

func NewService(repo Repository, hooks *sdk.HookRegistry, logger *zap.Logger) *Service {
    return &Service{repo: repo, hooks: hooks, logger: logger}
}

func (s *Service) Create(ctx context.Context, e *<Entity>) error {
    if err := s.hooks.Fire(ctx, "<entity>.before_create", e); err != nil {
        return err
    }
    if err := s.repo.Create(ctx, e); err != nil {
        return err
    }
    s.hooks.Fire(ctx, "<entity>.after_create", e)
    return nil
}
```

## handler.go

```go
package <entity>

import (
    "net/http"
    "github.com/go-chi/chi/v5"
    "github.com/go-playground/validator/v10"
    "go.uber.org/zap"
)

type Handler struct {
    svc      *Service
    validate *validator.Validate
    logger   *zap.Logger
}

func NewHandler(svc *Service, validate *validator.Validate, logger *zap.Logger) *Handler {
    return &Handler{svc: svc, validate: validate, logger: logger}
}

func (h *Handler) RegisterAdminRoutes(r chi.Router) {
    r.Get("/", h.list)
    r.Post("/", h.create)
    r.Get("/{id}", h.get)
    r.Put("/{id}", h.update)
    r.Delete("/{id}", h.delete)
}

// Lokale Response-Helpers (nicht shared, jeder Handler definiert sie selbst)
type apiResponse struct { Data interface{} `json:"data"` }
type apiError struct { Code string `json:"code"`; Detail string `json:"detail"` }

func writeJSON(w http.ResponseWriter, status int, v interface{}) { /* ... */ }
func writeError(w http.ResponseWriter, status int, code, detail string) { /* ... */ }
```

## DI-Wiring in `internal/app/app.go`

In `setupDomains()` hinzufügen:

```go
// 1. Repo
<entity>Repo := domain<entity>.NewPostgresRepository(pool, logger)

// 2. Service
<entity>Svc := domain<entity>.NewService(<entity>Repo, pluginRegistry.Hooks(), logger)

// 3. Handler
<entity>H := domain<entity>.NewHandler(<entity>Svc, validate, logger)

// 4. Routen mounten
r.Route("/api/v1/admin/<entities>", <entity>H.RegisterAdminRoutes)
r.Route("/api/v1/store/<entities>", <entity>H.RegisterStoreRoutes)
```

## Handler-Besonderheiten (existierende Domains)

- **category**: `RegisterAdminRoutes(r)` mountet relativ zu `/` → in `app.go` via `r.Route("/categories", handler.RegisterAdminRoutes)`
- **payment**: `NewHandler(methodSvc, transactionSvc, logger)` — zwei Services
- **cart**: `NewCartService(repo, productRepo, hooks, logger)` — `productRepo` satisfies privates `stockChecker`-Interface via `StockAvailable`
- **audit**: `NewService(repo, logger)` — kein HookRegistry
- **product.PostgresRepository**: `NewPostgresRepository(pool)` — kein Logger-Parameter

## Migration

Neue Tabelle in `migrations/000001_init.up.sql` hinzufügen (eine einzige Migration für das gesamte Schema):

```sql
CREATE TABLE <entities> (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    -- ... Felder
    custom_fields JSONB DEFAULT '{}',
    metadata      JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_<entities>_created_at ON <entities>(created_at DESC);
CREATE INDEX idx_<entities>_custom_fields ON <entities> USING GIN(custom_fields);
```

Für i18n-fähige Entities zusätzlich:

```sql
CREATE TABLE <entity>_translations (
    <entity>_id UUID NOT NULL REFERENCES <entities>(id) ON DELETE CASCADE,
    locale      VARCHAR(10) NOT NULL,
    -- ... übersetzte Felder
    PRIMARY KEY (<entity>_id, locale)
);
```

## Checkliste

- [ ] `internal/domain/<entity>/` mit 6 Dateien erstellt
- [ ] Sentinel Errors definiert (`ErrNotFound`, `ErrInvalid`)
- [ ] Hook-Events: `<entity>.before_create`, `<entity>.after_create` etc.
- [ ] Migration SQL ergänzt
- [ ] DI-Wiring in `app.go` (`setupDomains()`)
- [ ] Routen gemountet
- [ ] Tests in `*_test.go` (gleicher Package)
