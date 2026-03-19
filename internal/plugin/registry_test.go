package plugin

import (
	"context"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stoa-hq/stoa/pkg/sdk"
)

type mockPlugin struct {
	name    string
	version string
	initFn  func(app *sdk.AppContext) error
}

func (m *mockPlugin) Name() string        { return m.name }
func (m *mockPlugin) Version() string      { return m.version }
func (m *mockPlugin) Description() string  { return "mock plugin" }
func (m *mockPlugin) Init(app *sdk.AppContext) error {
	if m.initFn != nil {
		return m.initFn(app)
	}
	return nil
}
func (m *mockPlugin) Shutdown() error { return nil }

// mockSearchEngine implements sdk.SearchEngine for testing.
type mockSearchEngine struct{}

func (m *mockSearchEngine) Search(_ context.Context, _ sdk.SearchRequest) (*sdk.SearchResponse, error) {
	return &sdk.SearchResponse{}, nil
}
func (m *mockSearchEngine) Index(_ context.Context, _ string, _ string, _ map[string]interface{}) error {
	return nil
}
func (m *mockSearchEngine) Remove(_ context.Context, _ string, _ string) error {
	return nil
}

// mockSearchPlugin implements both sdk.Plugin and sdk.SearchPlugin.
type mockSearchPlugin struct {
	mockPlugin
	engine sdk.SearchEngine
}

func (m *mockSearchPlugin) SearchEngine() sdk.SearchEngine { return m.engine }

type panicNamePlugin struct{}

func (p *panicNamePlugin) Name() string                    { panic("name panic") }
func (p *panicNamePlugin) Version() string                 { return "0.0.0" }
func (p *panicNamePlugin) Description() string             { return "" }
func (p *panicNamePlugin) Init(_ *sdk.AppContext) error     { return nil }
func (p *panicNamePlugin) Shutdown() error                  { return nil }

func TestRegistry_Register_PanicRecovery(t *testing.T) {
	logger := zerolog.Nop()
	appCtx := &sdk.AppContext{}

	t.Run("panic in Init is recovered", func(t *testing.T) {
		reg := NewRegistry(logger)

		p := &mockPlugin{
			name:    "panic-init",
			version: "1.0.0",
			initFn: func(_ *sdk.AppContext) error {
				panic("init exploded")
			},
		}

		err := reg.Register(p, appCtx)
		if err == nil {
			t.Fatal("expected error from panicking plugin, got nil")
		}
		if !strings.Contains(err.Error(), "plugin panicked during registration") {
			t.Fatalf("unexpected error message: %s", err.Error())
		}

		// Plugin should not be registered
		if _, ok := reg.Get("panic-init"); ok {
			t.Error("panicking plugin should not be registered")
		}
	})

	t.Run("panic in Name is recovered", func(t *testing.T) {
		reg := NewRegistry(logger)

		err := reg.Register(&panicNamePlugin{}, appCtx)
		if err == nil {
			t.Fatal("expected error from panicking plugin, got nil")
		}
		if !strings.Contains(err.Error(), "plugin panicked during registration") {
			t.Fatalf("unexpected error message: %s", err.Error())
		}
	})

	t.Run("registry usable after panic", func(t *testing.T) {
		reg := NewRegistry(logger)

		// First: panicking plugin
		panicPlugin := &mockPlugin{
			name:    "bad-plugin",
			version: "1.0.0",
			initFn: func(_ *sdk.AppContext) error {
				panic("boom")
			},
		}
		_ = reg.Register(panicPlugin, appCtx)

		// Second: normal plugin should still register fine
		goodPlugin := &mockPlugin{
			name:    "good-plugin",
			version: "1.0.0",
		}
		if err := reg.Register(goodPlugin, appCtx); err != nil {
			t.Fatalf("expected no error registering good plugin after panic, got: %v", err)
		}
		if _, ok := reg.Get("good-plugin"); !ok {
			t.Error("good plugin should be registered")
		}
	})
}

func TestRegistry_SearchEngine(t *testing.T) {
	logger := zerolog.Nop()
	appCtx := &sdk.AppContext{}

	t.Run("returns nil when no search plugin registered", func(t *testing.T) {
		reg := NewRegistry(logger)

		// Register a regular plugin (not a SearchPlugin)
		regular := &mockPlugin{name: "regular", version: "1.0.0"}
		if err := reg.Register(regular, appCtx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if engine := reg.SearchEngine(); engine != nil {
			t.Error("expected nil engine when no search plugin registered")
		}
	})

	t.Run("returns engine from search plugin", func(t *testing.T) {
		reg := NewRegistry(logger)

		engine := &mockSearchEngine{}
		sp := &mockSearchPlugin{
			mockPlugin: mockPlugin{name: "meili", version: "1.0.0"},
			engine:     engine,
		}
		if err := reg.Register(sp, appCtx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := reg.SearchEngine()
		if got == nil {
			t.Fatal("expected non-nil engine")
		}
		if got != engine {
			t.Error("returned engine does not match registered engine")
		}
	})

	t.Run("returns first search plugin engine", func(t *testing.T) {
		reg := NewRegistry(logger)

		engine1 := &mockSearchEngine{}
		sp1 := &mockSearchPlugin{
			mockPlugin: mockPlugin{name: "first-search", version: "1.0.0"},
			engine:     engine1,
		}
		engine2 := &mockSearchEngine{}
		sp2 := &mockSearchPlugin{
			mockPlugin: mockPlugin{name: "second-search", version: "1.0.0"},
			engine:     engine2,
		}

		if err := reg.Register(sp1, appCtx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := reg.Register(sp2, appCtx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := reg.SearchEngine()
		if got != engine1 {
			t.Error("expected engine from first registered search plugin")
		}
	})

	t.Run("skips non-search plugins", func(t *testing.T) {
		reg := NewRegistry(logger)

		// Register regular plugin first
		regular := &mockPlugin{name: "analytics", version: "1.0.0"}
		if err := reg.Register(regular, appCtx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Then a search plugin
		engine := &mockSearchEngine{}
		sp := &mockSearchPlugin{
			mockPlugin: mockPlugin{name: "meili", version: "1.0.0"},
			engine:     engine,
		}
		if err := reg.Register(sp, appCtx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := reg.SearchEngine()
		if got != engine {
			t.Error("expected engine from search plugin, skipping non-search plugin")
		}
	})

	t.Run("returns nil for empty registry", func(t *testing.T) {
		reg := NewRegistry(logger)
		if engine := reg.SearchEngine(); engine != nil {
			t.Error("expected nil engine for empty registry")
		}
	})
}
