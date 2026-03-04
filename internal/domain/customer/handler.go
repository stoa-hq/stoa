package customer

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

// Handler handles HTTP requests for the customer domain.
type Handler struct {
	service   *CustomerService
	validator *validator.Validate
	logger    zerolog.Logger
}

// NewHandler creates a new customer Handler.
func NewHandler(service *CustomerService, validate *validator.Validate, logger zerolog.Logger) *Handler {
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
// Expected prefix: /customers
func (h *Handler) RegisterAdminRoutes(r chi.Router) {
	r.Get("/customers", h.adminList)
	r.Post("/customers", h.adminCreate)
	r.Get("/customers/{id}", h.adminGetByID)
	r.Put("/customers/{id}", h.adminUpdate)
	r.Delete("/customers/{id}", h.adminDelete)
}

// RegisterStoreRoutes mounts the self-service account endpoints.
// Expected prefix: (root of store router)
func (h *Handler) RegisterStoreRoutes(r chi.Router) {
	r.Post("/register", h.storeRegister)
	r.Get("/account", h.storeGetAccount)
	r.Put("/account", h.storeUpdateAccount)
}

// ---------------------------------------------------------------------------
// Admin handlers
// ---------------------------------------------------------------------------

// adminList handles GET /customers
// Query params: page, limit, search, active
func (h *Handler) adminList(w http.ResponseWriter, r *http.Request) {
	filter, page, limit := h.parseListFilter(r)

	customers, total, err := h.service.List(r.Context(), filter)
	if err != nil {
		h.serverError(w, r, err)
		return
	}

	pages := 0
	if limit > 0 {
		pages = int(math.Ceil(float64(total) / float64(limit)))
	}

	items := make([]CustomerResponse, len(customers))
	for i := range customers {
		items[i] = ToResponse(&customers[i])
	}

	h.writeJSON(w, http.StatusOK, apiResponse{
		Data: items,
		Meta: &apiMeta{
			Total: total,
			Page:  page,
			Limit: limit,
			Pages: pages,
		},
	})
}

// adminGetByID handles GET /customers/{id}
func (h *Handler) adminGetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUID(w, r, "id")
	if !ok {
		return
	}

	c, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			h.notFound(w, "customer not found")
			return
		}
		h.serverError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusOK, apiResponse{Data: ToResponse(c)})
}

// adminCreate handles POST /customers
func (h *Handler) adminCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateCustomerRequest
	if !h.decodeJSON(w, r, &req) {
		return
	}
	if !h.validate(w, &req) {
		return
	}

	c, err := h.service.Create(r.Context(), CreateCustomerInput{
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	})
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			h.writeError(w, http.StatusConflict, "email_taken", "email address is already in use", "email")
			return
		}
		h.serverError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, apiResponse{Data: ToResponse(c)})
}

// adminUpdate handles PUT /customers/{id}
func (h *Handler) adminUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUID(w, r, "id")
	if !ok {
		return
	}

	var req UpdateCustomerRequest
	if !h.decodeJSON(w, r, &req) {
		return
	}
	if !h.validate(w, &req) {
		return
	}

	c, err := h.service.Update(r.Context(), id, UpdateCustomerInput{
		Email:                    req.Email,
		FirstName:                req.FirstName,
		LastName:                 req.LastName,
		Active:                   req.Active,
		DefaultBillingAddressID:  req.DefaultBillingAddressID,
		DefaultShippingAddressID: req.DefaultShippingAddressID,
		CustomFields:             req.CustomFields,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			h.notFound(w, "customer not found")
		case errors.Is(err, ErrEmailTaken):
			h.writeError(w, http.StatusConflict, "email_taken", "email address is already in use", "email")
		default:
			h.serverError(w, r, err)
		}
		return
	}

	h.writeJSON(w, http.StatusOK, apiResponse{Data: ToResponse(c)})
}

// adminDelete handles DELETE /customers/{id}
func (h *Handler) adminDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUID(w, r, "id")
	if !ok {
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			h.notFound(w, "customer not found")
			return
		}
		h.serverError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Store handlers
// ---------------------------------------------------------------------------

// storeRegister handles POST /register
func (h *Handler) storeRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if !h.decodeJSON(w, r, &req) {
		return
	}
	if !h.validate(w, &req) {
		return
	}

	c, err := h.service.Create(r.Context(), CreateCustomerInput{
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	})
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			h.writeError(w, http.StatusConflict, "email_taken", "email address is already in use", "email")
			return
		}
		h.serverError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, apiResponse{Data: ToResponse(c)})
}

// storeGetAccount handles GET /account
// Requires the customer's UUID to be present in the request context (set by
// the auth middleware under the ctxKeyUserID key from the auth package).
func (h *Handler) storeGetAccount(w http.ResponseWriter, r *http.Request) {
	customerID, ok := h.customerIDFromContext(w, r)
	if !ok {
		return
	}

	c, err := h.service.GetByID(r.Context(), customerID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			h.notFound(w, "customer not found")
			return
		}
		h.serverError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusOK, apiResponse{Data: ToResponse(c)})
}

// storeUpdateAccount handles PUT /account
func (h *Handler) storeUpdateAccount(w http.ResponseWriter, r *http.Request) {
	customerID, ok := h.customerIDFromContext(w, r)
	if !ok {
		return
	}

	var req UpdateAccountRequest
	if !h.decodeJSON(w, r, &req) {
		return
	}
	if !h.validate(w, &req) {
		return
	}

	c, err := h.service.Update(r.Context(), customerID, UpdateCustomerInput{
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			h.notFound(w, "customer not found")
		case errors.Is(err, ErrEmailTaken):
			h.writeError(w, http.StatusConflict, "email_taken", "email address is already in use", "email")
		default:
			h.serverError(w, r, err)
		}
		return
	}

	h.writeJSON(w, http.StatusOK, apiResponse{Data: ToResponse(c)})
}

// ---------------------------------------------------------------------------
// Parsing helpers
// ---------------------------------------------------------------------------

// parseListFilter builds a CustomerFilter from URL query parameters.
func (h *Handler) parseListFilter(r *http.Request) (CustomerFilter, int, int) {
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

	filter := CustomerFilter{
		Page:   page,
		Limit:  limit,
		Search: q.Get("search"),
	}

	if v := q.Get("filter[active]"); v != "" {
		b := v == "true" || v == "1"
		filter.Active = &b
	}

	return filter, page, limit
}

// customerIDFromContext extracts the authenticated customer's UUID from the
// request context. It writes a 401 and returns false when the ID is absent.
func (h *Handler) customerIDFromContext(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	// The auth middleware stores the user ID as a uuid.UUID under its private
	// contextKeyType key.  We receive it here through the standard context
	// value mechanism using the same string key ("user_id").
	type contextKeyType string
	const ctxKeyUserID contextKeyType = "user_id"

	id, ok := r.Context().Value(ctxKeyUserID).(uuid.UUID)
	if !ok || id == uuid.Nil {
		h.writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required", "")
		return uuid.Nil, false
	}
	return id, true
}

// parseLocale extracts the primary locale tag from the Accept-Language header,
// defaulting to "en" when the header is absent or malformed.
func parseLocale(r *http.Request) string {
	al := r.Header.Get("Accept-Language")
	if al == "" {
		return "en"
	}
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

// Ensure parseLocale is used (referenced indirectly; kept for store route locale work).
var _ = parseLocale
