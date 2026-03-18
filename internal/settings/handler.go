package settings

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/internal/config"
)

type handler struct {
	svc      *Service
	cfg      *config.Config
	validate *validator.Validate
	logger   zerolog.Logger
}

// NewHandler creates a new settings HTTP handler.
func NewHandler(svc *Service, cfg *config.Config, validate *validator.Validate, logger zerolog.Logger) *handler {
	return &handler{
		svc:      svc,
		cfg:      cfg,
		validate: validate,
		logger:   logger,
	}
}

// RegisterStoreRoutes mounts the public store config route.
func (h *handler) RegisterStoreRoutes(r chi.Router) {
	r.Get("/config", h.getConfig)
	r.Get("/settings", h.getSettings)
}

// RegisterAdminRoutes mounts the admin config route.
func (h *handler) RegisterAdminRoutes(r chi.Router) {
	r.Get("/config", h.getConfig)
	r.Get("/settings", h.getSettings)
	r.Put("/settings", h.updateSettings)
}

// ConfigResponse is the DTO returned by the config endpoint.
type ConfigResponse struct {
	DefaultLocale    string   `json:"default_locale"`
	AvailableLocales []string `json:"available_locales"`
}

func (h *handler) getConfig(w http.ResponseWriter, r *http.Request) {
	resp := ConfigResponse{
		DefaultLocale:    h.cfg.I18n.DefaultLocale,
		AvailableLocales: h.cfg.I18n.AvailableLocales,
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: resp})
}

func (h *handler) getSettings(w http.ResponseWriter, r *http.Request) {
	s, err := h.svc.Get(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load settings")
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: s})
}

func (h *handler) updateSettings(w http.ResponseWriter, r *http.Request) {
	var req UpdateSettingsRequest
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

	s := &StoreSettings{
		StoreName:        req.StoreName,
		StoreDescription: req.StoreDescription,
		LogoURL:          req.LogoURL,
		FaviconURL:       req.FaviconURL,
		ContactEmail:     req.ContactEmail,
		Currency:         req.Currency,
		Country:          req.Country,
		Timezone:         req.Timezone,
		CopyrightText:    req.CopyrightText,
		MaintenanceMode:  req.MaintenanceMode,
	}

	result, err := h.svc.Update(r.Context(), s)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, "invalid_input", "store_name is required")
			return
		}
		writeError(w, http.StatusInternalServerError, "update_failed", "failed to update settings")
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: result})
}

// --- local response helpers ---

type apiResponse struct {
	Data   interface{} `json:"data,omitempty"`
	Errors []apiError  `json:"errors,omitempty"`
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
