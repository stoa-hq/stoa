package product

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ---------------------------------------------------------------------------
// API envelope types (local to handler)
// ---------------------------------------------------------------------------

type apiResponse struct {
	Data   interface{} `json:"data,omitempty"`
	Meta   *apiMeta    `json:"meta,omitempty"`
	Errors []apiError  `json:"errors,omitempty"`
}

type apiMeta struct {
	Total int `json:"total"`
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Pages int `json:"pages"`
}

type apiError struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
	Field  string `json:"field,omitempty"`
}

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

// Handler handles HTTP requests for the product domain.
type Handler struct {
	service   *Service
	validator *validator.Validate
	logger    zerolog.Logger
}

// NewHandler creates a new product Handler.
func NewHandler(service *Service, validate *validator.Validate, logger zerolog.Logger) *Handler {
	return &Handler{
		service:   service,
		validator: validate,
		logger:    logger,
	}
}

// ---------------------------------------------------------------------------
// Route registration
// ---------------------------------------------------------------------------

// RegisterAdminRoutes mounts the full CRUD surface under the given router.
// Expected prefix: /products
func (h *Handler) RegisterAdminRoutes(r chi.Router) {
	r.Get("/products", h.adminList)
	r.Post("/products", h.adminCreate)
	r.Get("/products/{id}", h.adminGetByID)
	r.Put("/products/{id}", h.adminUpdate)
	r.Delete("/products/{id}", h.adminDelete)
	// POST /products/{id}/variants: if option_groups present → generate all; otherwise → create single
	r.Post("/products/{id}/variants", h.adminCreateOrGenerateVariants)
	r.Put("/products/{id}/variants/{variantId}", h.adminUpdateVariant)
	r.Delete("/products/{id}/variants/{variantId}", h.adminDeleteVariant)

	// Property Groups
	r.Get("/property-groups", h.adminListPropertyGroups)
	r.Post("/property-groups", h.adminCreatePropertyGroup)
	r.Get("/property-groups/{id}", h.adminGetPropertyGroup)
	r.Put("/property-groups/{id}", h.adminUpdatePropertyGroup)
	r.Delete("/property-groups/{id}", h.adminDeletePropertyGroup)

	// Property Options (under group)
	r.Post("/property-groups/{id}/options", h.adminCreatePropertyOption)
	r.Put("/property-groups/{id}/options/{optId}", h.adminUpdatePropertyOption)
	r.Delete("/property-groups/{id}/options/{optId}", h.adminDeletePropertyOption)
}

// RegisterStoreRoutes mounts the public/customer-facing read endpoints.
// Expected prefix: /products
func (h *Handler) RegisterStoreRoutes(r chi.Router) {
	r.Get("/products", h.storeList)
	r.Get("/products/id/{id}", h.storeGetByID)
	r.Get("/products/{slug}", h.storeGetBySlug)
}

// ---------------------------------------------------------------------------
// Admin handlers
// ---------------------------------------------------------------------------

// adminList handles GET /products
// Query params: page, limit, sort, order, search, category_id, active
func (h *Handler) adminList(w http.ResponseWriter, r *http.Request) {
	filter, page, limit := h.parseListFilter(r)

	products, total, err := h.service.List(r.Context(), filter)
	if err != nil {
		h.serverError(w, r, err)
		return
	}

	pages := 0
	if limit > 0 {
		pages = int(math.Ceil(float64(total) / float64(limit)))
	}

	h.writeJSON(w, http.StatusOK, apiResponse{
		Data: ToResponseList(products),
		Meta: &apiMeta{
			Total: total,
			Page:  page,
			Limit: limit,
			Pages: pages,
		},
	})
}

// adminGetByID handles GET /products/{id}
func (h *Handler) adminGetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUID(w, r, "id")
	if !ok {
		return
	}

	p, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			h.notFound(w, "product not found")
			return
		}
		h.serverError(w, r, err)
		return
	}

	resp := ToResponse(p)
	h.writeJSON(w, http.StatusOK, apiResponse{Data: resp})
}

// adminCreate handles POST /products
func (h *Handler) adminCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateProductRequest
	if !h.decodeJSON(w, r, &req) {
		return
	}
	if !h.validate(w, &req) {
		return
	}

	p := FromCreateRequest(&req)
	p.ID = uuid.New()

	if err := h.service.Create(r.Context(), p); err != nil {
		h.serverError(w, r, err)
		return
	}

	resp := ToResponse(p)
	h.writeJSON(w, http.StatusCreated, apiResponse{Data: resp})
}

// adminUpdate handles PUT /products/{id}
func (h *Handler) adminUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUID(w, r, "id")
	if !ok {
		return
	}

	var req UpdateProductRequest
	if !h.decodeJSON(w, r, &req) {
		return
	}
	if !h.validate(w, &req) {
		return
	}

	existing, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			h.notFound(w, "product not found")
			return
		}
		h.serverError(w, r, err)
		return
	}

	ApplyUpdateRequest(existing, &req)

	if err := h.service.Update(r.Context(), existing); err != nil {
		if errors.Is(err, ErrNotFound) {
			h.notFound(w, "product not found")
			return
		}
		h.serverError(w, r, err)
		return
	}

	resp := ToResponse(existing)
	h.writeJSON(w, http.StatusOK, apiResponse{Data: resp})
}

// adminDelete handles DELETE /products/{id}
func (h *Handler) adminDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUID(w, r, "id")
	if !ok {
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			h.notFound(w, "product not found")
			return
		}
		h.serverError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// adminCreateOrGenerateVariants handles POST /products/{id}/variants.
// If the request body contains "option_groups" it generates the cartesian product;
// otherwise it creates a single variant.
func (h *Handler) adminCreateOrGenerateVariants(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUID(w, r, "id")
	if !ok {
		return
	}

	// Peek at the raw body to distinguish the two request shapes.
	var raw map[string]json.RawMessage
	if !h.decodeJSON(w, r, &raw) {
		return
	}

	if _, hasGroups := raw["option_groups"]; hasGroups {
		// GenerateVariants path.
		var req GenerateVariantsRequest
		if err := json.Unmarshal(mustMarshal(raw), &req); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid_body", "invalid request body: "+err.Error(), "")
			return
		}
		if !h.validate(w, &req) {
			return
		}
		variants, err := h.service.GenerateVariants(r.Context(), id, req.OptionGroups)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				h.notFound(w, "product not found")
				return
			}
			h.serverError(w, r, err)
			return
		}
		h.writeJSON(w, http.StatusCreated, apiResponse{Data: variants})
		return
	}

	// CreateVariant (single) path.
	var req CreateVariantRequest
	if err := json.Unmarshal(mustMarshal(raw), &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_body", "invalid request body: "+err.Error(), "")
		return
	}
	if !h.validate(w, &req) {
		return
	}
	v, err := h.service.CreateVariant(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			h.notFound(w, "product not found")
			return
		}
		h.serverError(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, apiResponse{Data: v})
}

// adminUpdateVariant handles PUT /products/{id}/variants/{variantId}
func (h *Handler) adminUpdateVariant(w http.ResponseWriter, r *http.Request) {
	_, ok := h.parseUUID(w, r, "id")
	if !ok {
		return
	}
	variantID, ok := h.parseUUID(w, r, "variantId")
	if !ok {
		return
	}

	var req UpdateVariantRequest
	if !h.decodeJSON(w, r, &req) {
		return
	}
	if !h.validate(w, &req) {
		return
	}

	v, err := h.service.UpdateVariant(r.Context(), variantID, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			h.notFound(w, "variant not found")
			return
		}
		h.serverError(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, apiResponse{Data: v})
}

// adminDeleteVariant handles DELETE /products/{id}/variants/{variantId}
func (h *Handler) adminDeleteVariant(w http.ResponseWriter, r *http.Request) {
	_, ok := h.parseUUID(w, r, "id")
	if !ok {
		return
	}
	variantID, ok := h.parseUUID(w, r, "variantId")
	if !ok {
		return
	}

	if err := h.service.DeleteVariant(r.Context(), variantID); err != nil {
		if errors.Is(err, ErrNotFound) {
			h.notFound(w, "variant not found")
			return
		}
		h.serverError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// adminListPropertyGroups handles GET /property-groups
func (h *Handler) adminListPropertyGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := h.service.ListPropertyGroups(r.Context())
	if err != nil {
		h.serverError(w, r, err)
		return
	}
	resp := make([]PropertyGroupResponse, len(groups))
	for i, g := range groups {
		resp[i] = PropertyGroupToResponse(g)
	}
	h.writeJSON(w, http.StatusOK, apiResponse{Data: resp})
}

// adminGetPropertyGroup handles GET /property-groups/{id}
func (h *Handler) adminGetPropertyGroup(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUID(w, r, "id")
	if !ok {
		return
	}
	g, err := h.service.GetPropertyGroupByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			h.notFound(w, "property group not found")
			return
		}
		h.serverError(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, apiResponse{Data: PropertyGroupToResponse(*g)})
}

// adminCreatePropertyGroup handles POST /property-groups
func (h *Handler) adminCreatePropertyGroup(w http.ResponseWriter, r *http.Request) {
	var req CreatePropertyGroupRequest
	if !h.decodeJSON(w, r, &req) {
		return
	}
	if !h.validate(w, &req) {
		return
	}

	g := &PropertyGroup{Position: req.Position}
	for _, t := range req.Translations {
		g.Translations = append(g.Translations, PropertyGroupTranslation{
			Locale: t.Locale,
			Name:   t.Name,
		})
	}

	if err := h.service.CreatePropertyGroup(r.Context(), g); err != nil {
		h.serverError(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, apiResponse{Data: PropertyGroupToResponse(*g)})
}

// adminUpdatePropertyGroup handles PUT /property-groups/{id}
func (h *Handler) adminUpdatePropertyGroup(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUID(w, r, "id")
	if !ok {
		return
	}

	var req UpdatePropertyGroupRequest
	if !h.decodeJSON(w, r, &req) {
		return
	}
	if !h.validate(w, &req) {
		return
	}

	g := &PropertyGroup{ID: id, Position: req.Position}
	for _, t := range req.Translations {
		g.Translations = append(g.Translations, PropertyGroupTranslation{
			GroupID: id,
			Locale:  t.Locale,
			Name:    t.Name,
		})
	}

	if err := h.service.UpdatePropertyGroup(r.Context(), g); err != nil {
		if errors.Is(err, ErrNotFound) {
			h.notFound(w, "property group not found")
			return
		}
		h.serverError(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, apiResponse{Data: PropertyGroupToResponse(*g)})
}

// adminDeletePropertyGroup handles DELETE /property-groups/{id}
func (h *Handler) adminDeletePropertyGroup(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUID(w, r, "id")
	if !ok {
		return
	}
	if err := h.service.DeletePropertyGroup(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			h.notFound(w, "property group not found")
			return
		}
		h.serverError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// adminCreatePropertyOption handles POST /property-groups/{id}/options
func (h *Handler) adminCreatePropertyOption(w http.ResponseWriter, r *http.Request) {
	groupID, ok := h.parseUUID(w, r, "id")
	if !ok {
		return
	}

	var req CreatePropertyOptionRequest
	if !h.decodeJSON(w, r, &req) {
		return
	}
	if !h.validate(w, &req) {
		return
	}

	o := &PropertyOption{
		GroupID:  groupID,
		Position: req.Position,
		ColorHex: req.ColorHex,
	}
	for _, t := range req.Translations {
		o.Translations = append(o.Translations, PropertyOptionTranslation{
			Locale: t.Locale,
			Name:   t.Name,
		})
	}

	if err := h.service.CreatePropertyOption(r.Context(), o); err != nil {
		h.serverError(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, apiResponse{Data: propertyOptionToResponse(*o)})
}

// adminUpdatePropertyOption handles PUT /property-groups/{id}/options/{optId}
func (h *Handler) adminUpdatePropertyOption(w http.ResponseWriter, r *http.Request) {
	groupID, ok := h.parseUUID(w, r, "id")
	if !ok {
		return
	}
	optID, ok := h.parseUUID(w, r, "optId")
	if !ok {
		return
	}

	var req UpdatePropertyOptionRequest
	if !h.decodeJSON(w, r, &req) {
		return
	}
	if !h.validate(w, &req) {
		return
	}

	o := &PropertyOption{
		ID:       optID,
		GroupID:  groupID,
		Position: req.Position,
		ColorHex: req.ColorHex,
	}
	for _, t := range req.Translations {
		o.Translations = append(o.Translations, PropertyOptionTranslation{
			OptionID: optID,
			Locale:   t.Locale,
			Name:     t.Name,
		})
	}

	if err := h.service.UpdatePropertyOption(r.Context(), o); err != nil {
		if errors.Is(err, ErrNotFound) {
			h.notFound(w, "property option not found")
			return
		}
		h.serverError(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, apiResponse{Data: propertyOptionToResponse(*o)})
}

// adminDeletePropertyOption handles DELETE /property-groups/{id}/options/{optId}
func (h *Handler) adminDeletePropertyOption(w http.ResponseWriter, r *http.Request) {
	_, ok := h.parseUUID(w, r, "id")
	if !ok {
		return
	}
	optID, ok := h.parseUUID(w, r, "optId")
	if !ok {
		return
	}
	if err := h.service.DeletePropertyOption(r.Context(), optID); err != nil {
		if errors.Is(err, ErrNotFound) {
			h.notFound(w, "property option not found")
			return
		}
		h.serverError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// mustMarshal marshals a value to JSON, panicking on error (only used for internal re-encoding).
func mustMarshal(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// ---------------------------------------------------------------------------
// Store handlers
// ---------------------------------------------------------------------------

// storeList handles GET /products (active products only)
// Query params: page, limit, sort, order, search, category_id
func (h *Handler) storeList(w http.ResponseWriter, r *http.Request) {
	filter, page, limit := h.parseListFilter(r)

	// Store API always filters to active-only products.
	active := true
	filter.Active = &active

	products, total, err := h.service.List(r.Context(), filter)
	if err != nil {
		h.serverError(w, r, err)
		return
	}

	pages := 0
	if limit > 0 {
		pages = int(math.Ceil(float64(total) / float64(limit)))
	}

	h.writeJSON(w, http.StatusOK, apiResponse{
		Data: ToResponseList(products),
		Meta: &apiMeta{
			Total: total,
			Page:  page,
			Limit: limit,
			Pages: pages,
		},
	})
}

// storeGetBySlug handles GET /products/{slug}
func (h *Handler) storeGetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		h.writeError(w, http.StatusBadRequest, "invalid_param", "slug is required", "slug")
		return
	}

	locale := parseLocale(r)

	p, err := h.service.GetBySlug(r.Context(), slug, locale)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			h.notFound(w, "product not found")
			return
		}
		h.serverError(w, r, err)
		return
	}

	resp := ToResponse(p)
	h.writeJSON(w, http.StatusOK, apiResponse{Data: resp})
}

// storeGetByID handles GET /products/id/{id}
func (h *Handler) storeGetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_param", "invalid product id", "id")
		return
	}

	p, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			h.notFound(w, "product not found")
			return
		}
		h.serverError(w, r, err)
		return
	}

	resp := ToResponse(p)
	h.writeJSON(w, http.StatusOK, apiResponse{Data: resp})
}

// ---------------------------------------------------------------------------
// Parsing helpers
// ---------------------------------------------------------------------------

// parseListFilter builds a ProductFilter from URL query parameters.
func (h *Handler) parseListFilter(r *http.Request) (ProductFilter, int, int) {
	q := r.URL.Query()

	page := 1
	if v := q.Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}

	limit := 25
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	filter := ProductFilter{
		Page:   page,
		Limit:  limit,
		Search: q.Get("search"),
		Sort:   q.Get("sort"),
		Order:  q.Get("order"),
	}

	if v := q.Get("category_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			filter.CategoryID = &id
		}
	}

	// Admin may pass filter[active]=true/false; store handler overrides this.
	if v := q.Get("filter[active]"); v != "" {
		b := v == "true" || v == "1"
		filter.Active = &b
	}

	return filter, page, limit
}

// parseLocale extracts the primary locale tag from the Accept-Language header,
// defaulting to "en" when the header is absent or malformed.
func parseLocale(r *http.Request) string {
	al := r.Header.Get("Accept-Language")
	if al == "" {
		return "en"
	}
	// Accept-Language: en-US,en;q=0.9,de;q=0.8
	// Take the first tag and strip quality value.
	parts := strings.SplitN(al, ",", 2)
	lang := strings.TrimSpace(parts[0])
	if idx := strings.Index(lang, ";"); idx != -1 {
		lang = lang[:idx]
	}
	lang = strings.TrimSpace(lang)
	if lang == "" {
		return "en"
	}
	return lang
}

// parseUUID reads a chi URL parameter as a UUID, writing a 400 on failure.
func (h *Handler) parseUUID(w http.ResponseWriter, r *http.Request, param string) (uuid.UUID, bool) {
	raw := chi.URLParam(r, param)
	id, err := uuid.Parse(raw)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_param", param+" must be a valid UUID", param)
		return uuid.Nil, false
	}
	return id, true
}

// decodeJSON decodes the request body into dst, writing a 400 on failure.
func (h *Handler) decodeJSON(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_body", "request body is not valid JSON: "+err.Error(), "")
		return false
	}
	return true
}

// validate runs the go-playground validator and writes validation errors on failure.
func (h *Handler) validate(w http.ResponseWriter, v interface{}) bool {
	if err := h.validator.Struct(v); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			errs := make([]apiError, 0, len(ve))
			for _, fe := range ve {
				errs = append(errs, apiError{
					Code:   "validation_error",
					Detail: fe.Tag() + " constraint violated on " + fe.Field(),
					Field:  fe.Field(),
				})
			}
			h.writeJSON(w, http.StatusUnprocessableEntity, apiResponse{Errors: errs})
			return false
		}
		h.writeError(w, http.StatusUnprocessableEntity, "validation_error", err.Error(), "")
		return false
	}
	return true
}

// ---------------------------------------------------------------------------
// Response helpers
// ---------------------------------------------------------------------------

func (h *Handler) writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode JSON response")
	}
}

func (h *Handler) writeError(w http.ResponseWriter, status int, code, detail, field string) {
	h.writeJSON(w, status, apiResponse{
		Errors: []apiError{{Code: code, Detail: detail, Field: field}},
	})
}

func (h *Handler) notFound(w http.ResponseWriter, detail string) {
	h.writeError(w, http.StatusNotFound, "not_found", detail, "")
}

func (h *Handler) serverError(w http.ResponseWriter, r *http.Request, err error) {
	h.logger.Error().Err(err).Str("method", r.Method).Str("path", r.URL.Path).Msg("internal server error")
	h.writeError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred", "")
}
