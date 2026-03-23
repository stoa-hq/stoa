package sdk

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// CheckoutFn performs a full store checkout programmatically.
// Plugins use this to trigger order creation with all server-side
// enforcement (price validation, tax, stock deduction, hooks).
// customerID may be nil for guest checkouts.
// req is a JSON-encoded CheckoutRequest, result is a JSON-encoded order response.
type CheckoutFn func(ctx context.Context, customerID *uuid.UUID, req json.RawMessage) (json.RawMessage, error)

// Plugin is the interface that all plugins must implement.
type Plugin interface {
	Name() string
	Version() string
	Description() string
	Init(app *AppContext) error
	Shutdown() error
}

// AppContext provides plugins access to application resources.
type AppContext struct {
	DB           *pgxpool.Pool
	Router       chi.Router
	AssetRouter  chi.Router // mounted under /plugins/{name}/assets/
	Hooks        *HookRegistry
	Config       map[string]interface{}
	Logger       zerolog.Logger
	Auth         *AuthHelper
	CheckoutFn   CheckoutFn
	SecureCookie bool // true when running behind HTTPS; use for cookie Secure flag
}

// AuthHelper gives plugins access to authentication middleware and context
// helpers without importing internal/auth.
type AuthHelper struct {
	// OptionalAuth is middleware that extracts auth info if present but never blocks.
	OptionalAuth func(http.Handler) http.Handler
	// Required is middleware that requires a valid token; returns 401 otherwise.
	Required func(http.Handler) http.Handler
	// RequireRole is middleware that checks the user has one of the given roles.
	// Roles are string constants: "super_admin", "admin", "manager", "customer", "api_client".
	RequireRole func(roles ...string) func(http.Handler) http.Handler
	// UserID extracts the authenticated user's UUID from the request context.
	UserID func(ctx context.Context) uuid.UUID
	// UserType returns "admin", "customer", or "api_key" from the request context.
	UserType func(ctx context.Context) string
}
