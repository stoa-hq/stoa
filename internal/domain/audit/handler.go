package audit

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/internal/server"
)

type handler struct {
	svc    AuditService
	logger zerolog.Logger
}

// NewHandler creates a new audit HTTP handler.
func NewHandler(svc AuditService, logger zerolog.Logger) *handler {
	return &handler{
		svc:    svc,
		logger: logger,
	}
}

// RegisterAdminRoutes mounts the admin-only audit log routes on r.
func (h *handler) RegisterAdminRoutes(r chi.Router) {
	r.Route("/audit-log", func(r chi.Router) {
		r.Get("/", h.list)
	})
}

// --- handlers ---

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	filter := AuditFilter{
		Page:       parseIntQuery(r, "page", 1),
		Limit:      parseIntQuery(r, "limit", 20),
		EntityType: r.URL.Query().Get("entity_type"),
		Action:     r.URL.Query().Get("action"),
	}

	if raw := r.URL.Query().Get("user_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_user_id", "user_id must be a valid UUID")
			return
		}
		filter.UserID = &id
	}
	if raw := r.URL.Query().Get("entity_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_entity_id", "entity_id must be a valid UUID")
			return
		}
		filter.EntityID = &id
	}

	logs, total, err := h.svc.List(r.Context(), filter)
	if err != nil {
		h.serverError(w, r, err)
		return
	}

	pages := 0
	if filter.Limit > 0 {
		pages = (total + filter.Limit - 1) / filter.Limit
	}
	writeJSON(w, http.StatusOK, apiResponse{
		Data: logs,
		Meta: &apiMeta{Total: total, Page: filter.Page, Limit: filter.Limit, Pages: pages},
	})
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
