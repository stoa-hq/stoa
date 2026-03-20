package admin

import (
	"testing"
)

func TestTransformProductArgs_Minimal(t *testing.T) {
	args := map[string]any{
		"name":  "Test Product",
		"slug":  "test-product",
		"price": float64(1999),
	}
	result := transformProductArgs(args)

	if result["price_net"] != 1999 {
		t.Errorf("price_net = %v, want 1999", result["price_net"])
	}
	if result["price_gross"] != 1999 {
		t.Errorf("price_gross = %v, want 1999", result["price_gross"])
	}
	if result["currency"] != "EUR" {
		t.Errorf("currency = %v, want EUR", result["currency"])
	}
	if _, ok := result["price"]; ok {
		t.Error("price should be removed")
	}
	if _, ok := result["name"]; ok {
		t.Error("top-level name should be removed")
	}
	if _, ok := result["slug"]; ok {
		t.Error("top-level slug should be removed")
	}

	translations, ok := result["translations"].([]any)
	if !ok || len(translations) != 1 {
		t.Fatalf("translations = %v, want []any with 1 element", result["translations"])
	}
	tr := translations[0].(map[string]any)
	if tr["locale"] != "en-US" {
		t.Errorf("locale = %v, want en-US", tr["locale"])
	}
	if tr["name"] != "Test Product" {
		t.Errorf("name = %v, want Test Product", tr["name"])
	}
	if tr["slug"] != "test-product" {
		t.Errorf("slug = %v, want test-product", tr["slug"])
	}
}

func TestTransformProductArgs_TranslationsObjectFormat(t *testing.T) {
	args := map[string]any{
		"price": float64(2999),
		"translations": map[string]any{
			"de-DE": map[string]any{"name": "Testprodukt", "slug": "testprodukt"},
			"en-US": map[string]any{"name": "Test Product", "slug": "test-product"},
		},
	}
	result := transformProductArgs(args)

	translations, ok := result["translations"].([]any)
	if !ok || len(translations) != 2 {
		t.Fatalf("translations = %v, want []any with 2 elements", result["translations"])
	}

	locales := map[string]bool{}
	for _, tr := range translations {
		entry := tr.(map[string]any)
		locales[entry["locale"].(string)] = true
	}
	if !locales["de-DE"] || !locales["en-US"] {
		t.Errorf("expected de-DE and en-US locales, got %v", locales)
	}
}

func TestTransformProductArgs_TranslationsArrayPassthrough(t *testing.T) {
	args := map[string]any{
		"price": float64(999),
		"translations": []any{
			map[string]any{"locale": "en-US", "name": "Test", "slug": "test"},
		},
	}
	result := transformProductArgs(args)

	translations, ok := result["translations"].([]any)
	if !ok || len(translations) != 1 {
		t.Fatalf("translations = %v, want []any with 1 element", result["translations"])
	}
}

func TestTransformProductArgs_ExplicitTranslationsOverrideTopLevel(t *testing.T) {
	args := map[string]any{
		"name":  "Ignored",
		"slug":  "ignored",
		"price": float64(100),
		"translations": map[string]any{
			"de-DE": map[string]any{"name": "Behalten", "slug": "behalten"},
		},
	}
	result := transformProductArgs(args)

	translations := result["translations"].([]any)
	if len(translations) != 1 {
		t.Fatalf("expected 1 translation, got %d", len(translations))
	}
	tr := translations[0].(map[string]any)
	if tr["name"] != "Behalten" {
		t.Errorf("name = %v, want Behalten", tr["name"])
	}
}

func TestTransformProductArgs_ExplicitCurrencyPreserved(t *testing.T) {
	args := map[string]any{
		"name":     "Test",
		"slug":     "test",
		"price":    float64(500),
		"currency": "USD",
	}
	result := transformProductArgs(args)
	if result["currency"] != "USD" {
		t.Errorf("currency = %v, want USD", result["currency"])
	}
}

func TestTransformProductArgs_ExplicitPriceNetGrossPreserved(t *testing.T) {
	args := map[string]any{
		"name":        "Test",
		"slug":        "test",
		"price":       float64(1999),
		"price_net":   float64(1680),
		"price_gross": float64(1999),
	}
	result := transformProductArgs(args)
	if result["price_net"] != float64(1680) {
		t.Errorf("price_net = %v, want 1680", result["price_net"])
	}
	if result["price_gross"] != float64(1999) {
		t.Errorf("price_gross = %v, want 1999", result["price_gross"])
	}
}

func TestTransformVariantArgs(t *testing.T) {
	args := map[string]any{
		"sku":   "VAR-001",
		"price": float64(599),
		"stock": float64(10),
	}
	result := transformVariantArgs(args)

	if result["price_net"] != 599 {
		t.Errorf("price_net = %v, want 599", result["price_net"])
	}
	if result["price_gross"] != 599 {
		t.Errorf("price_gross = %v, want 599", result["price_gross"])
	}
	if _, ok := result["price"]; ok {
		t.Error("price should be removed")
	}
	if result["sku"] != "VAR-001" {
		t.Errorf("sku = %v, want VAR-001", result["sku"])
	}
}

func TestTransformCategoryArgs(t *testing.T) {
	args := map[string]any{
		"name":        "Electronics",
		"slug":        "electronics",
		"description": "All electronics",
		"active":      true,
	}
	result := transformCategoryArgs(args)

	if _, ok := result["name"]; ok {
		t.Error("top-level name should be removed")
	}
	if _, ok := result["slug"]; ok {
		t.Error("top-level slug should be removed")
	}
	if _, ok := result["description"]; ok {
		t.Error("top-level description should be removed")
	}
	if result["active"] != true {
		t.Error("active should be preserved")
	}

	translations, ok := result["translations"].([]any)
	if !ok || len(translations) != 1 {
		t.Fatalf("translations = %v, want []any with 1 element", result["translations"])
	}
	tr := translations[0].(map[string]any)
	if tr["locale"] != "en-US" {
		t.Errorf("locale = %v, want en-US", tr["locale"])
	}
	if tr["name"] != "Electronics" {
		t.Errorf("name = %v, want Electronics", tr["name"])
	}
	if tr["slug"] != "electronics" {
		t.Errorf("slug = %v, want electronics", tr["slug"])
	}
	if tr["description"] != "All electronics" {
		t.Errorf("description = %v, want All electronics", tr["description"])
	}
}

func TestTransformCategoryArgs_WithTranslationsObject(t *testing.T) {
	args := map[string]any{
		"translations": map[string]any{
			"de-DE": map[string]any{"name": "Elektronik", "slug": "elektronik"},
		},
		"active": true,
	}
	result := transformCategoryArgs(args)

	translations := result["translations"].([]any)
	if len(translations) != 1 {
		t.Fatalf("expected 1 translation, got %d", len(translations))
	}
	tr := translations[0].(map[string]any)
	if tr["locale"] != "de-DE" {
		t.Errorf("locale = %v, want de-DE", tr["locale"])
	}
}

func TestTransformPropertyGroupArgs_NameOnly(t *testing.T) {
	args := map[string]any{
		"name":     "Color",
		"position": float64(1),
	}
	result := transformPropertyGroupArgs(args)

	if _, ok := result["name"]; ok {
		t.Error("top-level name should be removed")
	}
	if result["position"] != float64(1) {
		t.Errorf("position = %v, want 1", result["position"])
	}

	translations, ok := result["translations"].([]any)
	if !ok || len(translations) != 1 {
		t.Fatalf("translations = %v, want []any with 1 element", result["translations"])
	}
	tr := translations[0].(map[string]any)
	if tr["locale"] != "en-US" {
		t.Errorf("locale = %v, want en-US", tr["locale"])
	}
	if tr["name"] != "Color" {
		t.Errorf("name = %v, want Color", tr["name"])
	}
}

func TestTransformPropertyGroupArgs_WithTranslations(t *testing.T) {
	args := map[string]any{
		"translations": map[string]any{
			"de-DE": map[string]any{"name": "Farbe"},
			"en-US": map[string]any{"name": "Color"},
		},
	}
	result := transformPropertyGroupArgs(args)

	translations, ok := result["translations"].([]any)
	if !ok || len(translations) != 2 {
		t.Fatalf("translations = %v, want []any with 2 elements", result["translations"])
	}

	locales := map[string]bool{}
	for _, tr := range translations {
		entry := tr.(map[string]any)
		locales[entry["locale"].(string)] = true
	}
	if !locales["de-DE"] || !locales["en-US"] {
		t.Errorf("expected de-DE and en-US locales, got %v", locales)
	}
}

func TestTransformPropertyOptionArgs_NameOnly(t *testing.T) {
	args := map[string]any{
		"name":      "Red",
		"color_hex": "#FF0000",
		"position":  float64(0),
	}
	result := transformPropertyOptionArgs(args)

	if _, ok := result["name"]; ok {
		t.Error("top-level name should be removed")
	}
	if result["color_hex"] != "#FF0000" {
		t.Errorf("color_hex = %v, want #FF0000", result["color_hex"])
	}

	translations, ok := result["translations"].([]any)
	if !ok || len(translations) != 1 {
		t.Fatalf("translations = %v, want []any with 1 element", result["translations"])
	}
	tr := translations[0].(map[string]any)
	if tr["locale"] != "en-US" {
		t.Errorf("locale = %v, want en-US", tr["locale"])
	}
	if tr["name"] != "Red" {
		t.Errorf("name = %v, want Red", tr["name"])
	}
}

func TestTransformPropertyOptionArgs_WithTranslations(t *testing.T) {
	args := map[string]any{
		"color_hex": "#00FF00",
		"translations": map[string]any{
			"de-DE": map[string]any{"name": "Gruen"},
		},
	}
	result := transformPropertyOptionArgs(args)

	if result["color_hex"] != "#00FF00" {
		t.Errorf("color_hex = %v, want #00FF00", result["color_hex"])
	}

	translations := result["translations"].([]any)
	if len(translations) != 1 {
		t.Fatalf("expected 1 translation, got %d", len(translations))
	}
	tr := translations[0].(map[string]any)
	if tr["locale"] != "de-DE" {
		t.Errorf("locale = %v, want de-DE", tr["locale"])
	}
	if tr["name"] != "Gruen" {
		t.Errorf("name = %v, want Gruen", tr["name"])
	}
}

func TestToInt(t *testing.T) {
	tests := []struct {
		input any
		want  int
	}{
		{float64(1999), 1999},
		{int(42), 42},
		{int64(100), 100},
		{"not a number", 0},
		{nil, 0},
	}
	for _, tt := range tests {
		got := toInt(tt.input)
		if got != tt.want {
			t.Errorf("toInt(%v) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
