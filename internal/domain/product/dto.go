package product

import (
	"time"

	"github.com/google/uuid"
)

// --------------------------------------------------------------------------
// Request DTOs
// --------------------------------------------------------------------------

// TranslationInput carries locale-specific content in create/update requests.
type TranslationInput struct {
	Locale          string `json:"locale"           validate:"required,bcp47_language_tag|min=2,max=10"`
	Name            string `json:"name"             validate:"required,min=1,max=255"`
	Description     string `json:"description"`
	Slug            string `json:"slug"             validate:"required,min=1,max=255"`
	MetaTitle       string `json:"meta_title"       validate:"max=255"`
	MetaDescription string `json:"meta_description"`
}

// CreateProductRequest is the body accepted by the admin Create endpoint.
type CreateProductRequest struct {
	SKU          string                 `json:"sku"           validate:"max=100"`
	Active       bool                   `json:"active"`
	PriceNet     int                    `json:"price_net"     validate:"min=0"`
	PriceGross   int                    `json:"price_gross"   validate:"min=0"`
	Currency     string                 `json:"currency"      validate:"required,len=3"`
	TaxRuleID    *uuid.UUID             `json:"tax_rule_id"`
	Stock        int                    `json:"stock"         validate:"min=0"`
	Weight       int                    `json:"weight"        validate:"min=0"`
	CustomFields map[string]interface{} `json:"custom_fields"`
	Metadata     map[string]interface{} `json:"metadata"`
	Translations []TranslationInput     `json:"translations"  validate:"required,min=1,dive"`
	CategoryIDs  []uuid.UUID            `json:"category_ids"`
	TagIDs       []uuid.UUID            `json:"tag_ids"`
}

// UpdateProductRequest is the body accepted by the admin Update endpoint.
// All fields are optional; only provided fields should be applied.
type UpdateProductRequest struct {
	SKU          *string                `json:"sku"           validate:"omitempty,max=100"`
	Active       *bool                  `json:"active"`
	PriceNet     *int                   `json:"price_net"     validate:"omitempty,min=0"`
	PriceGross   *int                   `json:"price_gross"   validate:"omitempty,min=0"`
	Currency     *string                `json:"currency"      validate:"omitempty,len=3"`
	TaxRuleID    *uuid.UUID             `json:"tax_rule_id"`
	Stock        *int                   `json:"stock"         validate:"omitempty,min=0"`
	Weight       *int                   `json:"weight"        validate:"omitempty,min=0"`
	CustomFields map[string]interface{} `json:"custom_fields"`
	Metadata     map[string]interface{} `json:"metadata"`
	Translations []TranslationInput     `json:"translations"  validate:"omitempty,dive"`
	CategoryIDs  []uuid.UUID            `json:"category_ids"`
	TagIDs       []uuid.UUID            `json:"tag_ids"`
	MediaIDs     []uuid.UUID            `json:"media_ids"`
}

// GenerateVariantsRequest carries the grouped option IDs for variant generation.
type GenerateVariantsRequest struct {
	// OptionGroups is a slice-of-slices.  Each inner slice represents one property
	// axis (e.g. [[sizeS_id, sizeM_id], [colorRed_id, colorBlue_id]]).
	OptionGroups [][]uuid.UUID `json:"option_groups" validate:"required,min=1"`
}

// CreateVariantRequest is the body for creating a single product variant.
type CreateVariantRequest struct {
	SKU        string      `json:"sku"`
	PriceGross *int        `json:"price_gross"`
	PriceNet   *int        `json:"price_net"`
	Stock      int         `json:"stock"   validate:"min=0"`
	Active     bool        `json:"active"`
	OptionIDs  []uuid.UUID `json:"option_ids"`
}

// UpdateVariantRequest is the body for updating a single product variant.
type UpdateVariantRequest = CreateVariantRequest

// PropertyGroupTranslationInput carries a single locale translation for a property group.
type PropertyGroupTranslationInput struct {
	Locale string `json:"locale" validate:"required"`
	Name   string `json:"name"   validate:"required,min=1,max=255"`
}

// CreatePropertyGroupRequest is the body for creating a property group.
type CreatePropertyGroupRequest struct {
	Position     int                              `json:"position"`
	Translations []PropertyGroupTranslationInput `json:"translations" validate:"required,min=1,dive"`
}

// UpdatePropertyGroupRequest is the body for updating a property group.
type UpdatePropertyGroupRequest = CreatePropertyGroupRequest

// PropertyOptionTranslationInput carries a single locale translation for a property option.
type PropertyOptionTranslationInput struct {
	Locale string `json:"locale" validate:"required"`
	Name   string `json:"name"   validate:"required,min=1,max=255"`
}

// CreatePropertyOptionRequest is the body for creating a property option.
type CreatePropertyOptionRequest struct {
	Position     int                               `json:"position"`
	ColorHex     string                            `json:"color_hex"`
	Translations []PropertyOptionTranslationInput `json:"translations" validate:"required,min=1,dive"`
}

// UpdatePropertyOptionRequest is the body for updating a property option.
type UpdatePropertyOptionRequest = CreatePropertyOptionRequest

// --------------------------------------------------------------------------
// Bulk / Import DTOs
// --------------------------------------------------------------------------

// BulkImportOptionInput describes a property option by name for CSV/bulk import.
// The service resolves group/option names to IDs via find-or-create.
type BulkImportOptionInput struct {
	GroupName  string `json:"group_name"`
	OptionName string `json:"option_name"`
	Locale     string `json:"locale"`
}

// BulkImportVariantInput describes a variant within a bulk import request.
type BulkImportVariantInput struct {
	SKU        string                  `json:"sku"`
	Active     bool                    `json:"active"`
	Stock      int                     `json:"stock"       validate:"min=0"`
	PriceNet   *int                    `json:"price_net"`
	PriceGross *int                    `json:"price_gross"`
	Options    []BulkImportOptionInput `json:"options"`
}

// BulkCreateProductRequest extends CreateProductRequest with inline variants.
type BulkCreateProductRequest struct {
	CreateProductRequest
	Variants []BulkImportVariantInput `json:"variants"`
}

// BulkRequest is the body for the JSON bulk-create endpoint.
// Max 250 products per request.
type BulkRequest struct {
	Products []BulkCreateProductRequest `json:"products" validate:"required,min=1,max=250"`
}

// BulkResult holds the outcome for a single product within a bulk operation.
type BulkResult struct {
	Index   int      `json:"index"`
	SKU     string   `json:"sku,omitempty"`
	Success bool     `json:"success"`
	ID      string   `json:"id,omitempty"`
	Errors  []string `json:"errors,omitempty"`
}

// BulkResponse is returned by both the JSON bulk and CSV import endpoints.
type BulkResponse struct {
	Results   []BulkResult `json:"results"`
	Total     int          `json:"total"`
	Succeeded int          `json:"succeeded"`
	Failed    int          `json:"failed"`
}

// --------------------------------------------------------------------------
// Response DTOs
// --------------------------------------------------------------------------

// ProductTranslationResponse is the per-locale projection in API responses.
type ProductTranslationResponse struct {
	Locale          string `json:"locale"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Slug            string `json:"slug"`
	MetaTitle       string `json:"meta_title"`
	MetaDescription string `json:"meta_description"`
}

// ProductMediaResponse represents a media attachment in API responses.
type ProductMediaResponse struct {
	MediaID  uuid.UUID `json:"media_id"`
	Position int       `json:"position"`
	URL      string    `json:"url,omitempty"`
}

// PropertyOptionTranslationResponse is the per-locale translation in API responses.
type PropertyOptionTranslationResponse struct {
	Locale string `json:"locale"`
	Name   string `json:"name"`
}

// PropertyOptionResponse is the variant option projection in API responses.
type PropertyOptionResponse struct {
	ID           uuid.UUID                            `json:"id"`
	GroupID      uuid.UUID                            `json:"group_id"`
	ColorHex     string                               `json:"color_hex,omitempty"`
	Position     int                                  `json:"position"`
	Translations []PropertyOptionTranslationResponse `json:"translations,omitempty"`
}

// PropertyGroupTranslationResponse is the per-locale translation in API responses.
type PropertyGroupTranslationResponse struct {
	Locale string `json:"locale"`
	Name   string `json:"name"`
}

// PropertyGroupResponse is the full property group projection in API responses.
type PropertyGroupResponse struct {
	ID           uuid.UUID                           `json:"id"`
	Position     int                                 `json:"position"`
	CreatedAt    time.Time                           `json:"created_at"`
	UpdatedAt    time.Time                           `json:"updated_at"`
	Translations []PropertyGroupTranslationResponse `json:"translations,omitempty"`
	Options      []PropertyOptionResponse            `json:"options,omitempty"`
}

// ProductVariantResponse is the variant projection in API responses.
type ProductVariantResponse struct {
	ID           uuid.UUID              `json:"id"`
	ProductID    uuid.UUID              `json:"product_id"`
	SKU          string                 `json:"sku"`
	PriceNet     *int                   `json:"price_net,omitempty"`
	PriceGross   *int                   `json:"price_gross,omitempty"`
	Stock        int                    `json:"stock"`
	Active       bool                   `json:"active"`
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
	Options      []PropertyOptionResponse `json:"options,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// ProductResponse is the full product projection returned by the API.
type ProductResponse struct {
	ID           uuid.UUID                    `json:"id"`
	SKU          string                       `json:"sku"`
	Active       bool                         `json:"active"`
	PriceNet     int                          `json:"price_net"`
	PriceGross   int                          `json:"price_gross"`
	Currency     string                       `json:"currency"`
	TaxRuleID    *uuid.UUID                   `json:"tax_rule_id,omitempty"`
	Stock        int                          `json:"stock"`
	Weight       int                          `json:"weight"`
	HasVariants  bool                         `json:"has_variants"`
	CustomFields map[string]interface{}       `json:"custom_fields,omitempty"`
	Metadata     map[string]interface{}       `json:"metadata,omitempty"`
	CreatedAt    time.Time                    `json:"created_at"`
	UpdatedAt    time.Time                    `json:"updated_at"`
	Translations []ProductTranslationResponse `json:"translations,omitempty"`
	Categories   []uuid.UUID                  `json:"categories,omitempty"`
	Tags         []uuid.UUID                  `json:"tags,omitempty"`
	Media        []ProductMediaResponse       `json:"media,omitempty"`
	Variants     []ProductVariantResponse     `json:"variants,omitempty"`
}

// ProductListResponse wraps a page of ProductResponse values.
type ProductListResponse struct {
	Items []ProductResponse `json:"items"`
}

// --------------------------------------------------------------------------
// Mapping helpers – entity → response DTO
// --------------------------------------------------------------------------

// ToResponse converts a domain Product to its API response representation.
func ToResponse(p *Product) ProductResponse {
	resp := ProductResponse{
		ID:           p.ID,
		SKU:          p.SKU,
		Active:       p.Active,
		PriceNet:     p.PriceNet,
		PriceGross:   p.PriceGross,
		Currency:     p.Currency,
		TaxRuleID:    p.TaxRuleID,
		Stock:        p.Stock,
		Weight:       p.Weight,
		HasVariants:  p.HasVariants || len(p.Variants) > 0,
		CustomFields: p.CustomFields,
		Metadata:     p.Metadata,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
		Categories:   p.Categories,
		Tags:         p.Tags,
	}

	for _, t := range p.Translations {
		resp.Translations = append(resp.Translations, ProductTranslationResponse{
			Locale:          t.Locale,
			Name:            t.Name,
			Description:     t.Description,
			Slug:            t.Slug,
			MetaTitle:       t.MetaTitle,
			MetaDescription: t.MetaDescription,
		})
	}

	for _, m := range p.Media {
		resp.Media = append(resp.Media, ProductMediaResponse{
			MediaID:  m.MediaID,
			Position: m.Position,
			URL:      m.URL,
		})
	}

	for _, v := range p.Variants {
		// Inherit price from parent product when variant has no own price.
		// A nil pointer (NULL in DB) or a zero value both mean "no own price".
		priceNet := v.PriceNet
		priceGross := v.PriceGross
		if priceNet == nil || *priceNet == 0 {
			priceNet = &p.PriceNet
		}
		if priceGross == nil || *priceGross == 0 {
			priceGross = &p.PriceGross
		}

		vr := ProductVariantResponse{
			ID:           v.ID,
			ProductID:    v.ProductID,
			SKU:          v.SKU,
			PriceNet:     priceNet,
			PriceGross:   priceGross,
			Stock:        v.Stock,
			Active:       v.Active,
			CustomFields: v.CustomFields,
			CreatedAt:    v.CreatedAt,
			UpdatedAt:    v.UpdatedAt,
		}
		for _, o := range v.Options {
			vr.Options = append(vr.Options, propertyOptionToResponse(o))
		}
		resp.Variants = append(resp.Variants, vr)
	}

	return resp
}

// ToResponseList converts a slice of domain Products to a list DTO.
func ToResponseList(products []Product) ProductListResponse {
	items := make([]ProductResponse, len(products))
	for i := range products {
		items[i] = ToResponse(&products[i])
	}
	return ProductListResponse{Items: items}
}

// --------------------------------------------------------------------------
// Mapping helpers – request DTO → entity
// --------------------------------------------------------------------------

// FromCreateRequest builds a new Product entity from a CreateProductRequest.
func FromCreateRequest(req *CreateProductRequest) *Product {
	p := &Product{
		SKU:          req.SKU,
		Active:       req.Active,
		PriceNet:     req.PriceNet,
		PriceGross:   req.PriceGross,
		Currency:     req.Currency,
		TaxRuleID:    req.TaxRuleID,
		Stock:        req.Stock,
		Weight:       req.Weight,
		CustomFields: req.CustomFields,
		Metadata:     req.Metadata,
		Categories:   req.CategoryIDs,
		Tags:         req.TagIDs,
	}

	for _, t := range req.Translations {
		p.Translations = append(p.Translations, ProductTranslation{
			Locale:          t.Locale,
			Name:            t.Name,
			Description:     t.Description,
			Slug:            t.Slug,
			MetaTitle:       t.MetaTitle,
			MetaDescription: t.MetaDescription,
		})
	}

	return p
}

// propertyOptionToResponse maps a PropertyOption entity to its response DTO.
func propertyOptionToResponse(o PropertyOption) PropertyOptionResponse {
	resp := PropertyOptionResponse{
		ID:       o.ID,
		GroupID:  o.GroupID,
		ColorHex: o.ColorHex,
		Position: o.Position,
	}
	for _, t := range o.Translations {
		resp.Translations = append(resp.Translations, PropertyOptionTranslationResponse{
			Locale: t.Locale,
			Name:   t.Name,
		})
	}
	return resp
}

// PropertyGroupToResponse maps a PropertyGroup entity to its response DTO.
func PropertyGroupToResponse(g PropertyGroup) PropertyGroupResponse {
	resp := PropertyGroupResponse{
		ID:        g.ID,
		Position:  g.Position,
		CreatedAt: g.CreatedAt,
		UpdatedAt: g.UpdatedAt,
	}
	for _, t := range g.Translations {
		resp.Translations = append(resp.Translations, PropertyGroupTranslationResponse{
			Locale: t.Locale,
			Name:   t.Name,
		})
	}
	for _, o := range g.Options {
		resp.Options = append(resp.Options, propertyOptionToResponse(o))
	}
	return resp
}

// ApplyUpdateRequest applies the non-nil fields of an UpdateProductRequest to an existing Product.
func ApplyUpdateRequest(p *Product, req *UpdateProductRequest) {
	if req.SKU != nil {
		p.SKU = *req.SKU
	}
	if req.Active != nil {
		p.Active = *req.Active
	}
	if req.PriceNet != nil {
		p.PriceNet = *req.PriceNet
	}
	if req.PriceGross != nil {
		p.PriceGross = *req.PriceGross
	}
	if req.Currency != nil {
		p.Currency = *req.Currency
	}
	if req.TaxRuleID != nil {
		p.TaxRuleID = req.TaxRuleID
	}
	if req.Stock != nil {
		p.Stock = *req.Stock
	}
	if req.Weight != nil {
		p.Weight = *req.Weight
	}
	if req.CustomFields != nil {
		p.CustomFields = req.CustomFields
	}
	if req.Metadata != nil {
		p.Metadata = req.Metadata
	}
	if req.CategoryIDs != nil {
		p.Categories = req.CategoryIDs
	}
	if req.TagIDs != nil {
		p.Tags = req.TagIDs
	}
	if req.MediaIDs != nil {
		p.Media = make([]ProductMedia, len(req.MediaIDs))
		for i, id := range req.MediaIDs {
			p.Media[i] = ProductMedia{MediaID: id, Position: i}
		}
	}
	if len(req.Translations) > 0 {
		p.Translations = nil
		for _, t := range req.Translations {
			p.Translations = append(p.Translations, ProductTranslation{
				ProductID:       p.ID,
				Locale:          t.Locale,
				Name:            t.Name,
				Description:     t.Description,
				Slug:            t.Slug,
				MetaTitle:       t.MetaTitle,
				MetaDescription: t.MetaDescription,
			})
		}
	}
}
