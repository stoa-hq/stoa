package discount

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/internal/server"
)

type handler struct {
	svc      DiscountService
	validate *validator.Validate
	logger   zerolog.Logger
}

// NewHandler creates a new discount HTTP handler.
func NewHandler(svc DiscountService, logger zerolog.Logger) *handler {
	return &handler{
		svc:      svc,
		validate: validator.New(),
		logger:   logger,
	}
}

// RegisterAdminRoutes mounts the admin-only discount routes on r.
func (h *handler) RegisterAdminRoutes(r chi.Router) {
	r.Route("/discounts", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Get("/{id}", h.getByID)
		r.Put("/{id}", h.update)
		r.Delete("/{id}", h.delete)
		r.Post("/validate", h.validateCode)
		r.Post("/{id}/apply", h.applyDiscount)
	})
}

// --- handlers ---

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	filter := DiscountFilter{
		Page:  parseIntQuery(r, "page", 1),
		Limit: parseIntQuery(r, "limit", 20),
		Type:  r.URL.Query().Get("type"),
		Code:  r.URL.Query().Get("code"),
	}
	if raw := r.URL.Query().Get("active"); raw != "" {
		active := raw == "true"
		filter.Active = &active
	}

	discounts, total, err := h.svc.List(r.Context(), filter)
	if err != nil {
		h.serverError(w, r, err)
		return
	}

	pages := 0
	if filter.Limit > 0 {
		pages = (total + filter.Limit - 1) / filter.Limit
	}
	writeJSON(w, http.StatusOK, apiResponse{
		Data: discounts,
		Meta: &apiMeta{Total: total, Page: filter.Page, Limit: filter.Limit, Pages: pages},
	})
}

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	var req CreateDiscountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "body_too_large", "request body exceeds size limit")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		writeValidationErrors(w, err)
		return
	}

	d := &Discount{
		Code:          req.Code,
		Type:          req.Type,
		Value:         req.Value,
		MinOrderValue: req.MinOrderValue,
		MaxUses:       req.MaxUses,
		ValidFrom:     req.ValidFrom,
		ValidUntil:    req.ValidUntil,
		Active:        req.Active,
		Conditions:    req.Conditions,
	}
	if err := h.svc.Create(r.Context(), d); err != nil {
		h.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, apiResponse{Data: d})
}

func (h *handler) getByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	d, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "discount not found")
			return
		}
		h.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: d})
}

func (h *handler) update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	var req UpdateDiscountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "body_too_large", "request body exceeds size limit")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		writeValidationErrors(w, err)
		return
	}

	d := &Discount{
		ID:            id,
		Code:          req.Code,
		Type:          req.Type,
		Value:         req.Value,
		MinOrderValue: req.MinOrderValue,
		MaxUses:       req.MaxUses,
		ValidFrom:     req.ValidFrom,
		ValidUntil:    req.ValidUntil,
		Active:        req.Active,
		Conditions:    req.Conditions,
	}
	if err := h.svc.Update(r.Context(), d); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "discount not found")
			return
		}
		h.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: d})
}

func (h *handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "discount not found")
			return
		}
		h.serverError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) validateCode(w http.ResponseWriter, r *http.Request) {
	var req ValidateCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "body_too_large", "request body exceeds size limit")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		writeValidationErrors(w, err)
		return
	}

	d, err := h.svc.ValidateCode(r.Context(), req.Code, req.OrderTotal)
	if err != nil {
		if errors.Is(err, ErrCodeInvalid) {
			writeError(w, http.StatusUnprocessableEntity, "code_invalid", err.Error())
			return
		}
		if errors.Is(err, ErrMaxUsesReached) {
			writeError(w, http.StatusUnprocessableEntity, "max_uses_reached", err.Error())
			return
		}
		if errors.Is(err, ErrInvalidInput) {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
			return
		}
		h.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: d})
}

func (h *handler) applyDiscount(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	if err := h.svc.ApplyDiscount(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "discount not found")
			return
		}
		h.serverError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- shared response helpers ---

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
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, detail string) {
	writeJSON(w, status, apiResponse{
		Errors: []apiError{{Code: code, Detail: detail}},
	})
}

func (h *handler) serverError(w http.ResponseWriter, r *http.Request, err error) {
	h.logger.Error().Err(err).Str("request_id", server.RequestID(r.Context())).Str("method", r.Method).Str("path", r.URL.Path).Msg("internal server error")
	writeError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
}

func writeValidationErrors(w http.ResponseWriter, err error) {
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		writeError(w, http.StatusBadRequest, "validation_failed", "invalid request data")
		return
	}
	errs := make([]apiError, 0, len(ve))
	for _, fe := range ve {
		errs = append(errs, apiError{
			Code:   "validation_error",
			Detail: fe.Tag(),
			Field:  fe.Field(),
		})
	}
	writeJSON(w, http.StatusUnprocessableEntity, apiResponse{Errors: errs})
}

func parseIntQuery(r *http.Request, key string, defaultVal int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 1 {
		return defaultVal
	}
	return v
}
