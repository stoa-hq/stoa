package category

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Handler exposes category functionality over HTTP.
type Handler struct {
	svc      *Service
	validate *validator.Validate
	logger   zerolog.Logger
}

// NewHandler constructs a Handler.
func NewHandler(svc *Service, logger zerolog.Logger) *Handler {
	return &Handler{
		svc:      svc,
		validate: validator.New(),
		logger:   logger,
	}
}

// -------------------------------------------------------------------------
// Route registration
// -------------------------------------------------------------------------

// RegisterAdminRoutes mounts admin CRUD routes on r.
// Caller is responsible for applying authentication/authorization middleware
// to the sub-router before passing it here.
//
//	r.Route("/admin/categories", func(r chi.Router) {
//	    r.Use(authMW.Authenticate, authMW.RequirePermission(auth.PermCategoryRead))
//	    category.NewHandler(svc, logger).RegisterAdminRoutes(r)
//	})
func (h *Handler) RegisterAdminRoutes(r chi.Router) {
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
}

// RegisterStoreRoutes mounts the read-only storefront routes on r.
func (h *Handler) RegisterStoreRoutes(r chi.Router) {
	r.Get("/tree", h.GetTree)
}

// -------------------------------------------------------------------------
// Admin handlers
// -------------------------------------------------------------------------

// List handles GET /admin/categories
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page := 1
	if p, err := strconv.Atoi(q.Get("page")); err == nil && p > 0 {
		page = p
	}
	limit := 20
	if l, err := strconv.Atoi(q.Get("limit")); err == nil && l > 0 && l <= 200 {
		limit = l
	}

	filter := CategoryFilter{
		Page:  page,
		Limit: limit,
	}

	if raw := q.Get("parent_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_param", "parent_id must be a valid UUID", "")
			return
		}
		filter.ParentID = &id
	}
	if raw := q.Get("active"); raw != "" {
		b, err := strconv.ParseBool(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_param", "active must be a boolean", "")
			return
		}
		filter.Active = &b
	}

	cats, total, err := h.svc.List(r.Context(), filter)
	if err != nil {
		h.logger.Error().Err(err).Msg("List categories failed")
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list categories", "")
		return
	}

	pages := int(math.Ceil(float64(total) / float64(limit)))
	writeJSON(w, http.StatusOK, apiResponse{
		Data: ToResponseList(cats),
		Meta: &apiMeta{Total: total, Page: page, Limit: limit, Pages: pages},
	})
}

// Create handles POST /admin/categories
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "request body is not valid JSON", "")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		writeValidationErrors(w, err)
		return
	}

	cat := req.ToEntity()
	if err := h.svc.Create(r.Context(), cat); err != nil {
		h.logger.Error().Err(err).Msg("Create category failed")
		writeError(w, http.StatusInternalServerError, "internal_error", "could not create category", "")
		return
	}

	writeJSON(w, http.StatusCreated, apiResponse{Data: ToResponse(cat)})
}

// GetByID handles GET /admin/categories/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r)
	if !ok {
		return
	}

	cat, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "category not found", "")
			return
		}
		h.logger.Error().Err(err).Msg("GetByID category failed")
		writeError(w, http.StatusInternalServerError, "internal_error", "could not fetch category", "")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: ToResponse(cat)})
}

// Update handles PUT /admin/categories/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r)
	if !ok {
		return
	}

	var req UpdateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "request body is not valid JSON", "")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		writeValidationErrors(w, err)
		return
	}

	cat, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "category not found", "")
			return
		}
		h.logger.Error().Err(err).Msg("Update: GetByID failed")
		writeError(w, http.StatusInternalServerError, "internal_error", "could not fetch category", "")
		return
	}

	req.ApplyTo(cat)
	if err := h.svc.Update(r.Context(), cat); err != nil {
		h.logger.Error().Err(err).Msg("Update category failed")
		writeError(w, http.StatusInternalServerError, "internal_error", "could not update category", "")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: ToResponse(cat)})
}

// Delete handles DELETE /admin/categories/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r)
	if !ok {
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "category not found", "")
			return
		}
		h.logger.Error().Err(err).Msg("Delete category failed")
		writeError(w, http.StatusInternalServerError, "internal_error", "could not delete category", "")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// -------------------------------------------------------------------------
// Store handlers
// -------------------------------------------------------------------------

// GetTree handles GET /store/categories/tree
func (h *Handler) GetTree(w http.ResponseWriter, r *http.Request) {
	locale := r.URL.Query().Get("locale")
	if locale == "" {
		locale = "de-DE"
	}

	tree, err := h.svc.GetTree(r.Context(), locale)
	if err != nil {
		h.logger.Error().Err(err).Msg("GetTree failed")
		writeError(w, http.StatusInternalServerError, "internal_error", "could not fetch category tree", "")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: ToResponseList(tree)})
}

// -------------------------------------------------------------------------
// Internal helpers
// -------------------------------------------------------------------------

func parseUUID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	raw := chi.URLParam(r, "id")
	id, err := uuid.Parse(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID", "id")
		return uuid.Nil, false
	}
	return id, true
}

// apiResponse mirrors server.APIResponse without creating a cross-package dependency.
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

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, detail, field string) {
	writeJSON(w, status, apiResponse{
		Errors: []apiError{{Code: code, Detail: detail, Field: field}},
	})
}

func writeValidationErrors(w http.ResponseWriter, err error) {
	var verr validator.ValidationErrors
	if !errors.As(err, &verr) {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), "")
		return
	}
	errs := make([]apiError, 0, len(verr))
	for _, fe := range verr {
		errs = append(errs, apiError{
			Code:   "validation_error",
			Detail: fe.Tag() + " constraint violated",
			Field:  fe.Field(),
		})
	}
	writeJSON(w, http.StatusUnprocessableEntity, apiResponse{Errors: errs})
}
