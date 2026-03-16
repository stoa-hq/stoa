package warehouse

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Handler provides HTTP handlers for the warehouse domain.
type Handler struct {
	svc      *Service
	validate *validator.Validate
	logger   zerolog.Logger
}

// NewHandler creates a new warehouse HTTP handler.
func NewHandler(svc *Service, validate *validator.Validate, logger zerolog.Logger) *Handler {
	return &Handler{
		svc:      svc,
		validate: validate,
		logger:   logger,
	}
}

// RegisterAdminRoutes mounts the admin-only warehouse routes on r.
func (h *Handler) RegisterAdminRoutes(r chi.Router) {
	r.Route("/warehouses", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Get("/{id}", h.getByID)
		r.Put("/{id}", h.update)
		r.Delete("/{id}", h.delete)
		r.Get("/{id}/stock", h.getStockByWarehouse)
		r.Put("/{id}/stock", h.setStock)
		r.Delete("/{id}/stock/{stockID}", h.removeStock)
	})

	// Product stock overview (mounted separately).
	r.Get("/products/{productID}/stock", h.getStockByProduct)
}

// --- handlers ---

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	filter := WarehouseFilter{
		Page:  parseIntQuery(r, "page", 1),
		Limit: parseIntQuery(r, "limit", 20),
	}
	if raw := r.URL.Query().Get("active"); raw != "" {
		active := raw == "true"
		filter.Active = &active
	}

	warehouses, total, err := h.svc.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	pages := 0
	if filter.Limit > 0 {
		pages = (total + filter.Limit - 1) / filter.Limit
	}
	writeJSON(w, http.StatusOK, apiResponse{
		Data: warehouses,
		Meta: &apiMeta{Total: total, Page: filter.Page, Limit: filter.Limit, Pages: pages},
	})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req CreateWarehouseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		writeValidationErrors(w, err)
		return
	}

	wh := &Warehouse{
		Name:               req.Name,
		Code:               req.Code,
		Active:             req.Active,
		Priority:           req.Priority,
		AddressLine1:       req.AddressLine1,
		AddressLine2:       req.AddressLine2,
		City:               req.City,
		State:              req.State,
		PostalCode:         req.PostalCode,
		Country:            req.Country,
		CustomFields:       req.CustomFields,
		Metadata:           req.Metadata,
	}
	if req.AllowNegativeStock != nil {
		wh.AllowNegativeStock = *req.AllowNegativeStock
	}

	if err := h.svc.Create(r.Context(), wh); err != nil {
		if errors.Is(err, ErrDuplicateCode) {
			writeError(w, http.StatusConflict, "duplicate_code", "a warehouse with this code already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, apiResponse{Data: wh})
}

func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	wh, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "warehouse not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: wh})
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	var req UpdateWarehouseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		writeValidationErrors(w, err)
		return
	}

	wh := &Warehouse{
		ID:           id,
		Name:         req.Name,
		Code:         req.Code,
		Active:       req.Active,
		Priority:     req.Priority,
		AddressLine1: req.AddressLine1,
		AddressLine2: req.AddressLine2,
		City:         req.City,
		State:        req.State,
		PostalCode:   req.PostalCode,
		Country:      req.Country,
		CustomFields: req.CustomFields,
		Metadata:     req.Metadata,
	}
	if req.AllowNegativeStock != nil {
		wh.AllowNegativeStock = *req.AllowNegativeStock
	}

	if err := h.svc.Update(r.Context(), wh); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "warehouse not found")
			return
		}
		if errors.Is(err, ErrDuplicateCode) {
			writeError(w, http.StatusConflict, "duplicate_code", "a warehouse with this code already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: wh})
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "warehouse not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "delete_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getStockByWarehouse(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	stocks, err := h.svc.GetStockByWarehouse(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: stocks})
}

func (h *Handler) setStock(w http.ResponseWriter, r *http.Request) {
	warehouseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	var req SetStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		writeValidationErrors(w, err)
		return
	}

	var results []*WarehouseStock
	for _, item := range req.Items {
		ws, err := h.svc.SetStock(r.Context(), warehouseID, item.ProductID, item.VariantID, item.Quantity, item.Reference)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "set_stock_failed", err.Error())
			return
		}
		results = append(results, ws)
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: results})
}

func (h *Handler) removeStock(w http.ResponseWriter, r *http.Request) {
	stockID, err := uuid.Parse(chi.URLParam(r, "stockID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "stockID must be a valid UUID")
		return
	}

	if err := h.svc.RemoveStock(r.Context(), stockID); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "stock entry not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "remove_stock_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getStockByProduct(w http.ResponseWriter, r *http.Request) {
	productID, err := uuid.Parse(chi.URLParam(r, "productID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "productID must be a valid UUID")
		return
	}

	stocks, err := h.svc.GetStockByProduct(r.Context(), productID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: stocks})
}

// --- shared response helpers (local per handler) ---

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
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeError(w http.ResponseWriter, status int, code, detail string) {
	writeJSON(w, status, apiResponse{
		Errors: []apiError{{Code: code, Detail: detail}},
	})
}

func writeValidationErrors(w http.ResponseWriter, err error) {
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		writeError(w, http.StatusBadRequest, "validation_failed", err.Error())
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
