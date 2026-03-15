package payment

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

type handler struct {
	methodSvc      PaymentMethodService
	transactionSvc PaymentTransactionService
	validate       *validator.Validate
	logger         zerolog.Logger
}

// NewHandler creates a new payment HTTP handler.
func NewHandler(methodSvc PaymentMethodService, transactionSvc PaymentTransactionService, logger zerolog.Logger) *handler {
	return &handler{
		methodSvc:      methodSvc,
		transactionSvc: transactionSvc,
		validate:       validator.New(),
		logger:         logger,
	}
}

// RegisterAdminRoutes mounts the admin-only payment method routes on r.
func (h *handler) RegisterAdminRoutes(r chi.Router) {
	r.Route("/payment-methods", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Get("/{id}", h.getByID)
		r.Put("/{id}", h.update)
		r.Delete("/{id}", h.delete)
	})
}

// ListTransactionsByOrder returns all payment transactions for a given order.
func (h *handler) ListTransactionsByOrder(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(chi.URLParam(r, "orderID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "orderID must be a valid UUID")
		return
	}

	txns, err := h.transactionSvc.GetTransactionsByOrderID(r.Context(), orderID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load transactions")
		return
	}
	if txns == nil {
		txns = []PaymentTransaction{}
	}

	writeJSON(w, http.StatusOK, apiResponse{
		Data: txns,
		Meta: &apiMeta{Total: len(txns), Page: 1, Limit: len(txns), Pages: 1},
	})
}

// RegisterStoreRoutes mounts store-facing (active-only) payment method routes on r.
func (h *handler) RegisterStoreRoutes(r chi.Router) {
	r.Route("/payment-methods", func(r chi.Router) {
		r.Get("/", h.listActive)
		r.Get("/{id}", h.getByID)
	})
}

// --- handlers ---

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	filter := PaymentMethodFilter{
		Page:  parseIntQuery(r, "page", 1),
		Limit: parseIntQuery(r, "limit", 20),
	}
	if raw := r.URL.Query().Get("active"); raw != "" {
		active := raw == "true"
		filter.Active = &active
	}

	methods, total, err := h.methodSvc.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	pages := 0
	if filter.Limit > 0 {
		pages = (total + filter.Limit - 1) / filter.Limit
	}
	writeJSON(w, http.StatusOK, apiResponse{
		Data: methods,
		Meta: &apiMeta{Total: total, Page: filter.Page, Limit: filter.Limit, Pages: pages},
	})
}

func (h *handler) listActive(w http.ResponseWriter, r *http.Request) {
	active := true
	filter := PaymentMethodFilter{
		Page:   parseIntQuery(r, "page", 1),
		Limit:  parseIntQuery(r, "limit", 20),
		Active: &active,
	}

	methods, total, err := h.methodSvc.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	pages := 0
	if filter.Limit > 0 {
		pages = (total + filter.Limit - 1) / filter.Limit
	}
	writeJSON(w, http.StatusOK, apiResponse{
		Data: methods,
		Meta: &apiMeta{Total: total, Page: filter.Page, Limit: filter.Limit, Pages: pages},
	})
}

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	var req CreatePaymentMethodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		writeValidationErrors(w, err)
		return
	}

	m := &PaymentMethod{
		Provider:     req.Provider,
		Active:       req.Active,
		Config:       req.Config,
		CustomFields: req.CustomFields,
		Translations: make([]PaymentMethodTranslation, 0, len(req.Translations)),
	}
	for _, t := range req.Translations {
		m.Translations = append(m.Translations, PaymentMethodTranslation{
			Locale:      t.Locale,
			Name:        t.Name,
			Description: t.Description,
		})
	}

	if err := h.methodSvc.Create(r.Context(), m); err != nil {
		writeError(w, http.StatusInternalServerError, "create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, apiResponse{Data: m})
}

func (h *handler) getByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	m, err := h.methodSvc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrMethodNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "payment method not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: m})
}

func (h *handler) update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	var req UpdatePaymentMethodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		writeValidationErrors(w, err)
		return
	}

	m := &PaymentMethod{
		ID:           id,
		Provider:     req.Provider,
		Active:       req.Active,
		Config:       req.Config,
		CustomFields: req.CustomFields,
		Translations: make([]PaymentMethodTranslation, 0, len(req.Translations)),
	}
	for _, t := range req.Translations {
		m.Translations = append(m.Translations, PaymentMethodTranslation{
			Locale:      t.Locale,
			Name:        t.Name,
			Description: t.Description,
		})
	}

	if err := h.methodSvc.Update(r.Context(), m); err != nil {
		if errors.Is(err, ErrMethodNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "payment method not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: m})
}

func (h *handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	if err := h.methodSvc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrMethodNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "payment method not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "delete_failed", err.Error())
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
