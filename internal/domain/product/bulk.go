package product

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// maxBulkProducts is the maximum number of products allowed per bulk request.
const maxBulkProducts = 250

// --------------------------------------------------------------------------
// BulkCreate
// --------------------------------------------------------------------------

// BulkCreate creates up to maxBulkProducts products in a single call.
// Each product is processed independently; failures are collected and returned
// as partial results (partial-success semantics).
func (s *Service) BulkCreate(ctx context.Context, reqs []BulkCreateProductRequest) BulkResponse {
	resp := BulkResponse{
		Total:   len(reqs),
		Results: make([]BulkResult, 0, len(reqs)),
	}

	for i, req := range reqs {
		result := BulkResult{Index: i, SKU: req.SKU}

		// Upsert: find by SKU → update if exists, create if not.
		existing, findErr := s.repo.FindBySKU(ctx, req.SKU)
		if findErr != nil && !errors.Is(findErr, ErrNotFound) {
			result.Errors = []string{findErr.Error()}
			resp.Failed++
			resp.Results = append(resp.Results, result)
			continue
		}

		var p *Product
		if existing != nil {
			// Update existing product with imported data.
			p = existing
			applyCreateRequest(p, &req.CreateProductRequest)
			if err := s.repo.Update(ctx, p); err != nil {
				result.Errors = []string{err.Error()}
				resp.Failed++
				resp.Results = append(resp.Results, result)
				continue
			}
		} else {
			// Create new product.
			p = FromCreateRequest(&req.CreateProductRequest)
			p.ID = uuid.New()
			if err := s.Create(ctx, p); err != nil {
				result.Errors = []string{err.Error()}
				resp.Failed++
				resp.Results = append(resp.Results, result)
				continue
			}
		}

		// Resolve property options and create variants.
		for vi, varReq := range req.Variants {
			optionIDs, err := s.resolveOptionIDs(ctx, varReq.Options)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("variant %d: resolve options: %s", vi, err.Error()))
				continue
			}

			cvReq := CreateVariantRequest{
				SKU:        varReq.SKU,
				Active:     varReq.Active,
				Stock:      varReq.Stock,
				PriceNet:   varReq.PriceNet,
				PriceGross: varReq.PriceGross,
				OptionIDs:  optionIDs,
			}
			if _, err := s.CreateVariant(ctx, p.ID, cvReq); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("variant %d (%s): %s", vi, varReq.SKU, err.Error()))
			}
		}

		result.Success = len(result.Errors) == 0
		result.ID = p.ID.String()
		if result.Success {
			resp.Succeeded++
		} else {
			// Product was created but some variants failed – still counts as partial success.
			// We mark success=false only if the product itself failed (handled above via continue).
			result.Success = true
			resp.Succeeded++
		}
		resp.Results = append(resp.Results, result)
	}

	return resp
}

// resolveOptionIDs translates BulkImportOptionInput (names) into property option UUIDs.
func (s *Service) resolveOptionIDs(ctx context.Context, options []BulkImportOptionInput) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(options))
	for _, opt := range options {
		if opt.GroupName == "" || opt.OptionName == "" {
			continue
		}
		locale := opt.Locale
		if locale == "" {
			locale = "de-DE"
		}

		group, err := s.repo.FindOrCreatePropertyGroup(ctx, locale, opt.GroupName)
		if err != nil {
			return nil, fmt.Errorf("group %q: %w", opt.GroupName, err)
		}

		option, err := s.repo.FindOrCreatePropertyOption(ctx, group.ID, locale, opt.OptionName)
		if err != nil {
			return nil, fmt.Errorf("option %q in group %q: %w", opt.OptionName, opt.GroupName, err)
		}

		ids = append(ids, option.ID)
	}
	return ids, nil
}

// --------------------------------------------------------------------------
// CSV Template
// --------------------------------------------------------------------------

// CSVTemplate returns the bytes of a ready-to-fill CSV template with one
// example product row and one continuation variant row.
func CSVTemplate() []byte {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	header := csvColumns()
	_ = w.Write(header)

	// Example product row with one variant.
	_ = w.Write([]string{
		"SHIRT-001", "true", "1999", "2499", "EUR", "", "100", "300",
		"de-DE", "Beispiel T-Shirt", "beispiel-t-shirt", "Produktbeschreibung", "", "",
		"", "",
		"SHIRT-001-S-ROT", "true", "20", "", "",
		"Größe", "S", "Farbe", "Rot", "", "",
	})

	// Continuation variant row (sku empty → same product).
	_ = w.Write([]string{
		"", "", "", "", "", "", "", "",
		"", "", "", "", "", "",
		"", "",
		"SHIRT-001-M-ROT", "true", "30", "", "",
		"Größe", "M", "Farbe", "Rot", "", "",
	})

	w.Flush()
	return buf.Bytes()
}

// csvColumns returns the ordered CSV column names.
func csvColumns() []string {
	return []string{
		"sku", "active", "price_net", "price_gross", "currency", "tax_rule_id",
		"stock", "weight",
		"locale", "name", "slug", "description", "meta_title", "meta_description",
		"category_ids", "tag_ids",
		"variant_sku", "variant_active", "variant_stock", "variant_price_net", "variant_price_gross",
		"option_1_group", "option_1_value", "option_2_group", "option_2_value", "option_3_group", "option_3_value",
	}
}

// --------------------------------------------------------------------------
// ParseCSV
// --------------------------------------------------------------------------

// ParseCSV reads a CSV stream and creates products.
// Rows with a non-empty "sku" column start a new product.
// Rows with an empty "sku" column add a variant to the most recent product.
func (s *Service) ParseCSV(ctx context.Context, r io.Reader) (BulkResponse, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1 // allow variable column counts

	rows, err := reader.ReadAll()
	if err != nil {
		return BulkResponse{}, fmt.Errorf("CSV read: %w", err)
	}
	if len(rows) < 2 {
		return BulkResponse{}, fmt.Errorf("CSV must contain a header row and at least one data row")
	}

	// Build column index map from header.
	colIdx := make(map[string]int, len(rows[0]))
	for i, col := range rows[0] {
		colIdx[strings.ToLower(strings.TrimSpace(col))] = i
	}

	col := func(row []string, name string) string {
		i, ok := colIdx[name]
		if !ok || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}

	// Parse data rows into BulkCreateProductRequest list.
	var reqs []BulkCreateProductRequest
	var current *BulkCreateProductRequest

	for rowNum, row := range rows[1:] {
		if len(row) == 0 {
			continue
		}

		sku := col(row, "sku")

		if sku != "" {
			// New product row.
			if len(reqs) >= maxBulkProducts {
				return BulkResponse{}, fmt.Errorf("row %d: exceeds maximum of %d products per import", rowNum+2, maxBulkProducts)
			}

			req, err := parseProductRow(row, col)
			if err != nil {
				return BulkResponse{}, fmt.Errorf("row %d: %w", rowNum+2, err)
			}

			// Parse optional inline variant from same product row.
			if v := parseVariantFromRow(row, col); v != nil {
				req.Variants = append(req.Variants, *v)
			}

			reqs = append(reqs, req)
			current = &reqs[len(reqs)-1]
		} else {
			// Continuation row – adds a variant to the current product.
			if current == nil {
				return BulkResponse{}, fmt.Errorf("row %d: variant row found before any product row", rowNum+2)
			}
			if v := parseVariantFromRow(row, col); v != nil {
				current.Variants = append(current.Variants, *v)
			}
		}
	}

	if len(reqs) == 0 {
		return BulkResponse{}, fmt.Errorf("CSV contains no product rows")
	}

	return s.BulkCreate(ctx, reqs), nil
}

// parseProductRow maps CSV columns onto a BulkCreateProductRequest.
func parseProductRow(row []string, col func([]string, string) string) (BulkCreateProductRequest, error) {
	req := BulkCreateProductRequest{}

	req.SKU = col(row, "sku")
	req.Active = col(row, "active") == "true"
	req.Currency = col(row, "currency")
	if req.Currency == "" {
		req.Currency = "EUR"
	}

	if v := col(row, "price_net"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return req, fmt.Errorf("price_net must be an integer: %w", err)
		}
		req.PriceNet = n
	}
	if v := col(row, "price_gross"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return req, fmt.Errorf("price_gross must be an integer: %w", err)
		}
		req.PriceGross = n
	}
	if v := col(row, "stock"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return req, fmt.Errorf("stock must be an integer: %w", err)
		}
		req.Stock = n
	}
	if v := col(row, "weight"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return req, fmt.Errorf("weight must be an integer: %w", err)
		}
		req.Weight = n
	}
	if v := col(row, "tax_rule_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return req, fmt.Errorf("tax_rule_id must be a valid UUID: %w", err)
		}
		req.TaxRuleID = &id
	}

	// Translation.
	locale := col(row, "locale")
	if locale == "" {
		locale = "de-DE"
	}
	name := col(row, "name")
	slug := col(row, "slug")
	if name == "" {
		return req, fmt.Errorf("name is required")
	}
	if slug == "" {
		return req, fmt.Errorf("slug is required")
	}
	req.Translations = []TranslationInput{
		{
			Locale:          locale,
			Name:            name,
			Slug:            slug,
			Description:     col(row, "description"),
			MetaTitle:       col(row, "meta_title"),
			MetaDescription: col(row, "meta_description"),
		},
	}

	// Categories (semicolon-separated UUIDs).
	if v := col(row, "category_ids"); v != "" {
		for _, raw := range strings.Split(v, ";") {
			raw = strings.TrimSpace(raw)
			if raw == "" {
				continue
			}
			id, err := uuid.Parse(raw)
			if err != nil {
				return req, fmt.Errorf("category_ids: invalid UUID %q: %w", raw, err)
			}
			req.CategoryIDs = append(req.CategoryIDs, id)
		}
	}

	// Tags (semicolon-separated UUIDs).
	if v := col(row, "tag_ids"); v != "" {
		for _, raw := range strings.Split(v, ";") {
			raw = strings.TrimSpace(raw)
			if raw == "" {
				continue
			}
			id, err := uuid.Parse(raw)
			if err != nil {
				return req, fmt.Errorf("tag_ids: invalid UUID %q: %w", raw, err)
			}
			req.TagIDs = append(req.TagIDs, id)
		}
	}

	return req, nil
}

// parseVariantFromRow reads variant columns from a CSV row.
// Returns nil when no variant_sku is present.
func parseVariantFromRow(row []string, col func([]string, string) string) *BulkImportVariantInput {
	varSKU := col(row, "variant_sku")
	if varSKU == "" {
		return nil
	}

	v := &BulkImportVariantInput{
		SKU:    varSKU,
		Active: col(row, "variant_active") != "false",
	}

	if s := col(row, "variant_stock"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			v.Stock = n
		}
	}
	if s := col(row, "variant_price_net"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			v.PriceNet = &n
		}
	}
	if s := col(row, "variant_price_gross"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			v.PriceGross = &n
		}
	}

	// Locale for option resolution: use product row's locale or default.
	locale := col(row, "locale")
	if locale == "" {
		locale = "de-DE"
	}

	// Up to 3 option pairs (option_N_group / option_N_value).
	for n := 1; n <= 3; n++ {
		group := col(row, fmt.Sprintf("option_%d_group", n))
		value := col(row, fmt.Sprintf("option_%d_value", n))
		if group != "" && value != "" {
			v.Options = append(v.Options, BulkImportOptionInput{
				GroupName:  group,
				OptionName: value,
				Locale:     locale,
			})
		}
	}

	return v
}

// applyCreateRequest overwrites the mutable fields of an existing Product
// with the values from a CreateProductRequest (used for upsert during import).
func applyCreateRequest(p *Product, req *CreateProductRequest) {
	p.SKU = req.SKU
	p.Active = req.Active
	p.PriceNet = req.PriceNet
	p.PriceGross = req.PriceGross
	p.Currency = req.Currency
	p.TaxRuleID = req.TaxRuleID
	p.Stock = req.Stock
	p.Weight = req.Weight
	p.Categories = req.CategoryIDs
	p.Tags = req.TagIDs

	p.Translations = p.Translations[:0]
	for _, t := range req.Translations {
		p.Translations = append(p.Translations, ProductTranslation{
			ProductID:       p.ID,
			Locale:          t.Locale,
			Name:            t.Name,
			Slug:            t.Slug,
			Description:     t.Description,
			MetaTitle:       t.MetaTitle,
			MetaDescription: t.MetaDescription,
		})
	}
}
