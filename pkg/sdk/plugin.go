package sdk

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

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
	DB     *pgxpool.Pool
	Router chi.Router
	Hooks  *HookRegistry
	Config map[string]interface{}
	Logger zerolog.Logger
}
