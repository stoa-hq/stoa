package product

import (
	"testing"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// cartesianProduct
// ---------------------------------------------------------------------------

func TestCartesianProduct_TwoGroups(t *testing.T) {
	a, b, c, d := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	result := cartesianProduct([][]uuid.UUID{{a, b}, {c, d}})
	if len(result) != 4 {
		t.Fatalf("want 4 combinations, got %d", len(result))
	}
}

func TestCartesianProduct_SingleGroup(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	result := cartesianProduct([][]uuid.UUID{{a, b}})
	if len(result) != 2 {
		t.Fatalf("want 2 combinations, got %d", len(result))
	}
}

func TestCartesianProduct_ThreeGroups(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	// 2 × 2 × 2 = 8
	result := cartesianProduct([][]uuid.UUID{{a, b}, {a, b}, {b, c}})
	if len(result) != 8 {
		t.Fatalf("want 8 combinations, got %d", len(result))
	}
}

func TestCartesianProduct_Empty(t *testing.T) {
	if result := cartesianProduct(nil); result != nil {
		t.Errorf("want nil for empty input, got %v", result)
	}
}

func TestCartesianProduct_EachComboHasOnePerGroup(t *testing.T) {
	a, b, c, d := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	result := cartesianProduct([][]uuid.UUID{{a, b}, {c, d}})
	for _, combo := range result {
		if len(combo) != 2 {
			t.Errorf("each combination should have 2 elements, got %d", len(combo))
		}
	}
}

// ---------------------------------------------------------------------------
// ToResponse
// ---------------------------------------------------------------------------

func TestToResponse_CoreFields(t *testing.T) {
	id := uuid.New()
	p := &Product{
		ID:       id,
		SKU:      "TEST-001",
		Active:   true,
		PriceNet: 1000,
		Currency: "EUR",
		Stock:    5,
	}

	resp := ToResponse(p)

	if resp.ID != id {
		t.Errorf("ID: got %s, want %s", resp.ID, id)
	}
	if resp.SKU != "TEST-001" {
		t.Errorf("SKU: got %q, want TEST-001", resp.SKU)
	}
	if resp.PriceNet != 1000 {
		t.Errorf("PriceNet: got %d, want 1000", resp.PriceNet)
	}
	if resp.Currency != "EUR" {
		t.Errorf("Currency: got %q, want EUR", resp.Currency)
	}
	if !resp.Active {
		t.Error("Active should be true")
	}
}

func TestToResponse_Translations(t *testing.T) {
	p := &Product{
		ID: uuid.New(),
		Translations: []ProductTranslation{
			{Locale: "de-DE", Name: "Produkt", Slug: "produkt"},
			{Locale: "en-US", Name: "Product", Slug: "product"},
		},
	}

	resp := ToResponse(p)
	if len(resp.Translations) != 2 {
		t.Fatalf("want 2 translations, got %d", len(resp.Translations))
	}
	if resp.Translations[0].Locale != "de-DE" || resp.Translations[0].Slug != "produkt" {
		t.Errorf("unexpected first translation: %+v", resp.Translations[0])
	}
}

// ---------------------------------------------------------------------------
// FromCreateRequest
// ---------------------------------------------------------------------------

func TestFromCreateRequest_FieldMapping(t *testing.T) {
	req := &CreateProductRequest{
		SKU:      "SKU-42",
		Active:   true,
		PriceNet: 999,
		Currency: "USD",
		Stock:    10,
		Translations: []TranslationInput{
			{Locale: "en", Name: "Widget", Slug: "widget"},
		},
	}

	p := FromCreateRequest(req)

	if p.SKU != "SKU-42" {
		t.Errorf("SKU: got %q, want SKU-42", p.SKU)
	}
	if p.Currency != "USD" {
		t.Errorf("Currency: got %q, want USD", p.Currency)
	}
	if p.PriceNet != 999 {
		t.Errorf("PriceNet: got %d, want 999", p.PriceNet)
	}
	if p.Stock != 10 {
		t.Errorf("Stock: got %d, want 10", p.Stock)
	}
	if len(p.Translations) != 1 || p.Translations[0].Slug != "widget" {
		t.Errorf("Translations: %+v", p.Translations)
	}
}

// ---------------------------------------------------------------------------
// ApplyUpdateRequest
// ---------------------------------------------------------------------------

func TestApplyUpdateRequest_PartialUpdate(t *testing.T) {
	p := &Product{
		SKU:      "OLD",
		PriceNet: 100,
		Currency: "EUR",
		Active:   false,
	}

	newSKU := "NEW"
	newPrice := 200
	active := true

	ApplyUpdateRequest(p, &UpdateProductRequest{
		SKU:      &newSKU,
		PriceNet: &newPrice,
		Active:   &active,
		// Currency intentionally omitted → must remain "EUR"
	})

	if p.SKU != "NEW" {
		t.Errorf("SKU: got %q, want NEW", p.SKU)
	}
	if p.PriceNet != 200 {
		t.Errorf("PriceNet: got %d, want 200", p.PriceNet)
	}
	if !p.Active {
		t.Error("Active should be true after update")
	}
	if p.Currency != "EUR" {
		t.Errorf("Currency should remain EUR, got %q", p.Currency)
	}
}

func TestApplyUpdateRequest_TranslationsReplaced(t *testing.T) {
	p := &Product{
		ID:           uuid.New(),
		Translations: []ProductTranslation{{Locale: "de-DE", Name: "Alt"}},
	}

	ApplyUpdateRequest(p, &UpdateProductRequest{
		Translations: []TranslationInput{
			{Locale: "en-US", Name: "New", Slug: "new"},
		},
	})

	if len(p.Translations) != 1 {
		t.Fatalf("want 1 translation, got %d", len(p.Translations))
	}
	if p.Translations[0].Locale != "en-US" {
		t.Errorf("expected en-US translation after replace, got %q", p.Translations[0].Locale)
	}
}

func TestApplyUpdateRequest_NilFieldsUnchanged(t *testing.T) {
	p := &Product{
		SKU:        "ORIGINAL",
		PriceGross: 500,
	}

	// Empty request — nothing should change.
	ApplyUpdateRequest(p, &UpdateProductRequest{})

	if p.SKU != "ORIGINAL" {
		t.Errorf("SKU should be unchanged, got %q", p.SKU)
	}
	if p.PriceGross != 500 {
		t.Errorf("PriceGross should be unchanged, got %d", p.PriceGross)
	}
}
