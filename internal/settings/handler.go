package settings

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/internal/config"
)

type handler struct {
	cfg    *config.Config
	logger zerolog.Logger
}

// NewHandler creates a new settings HTTP handler.
func NewHandler(cfg *config.Config, logger zerolog.Logger) *handler {
	return &handler{
		cfg:    cfg,
		logger: logger,
	}
}

// RegisterStoreRoutes mounts the public store config route.
func (h *handler) RegisterStoreRoutes(r chi.Router) {
	r.Get("/config", h.getConfig)
}

// RegisterAdminRoutes mounts the admin config route.
func (h *handler) RegisterAdminRoutes(r chi.Router) {
	r.Get("/config", h.getConfig)
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

// --- local response helpers ---

type apiResponse struct {
	Data   interface{} `json:"data,omitempty"`
	Errors []apiError  `json:"errors,omitempty"`
}

type apiError struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
