package plugin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stoa-hq/stoa/pkg/sdk"
)

// mockUIPlugin implements both sdk.Plugin and sdk.UIPlugin.
type mockUIPlugin struct {
	name       string
	extensions []sdk.UIExtension
}

func (m *mockUIPlugin) Name() string        { return m.name }
func (m *mockUIPlugin) Version() string      { return "1.0.0" }
func (m *mockUIPlugin) Description() string  { return "mock" }
func (m *mockUIPlugin) Init(*sdk.AppContext) error { return nil }
func (m *mockUIPlugin) Shutdown() error      { return nil }
func (m *mockUIPlugin) UIExtensions() []sdk.UIExtension { return m.extensions }

func TestCollectUIExtensions(t *testing.T) {
	logger := zerolog.Nop()
	reg := NewRegistry(logger)

	p := &mockUIPlugin{
		name: "test",
		extensions: []sdk.UIExtension{
			{
				ID:   "test_settings",
				Slot: "admin:payment:settings",
				Type: "schema",
				Schema: &sdk.UISchema{
					Fields: []sdk.UISchemaField{
						{Key: "api_key", Type: "password", Label: map[string]string{"en": "API Key"}},
					},
				},
			},
			{
				ID:   "test_checkout",
				Slot: "storefront:checkout:payment",
				Type: "component",
				Component: &sdk.UIComponent{
					TagName:   "stoa-test-checkout",
					ScriptURL: "/plugins/test/assets/checkout.js",
					Integrity: "sha256-abc",
				},
			},
		},
	}

	if err := reg.Register(p, &sdk.AppContext{}); err != nil {
		t.Fatalf("register: %v", err)
	}

	reg.CollectUIExtensions()
	exts := reg.UIExtensions()

	if len(exts) != 2 {
		t.Fatalf("expected 2 extensions, got %d", len(exts))
	}
}

func TestCollectUIExtensions_SkipsInvalid(t *testing.T) {
	logger := zerolog.Nop()
	reg := NewRegistry(logger)

	p := &mockUIPlugin{
		name: "test",
		extensions: []sdk.UIExtension{
			{
				ID:   "valid",
				Slot: "admin:settings",
				Type: "schema",
				Schema: &sdk.UISchema{
					Fields: []sdk.UISchemaField{{Key: "k", Type: "text", Label: map[string]string{"en": "K"}}},
				},
			},
			{
				ID:   "invalid",
				Slot: "bad:slot",
				Type: "schema",
				Schema: &sdk.UISchema{},
			},
		},
	}

	if err := reg.Register(p, &sdk.AppContext{}); err != nil {
		t.Fatalf("register: %v", err)
	}

	reg.CollectUIExtensions()
	exts := reg.UIExtensions()

	if len(exts) != 1 {
		t.Fatalf("expected 1 valid extension, got %d", len(exts))
	}
	if exts[0].ID != "valid" {
		t.Errorf("expected 'valid', got %q", exts[0].ID)
	}
}

func TestCollectUIExtensions_NoUIPlugins(t *testing.T) {
	logger := zerolog.Nop()
	reg := NewRegistry(logger)

	// Register a regular plugin (not UIPlugin)
	reg.CollectUIExtensions()
	exts := reg.UIExtensions()

	if exts != nil {
		t.Errorf("expected nil extensions, got %v", exts)
	}
}

func TestManifestHandler_StoreManifest(t *testing.T) {
	logger := zerolog.Nop()
	reg := NewRegistry(logger)

	p := &mockUIPlugin{
		name: "test",
		extensions: []sdk.UIExtension{
			{
				ID:   "test_checkout",
				Slot: "storefront:checkout:payment",
				Type: "component",
				Component: &sdk.UIComponent{
					TagName:   "stoa-test-checkout",
					ScriptURL: "/plugins/test/assets/checkout.js",
					Integrity: "sha256-abc",
				},
			},
			{
				ID:   "test_settings",
				Slot: "admin:payment:settings",
				Type: "schema",
				Schema: &sdk.UISchema{
					Fields: []sdk.UISchemaField{{Key: "k", Type: "text", Label: map[string]string{"en": "K"}}},
				},
			},
		},
	}

	if err := reg.Register(p, &sdk.AppContext{}); err != nil {
		t.Fatalf("register: %v", err)
	}
	reg.CollectUIExtensions()

	h := NewManifestHandler(reg)
	req := httptest.NewRequest("GET", "/api/v1/store/plugin-manifest", nil)
	rec := httptest.NewRecorder()

	h.StoreManifest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}

	var resp manifestResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(resp.Data.Extensions) != 1 {
		t.Fatalf("expected 1 storefront extension, got %d", len(resp.Data.Extensions))
	}
	if resp.Data.Extensions[0].ID != "test_checkout" {
		t.Errorf("expected test_checkout, got %q", resp.Data.Extensions[0].ID)
	}
}

func TestManifestHandler_AdminManifest(t *testing.T) {
	logger := zerolog.Nop()
	reg := NewRegistry(logger)

	p := &mockUIPlugin{
		name: "test",
		extensions: []sdk.UIExtension{
			{
				ID:   "test_checkout",
				Slot: "storefront:checkout:payment",
				Type: "component",
				Component: &sdk.UIComponent{
					TagName:   "stoa-test-checkout",
					ScriptURL: "/plugins/test/assets/checkout.js",
					Integrity: "sha256-abc",
				},
			},
			{
				ID:   "test_settings",
				Slot: "admin:payment:settings",
				Type: "schema",
				Schema: &sdk.UISchema{
					Fields: []sdk.UISchemaField{{Key: "k", Type: "text", Label: map[string]string{"en": "K"}}},
				},
			},
		},
	}

	if err := reg.Register(p, &sdk.AppContext{}); err != nil {
		t.Fatalf("register: %v", err)
	}
	reg.CollectUIExtensions()

	h := NewManifestHandler(reg)
	req := httptest.NewRequest("GET", "/api/v1/admin/plugin-manifest", nil)
	rec := httptest.NewRecorder()

	h.AdminManifest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}

	var resp manifestResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(resp.Data.Extensions) != 1 {
		t.Fatalf("expected 1 admin extension, got %d", len(resp.Data.Extensions))
	}
	if resp.Data.Extensions[0].ID != "test_settings" {
		t.Errorf("expected test_settings, got %q", resp.Data.Extensions[0].ID)
	}
}

func TestManifestHandler_EmptyManifest(t *testing.T) {
	logger := zerolog.Nop()
	reg := NewRegistry(logger)
	reg.CollectUIExtensions()

	h := NewManifestHandler(reg)
	req := httptest.NewRequest("GET", "/api/v1/store/plugin-manifest", nil)
	rec := httptest.NewRecorder()

	h.StoreManifest(rec, req)

	var resp manifestResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(resp.Data.Extensions) != 0 {
		t.Errorf("expected 0 extensions, got %d", len(resp.Data.Extensions))
	}
}
