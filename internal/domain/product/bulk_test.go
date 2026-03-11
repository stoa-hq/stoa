package product

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// BulkCreate
// ---------------------------------------------------------------------------

func TestService_BulkCreate_AllSucceed(t *testing.T) {
	repo := &mockRepo{
		create: func(_ context.Context, _ *Product) error { return nil },
	}
	svc := newTestService(repo)

	reqs := []BulkCreateProductRequest{
		{CreateProductRequest: CreateProductRequest{SKU: "A", Currency: "EUR", Translations: []TranslationInput{{Locale: "de-DE", Name: "Produkt A", Slug: "produkt-a"}}}},
		{CreateProductRequest: CreateProductRequest{SKU: "B", Currency: "EUR", Translations: []TranslationInput{{Locale: "de-DE", Name: "Produkt B", Slug: "produkt-b"}}}},
	}

	resp := svc.BulkCreate(context.Background(), reqs)

	if resp.Total != 2 {
		t.Errorf("Total: got %d, want 2", resp.Total)
	}
	if resp.Succeeded != 2 {
		t.Errorf("Succeeded: got %d, want 2", resp.Succeeded)
	}
	if resp.Failed != 0 {
		t.Errorf("Failed: got %d, want 0", resp.Failed)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("Results length: got %d, want 2", len(resp.Results))
	}
	for i, r := range resp.Results {
		if !r.Success {
			t.Errorf("Results[%d].Success: got false, want true", i)
		}
		if r.ID == "" {
			t.Errorf("Results[%d].ID: expected non-empty ID", i)
		}
	}
}

func TestService_BulkCreate_PartialFailure(t *testing.T) {
	callCount := 0
	repo := &mockRepo{
		create: func(_ context.Context, p *Product) error {
			callCount++
			if p.SKU == "FAIL" {
				return errors.New("db error")
			}
			return nil
		},
	}
	svc := newTestService(repo)

	reqs := []BulkCreateProductRequest{
		{CreateProductRequest: CreateProductRequest{SKU: "OK", Currency: "EUR", Translations: []TranslationInput{{Locale: "de-DE", Name: "OK", Slug: "ok"}}}},
		{CreateProductRequest: CreateProductRequest{SKU: "FAIL", Currency: "EUR", Translations: []TranslationInput{{Locale: "de-DE", Name: "Fail", Slug: "fail"}}}},
		{CreateProductRequest: CreateProductRequest{SKU: "OK2", Currency: "EUR", Translations: []TranslationInput{{Locale: "de-DE", Name: "OK2", Slug: "ok2"}}}},
	}

	resp := svc.BulkCreate(context.Background(), reqs)

	if resp.Total != 3 {
		t.Errorf("Total: got %d, want 3", resp.Total)
	}
	if resp.Succeeded != 2 {
		t.Errorf("Succeeded: got %d, want 2", resp.Succeeded)
	}
	if resp.Failed != 1 {
		t.Errorf("Failed: got %d, want 1", resp.Failed)
	}
	if resp.Results[1].Success {
		t.Error("Results[1].Success: got true, want false")
	}
	if len(resp.Results[1].Errors) == 0 {
		t.Error("Results[1].Errors: expected errors for failed product")
	}
}

func TestService_BulkCreate_Upsert(t *testing.T) {
	existingID := uuid.New()
	updateCalled := false

	repo := &mockRepo{
		findBySKU: func(_ context.Context, sku string) (*Product, error) {
			if sku == "EXISTING" {
				return &Product{
					ID:  existingID,
					SKU: "EXISTING",
					Translations: []ProductTranslation{
						{ProductID: existingID, Locale: "de-DE", Name: "Alt", Slug: "alt"},
					},
				}, nil
			}
			return nil, ErrNotFound
		},
		create: func(_ context.Context, _ *Product) error { return nil },
		update: func(_ context.Context, p *Product) error {
			updateCalled = true
			if p.ID != existingID {
				t.Errorf("update called with wrong ID: %v", p.ID)
			}
			if len(p.Translations) == 0 || p.Translations[0].Name != "Neu" {
				t.Errorf("update: translations not applied, got %+v", p.Translations)
			}
			return nil
		},
	}
	svc := newTestService(repo)

	reqs := []BulkCreateProductRequest{
		{CreateProductRequest: CreateProductRequest{SKU: "EXISTING", Currency: "EUR", Translations: []TranslationInput{{Locale: "de-DE", Name: "Neu", Slug: "neu"}}}},
		{CreateProductRequest: CreateProductRequest{SKU: "NEW", Currency: "EUR", Translations: []TranslationInput{{Locale: "de-DE", Name: "Neu Produkt", Slug: "neu-produkt"}}}},
	}

	resp := svc.BulkCreate(context.Background(), reqs)

	if resp.Total != 2 {
		t.Errorf("Total: got %d, want 2", resp.Total)
	}
	if resp.Succeeded != 2 {
		t.Errorf("Succeeded: got %d, want 2", resp.Succeeded)
	}
	if !updateCalled {
		t.Error("update was not called for existing product")
	}
	if resp.Results[0].ID != existingID.String() {
		t.Errorf("Results[0].ID: got %s, want %s", resp.Results[0].ID, existingID.String())
	}
}

func TestService_BulkCreate_WithVariants(t *testing.T) {
	repo := &mockRepo{
		create: func(_ context.Context, _ *Product) error { return nil },
	}
	svc := newTestService(repo)

	reqs := []BulkCreateProductRequest{
		{
			CreateProductRequest: CreateProductRequest{
				SKU:      "SHIRT",
				Currency: "EUR",
				Translations: []TranslationInput{
					{Locale: "de-DE", Name: "Shirt", Slug: "shirt"},
				},
			},
			Variants: []BulkImportVariantInput{
				{
					SKU:    "SHIRT-S",
					Active: true,
					Stock:  10,
					Options: []BulkImportOptionInput{
						{GroupName: "Größe", OptionName: "S", Locale: "de"},
					},
				},
				{
					SKU:    "SHIRT-M",
					Active: true,
					Stock:  20,
					Options: []BulkImportOptionInput{
						{GroupName: "Größe", OptionName: "M", Locale: "de"},
					},
				},
			},
		},
	}

	resp := svc.BulkCreate(context.Background(), reqs)

	if resp.Succeeded != 1 {
		t.Errorf("Succeeded: got %d, want 1", resp.Succeeded)
	}
	if resp.Results[0].ID == "" {
		t.Error("Results[0].ID: expected non-empty")
	}
}

// ---------------------------------------------------------------------------
// ParseCSV
// ---------------------------------------------------------------------------

func TestParseCSV_SimpleProduct(t *testing.T) {
	repo := &mockRepo{
		create: func(_ context.Context, _ *Product) error { return nil },
	}
	svc := newTestService(repo)

	csvData := `sku,active,price_net,price_gross,currency,tax_rule_id,stock,weight,locale,name,slug,description,meta_title,meta_description,category_ids,tag_ids,variant_sku,variant_active,variant_stock,variant_price_net,variant_price_gross,option_1_group,option_1_value,option_2_group,option_2_value,option_3_group,option_3_value
PROD-001,true,1999,2499,EUR,,100,300,de-DE,Testprodukt,testprodukt,Beschreibung,,,,,,,,,,,,,,`

	resp, err := svc.ParseCSV(context.Background(), strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("ParseCSV error: %v", err)
	}

	if resp.Total != 1 {
		t.Errorf("Total: got %d, want 1", resp.Total)
	}
	if resp.Succeeded != 1 {
		t.Errorf("Succeeded: got %d, want 1", resp.Succeeded)
	}
}

func TestParseCSV_WithVariants(t *testing.T) {
	repo := &mockRepo{
		create: func(_ context.Context, _ *Product) error { return nil },
	}
	svc := newTestService(repo)

	csvData := `sku,active,price_net,price_gross,currency,tax_rule_id,stock,weight,locale,name,slug,description,meta_title,meta_description,category_ids,tag_ids,variant_sku,variant_active,variant_stock,variant_price_net,variant_price_gross,option_1_group,option_1_value,option_2_group,option_2_value,option_3_group,option_3_value
SHIRT,true,1999,2499,EUR,,50,300,de-DE,T-Shirt,t-shirt,,,,,,,,,,,,,,,
,,,,,,,,,,,,,,,,SHIRT-S,true,10,,,Größe,S,,,
,,,,,,,,,,,,,,,,SHIRT-M,true,20,,,Größe,M,,,`

	resp, err := svc.ParseCSV(context.Background(), strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("ParseCSV error: %v", err)
	}

	if resp.Total != 1 {
		t.Errorf("Total: got %d, want 1", resp.Total)
	}
	if resp.Succeeded != 1 {
		t.Errorf("Succeeded: got %d, want 1", resp.Succeeded)
	}
}

func TestParseCSV_MultipleProducts(t *testing.T) {
	repo := &mockRepo{
		create: func(_ context.Context, _ *Product) error { return nil },
	}
	svc := newTestService(repo)

	csvData := `sku,active,price_net,price_gross,currency,tax_rule_id,stock,weight,locale,name,slug,description,meta_title,meta_description,category_ids,tag_ids,variant_sku,variant_active,variant_stock,variant_price_net,variant_price_gross,option_1_group,option_1_value,option_2_group,option_2_value,option_3_group,option_3_value
P1,true,999,1199,EUR,,10,100,de-DE,Produkt 1,produkt-1,,,,,,,,,,,,,,,
P2,false,2999,3599,EUR,,5,500,de-DE,Produkt 2,produkt-2,,,,,,,,,,,,,,,`

	resp, err := svc.ParseCSV(context.Background(), strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("ParseCSV error: %v", err)
	}

	if resp.Total != 2 {
		t.Errorf("Total: got %d, want 2", resp.Total)
	}
	if resp.Succeeded != 2 {
		t.Errorf("Succeeded: got %d, want 2", resp.Succeeded)
	}
}

func TestParseCSV_EmptyCSV(t *testing.T) {
	svc := newTestService(&mockRepo{})

	_, err := svc.ParseCSV(context.Background(), strings.NewReader("header\n"))
	if err == nil {
		t.Error("expected error for CSV with no data rows")
	}
}

func TestParseCSV_MissingName(t *testing.T) {
	svc := newTestService(&mockRepo{})

	csvData := `sku,active,price_net,price_gross,currency,tax_rule_id,stock,weight,locale,name,slug,description,meta_title,meta_description,category_ids,tag_ids,variant_sku,variant_active,variant_stock,variant_price_net,variant_price_gross,option_1_group,option_1_value,option_2_group,option_2_value,option_3_group,option_3_value
PROD-002,true,999,1199,EUR,,10,100,de-DE,,produkt-2,,,,,,,,,,,,,,,`

	_, err := svc.ParseCSV(context.Background(), strings.NewReader(csvData))
	if err == nil {
		t.Error("expected error when name is missing")
	}
}

func TestParseCSV_VariantBeforeProduct(t *testing.T) {
	svc := newTestService(&mockRepo{})

	csvData := `sku,active,price_net,price_gross,currency,tax_rule_id,stock,weight,locale,name,slug,description,meta_title,meta_description,category_ids,tag_ids,variant_sku,variant_active,variant_stock,variant_price_net,variant_price_gross,option_1_group,option_1_value,option_2_group,option_2_value,option_3_group,option_3_value
,,,,,,,,,,,,,,,,VARIANT-SKU,true,5,,,Größe,S,,,`

	_, err := svc.ParseCSV(context.Background(), strings.NewReader(csvData))
	if err == nil {
		t.Error("expected error when variant row appears before any product row")
	}
}

// ---------------------------------------------------------------------------
// CSVTemplate
// ---------------------------------------------------------------------------

func TestCSVTemplate_NotEmpty(t *testing.T) {
	data := CSVTemplate()
	if len(data) == 0 {
		t.Error("CSVTemplate returned empty bytes")
	}
	content := string(data)
	if !strings.Contains(content, "sku") {
		t.Error("CSV template missing 'sku' header")
	}
	if !strings.Contains(content, "variant_sku") {
		t.Error("CSV template missing 'variant_sku' header")
	}
}
