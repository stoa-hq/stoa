package plugin

import (
	"fmt"
	"sync"

	"github.com/rs/zerolog"
	"github.com/stoa-hq/stoa/pkg/sdk"
)

type Registry struct {
	mu           sync.RWMutex
	plugins      map[string]Plugin
	order        []string
	hooks        *HookRegistry
	logger       zerolog.Logger
	uiExtensions []sdk.UIExtension
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

func (r *Registry) Register(p Plugin, appCtx *AppContext) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("plugin panicked during registration: %v", rec)
		}
	}()

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

// CollectUIExtensions iterates over all registered plugins, checks for the
// UIPlugin interface, validates extensions, and caches them. Call once after
// all plugins have been registered.
func (r *Registry) CollectUIExtensions() {
	r.mu.Lock()
	defer r.mu.Unlock()

	var exts []sdk.UIExtension
	for _, name := range r.order {
		p := r.plugins[name]
		uiPlugin, ok := p.(sdk.UIPlugin)
		if !ok {
			continue
		}

		for _, ext := range uiPlugin.UIExtensions() {
			if err := sdk.ValidateUIExtension(name, ext); err != nil {
				r.logger.Warn().Err(err).Str("plugin", name).Msg("invalid UI extension, skipping")
				continue
			}
			sdk.SanitizeUIExtension(&ext)
			exts = append(exts, ext)
		}
		r.logger.Info().Str("plugin", name).Int("extensions", len(exts)).Msg("collected UI extensions")
	}
	r.uiExtensions = exts
}

// UIExtensions returns all validated UI extensions collected from plugins.
func (r *Registry) UIExtensions() []sdk.UIExtension {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.uiExtensions
}

// SearchEngine returns the SearchEngine from the first registered SearchPlugin,
// or nil if no SearchPlugin is registered.
func (r *Registry) SearchEngine() sdk.SearchEngine {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, name := range r.order {
		if sp, ok := r.plugins[name].(sdk.SearchPlugin); ok {
			return sp.SearchEngine()
		}
	}
	return nil
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
