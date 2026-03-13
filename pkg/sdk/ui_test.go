package sdk

import (
	"encoding/json"
	"testing"
)

func TestValidateUIExtension_ValidSchema(t *testing.T) {
	ext := UIExtension{
		ID:   "test_settings",
		Slot: "admin:payment:settings",
		Type: "schema",
		Schema: &UISchema{
			Fields: []UISchemaField{
				{
					Key:   "api_key",
					Type:  "password",
					Label: map[string]string{"en": "API Key", "de": "API-Schlüssel"},
				},
				{
					Key:   "mode",
					Type:  "select",
					Label: map[string]string{"en": "Mode"},
					Options: []UISelectOption{
						{Value: "test", Label: map[string]string{"en": "Test"}},
						{Value: "live", Label: map[string]string{"en": "Live"}},
					},
				},
			},
			SubmitURL: "/api/v1/admin/plugins/test/settings",
			LoadURL:   "/api/v1/admin/plugins/test/settings",
		},
	}

	if err := ValidateUIExtension("test", ext); err != nil {
		t.Fatalf("expected valid, got: %v", err)
	}
}

func TestValidateUIExtension_ValidComponent(t *testing.T) {
	ext := UIExtension{
		ID:   "test_checkout",
		Slot: "storefront:checkout:payment",
		Type: "component",
		Component: &UIComponent{
			TagName:   "stoa-test-checkout",
			ScriptURL: "/plugins/test/assets/checkout.js",
			Integrity: "sha256-abc123",
		},
	}

	if err := ValidateUIExtension("test", ext); err != nil {
		t.Fatalf("expected valid, got: %v", err)
	}
}

func TestValidateUIExtension_EmptyID(t *testing.T) {
	ext := UIExtension{
		ID:   "",
		Slot: "admin:test",
		Type: "schema",
		Schema: &UISchema{
			Fields: []UISchemaField{{Key: "k", Type: "text", Label: map[string]string{"en": "K"}}},
		},
	}

	if err := ValidateUIExtension("test", ext); err == nil {
		t.Fatal("expected error for empty ID")
	}
}

func TestValidateUIExtension_InvalidSlot(t *testing.T) {
	ext := UIExtension{
		ID:   "test_ext",
		Slot: "invalid:slot",
		Type: "schema",
		Schema: &UISchema{
			Fields: []UISchemaField{{Key: "k", Type: "text", Label: map[string]string{"en": "K"}}},
		},
	}

	if err := ValidateUIExtension("test", ext); err == nil {
		t.Fatal("expected error for invalid slot prefix")
	}
}

func TestValidateUIExtension_InvalidFieldType(t *testing.T) {
	ext := UIExtension{
		ID:   "test_ext",
		Slot: "admin:settings",
		Type: "schema",
		Schema: &UISchema{
			Fields: []UISchemaField{{Key: "k", Type: "unknown_type", Label: map[string]string{"en": "K"}}},
		},
	}

	if err := ValidateUIExtension("test", ext); err == nil {
		t.Fatal("expected error for invalid field type")
	}
}

func TestValidateUIExtension_EmptyFieldKey(t *testing.T) {
	ext := UIExtension{
		ID:   "test_ext",
		Slot: "admin:settings",
		Type: "schema",
		Schema: &UISchema{
			Fields: []UISchemaField{{Key: "", Type: "text", Label: map[string]string{"en": "K"}}},
		},
	}

	if err := ValidateUIExtension("test", ext); err == nil {
		t.Fatal("expected error for empty field key")
	}
}

func TestValidateUIExtension_InvalidType(t *testing.T) {
	ext := UIExtension{
		ID:   "test_ext",
		Slot: "admin:settings",
		Type: "invalid",
	}

	if err := ValidateUIExtension("test", ext); err == nil {
		t.Fatal("expected error for invalid extension type")
	}
}

func TestValidateUIExtension_SchemaNil(t *testing.T) {
	ext := UIExtension{
		ID:   "test_ext",
		Slot: "admin:settings",
		Type: "schema",
	}

	if err := ValidateUIExtension("test", ext); err == nil {
		t.Fatal("expected error for nil schema")
	}
}

func TestValidateUIExtension_ComponentNil(t *testing.T) {
	ext := UIExtension{
		ID:   "test_ext",
		Slot: "storefront:checkout:payment",
		Type: "component",
	}

	if err := ValidateUIExtension("test", ext); err == nil {
		t.Fatal("expected error for nil component")
	}
}

func TestValidateUIExtension_WrongTagNamePrefix(t *testing.T) {
	ext := UIExtension{
		ID:   "test_ext",
		Slot: "storefront:checkout:payment",
		Type: "component",
		Component: &UIComponent{
			TagName:   "stoa-other-checkout",
			ScriptURL: "/plugins/test/assets/checkout.js",
			Integrity: "sha256-abc",
		},
	}

	if err := ValidateUIExtension("test", ext); err == nil {
		t.Fatal("expected error for wrong tag name prefix")
	}
}

func TestValidateUIExtension_PathTraversal(t *testing.T) {
	ext := UIExtension{
		ID:   "test_ext",
		Slot: "admin:settings",
		Type: "schema",
		Schema: &UISchema{
			Fields:    []UISchemaField{{Key: "k", Type: "text", Label: map[string]string{"en": "K"}}},
			SubmitURL: "/api/../../etc/passwd",
		},
	}

	if err := ValidateUIExtension("test", ext); err == nil {
		t.Fatal("expected error for path traversal")
	}
}

func TestValidateUIExtension_AbsoluteURL(t *testing.T) {
	ext := UIExtension{
		ID:   "test_ext",
		Slot: "admin:settings",
		Type: "schema",
		Schema: &UISchema{
			Fields:    []UISchemaField{{Key: "k", Type: "text", Label: map[string]string{"en": "K"}}},
			SubmitURL: "https://evil.com/steal",
		},
	}

	if err := ValidateUIExtension("test", ext); err == nil {
		t.Fatal("expected error for absolute URL")
	}
}

func TestUIExtension_JSONSerialization(t *testing.T) {
	ext := UIExtension{
		ID:   "stripe_checkout",
		Slot: "storefront:checkout:payment",
		Type: "component",
		Component: &UIComponent{
			TagName:         "stoa-stripe-checkout",
			ScriptURL:       "/plugins/stripe/assets/checkout.js",
			Integrity:       "sha256-abc123",
			ExternalScripts: []string{"https://js.stripe.com/v3/"},
		},
	}

	data, err := json.Marshal(ext)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded UIExtension
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.ID != ext.ID {
		t.Errorf("ID: got %q, want %q", decoded.ID, ext.ID)
	}
	if decoded.Component == nil {
		t.Fatal("component is nil after unmarshal")
	}
	if decoded.Component.TagName != ext.Component.TagName {
		t.Errorf("TagName: got %q, want %q", decoded.Component.TagName, ext.Component.TagName)
	}
	if len(decoded.Component.ExternalScripts) != 1 {
		t.Fatalf("ExternalScripts: got %d, want 1", len(decoded.Component.ExternalScripts))
	}
	if decoded.Schema != nil {
		t.Error("schema should be nil for component type")
	}
}

func TestValidateUIExtension_ComponentPathTraversal(t *testing.T) {
	ext := UIExtension{
		ID:   "test_ext",
		Slot: "storefront:checkout:payment",
		Type: "component",
		Component: &UIComponent{
			TagName:   "stoa-test-checkout",
			ScriptURL: "/plugins/test/../../../etc/passwd",
			Integrity: "sha256-abc",
		},
	}

	if err := ValidateUIExtension("test", ext); err == nil {
		t.Fatal("expected error for script_url path traversal")
	}
}

func TestValidateUIExtension_AllFieldTypes(t *testing.T) {
	types := []string{"text", "password", "toggle", "select", "number", "textarea"}
	for _, ft := range types {
		ext := UIExtension{
			ID:   "test_ext",
			Slot: "admin:settings",
			Type: "schema",
			Schema: &UISchema{
				Fields: []UISchemaField{{Key: "field", Type: ft, Label: map[string]string{"en": "F"}}},
			},
		}
		if err := ValidateUIExtension("test", ext); err != nil {
			t.Errorf("field type %q should be valid, got: %v", ft, err)
		}
	}
}
