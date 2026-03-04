package plugin

import (
	"fmt"
	"sync"

	"github.com/rs/zerolog"
)

type Registry struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
	order   []string
	hooks   *HookRegistry
	logger  zerolog.Logger
}

func NewRegistry(logger zerolog.Logger) *Registry {
	return &Registry{
		plugins: make(map[string]Plugin),
		hooks:   NewHookRegistry(),
		logger:  logger,
	}
}

func (r *Registry) Hooks() *HookRegistry {
	return r.hooks
}

func (r *Registry) Register(p Plugin, appCtx *AppContext) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := p.Name()
	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin %q already registered", name)
	}

	appCtx.Hooks = r.hooks

	if err := p.Init(appCtx); err != nil {
		return fmt.Errorf("initializing plugin %q: %w", name, err)
	}

	r.plugins[name] = p
	r.order = append(r.order, name)
	r.logger.Info().Str("plugin", name).Str("version", p.Version()).Msg("plugin registered")

	return nil
}

func (r *Registry) Get(name string) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.plugins[name]
	return p, ok
}

func (r *Registry) List() []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Plugin, 0, len(r.order))
	for _, name := range r.order {
		result = append(result, r.plugins[name])
	}
	return result
}

func (r *Registry) ShutdownAll() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Shutdown in reverse order
	for i := len(r.order) - 1; i >= 0; i-- {
		name := r.order[i]
		if err := r.plugins[name].Shutdown(); err != nil {
			r.logger.Error().Err(err).Str("plugin", name).Msg("plugin shutdown error")
		}
	}
	return nil
}
