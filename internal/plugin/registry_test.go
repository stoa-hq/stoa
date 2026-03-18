package plugin

import (
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
