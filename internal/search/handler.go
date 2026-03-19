package search

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

type handler struct {
	engine Engine
	logger zerolog.Logger
}

// NewHandler creates a new search HTTP handler.
func NewHandler(engine Engine, logger zerolog.Logger) *handler {
	return &handler{engine: engine, logger: logger}
}

// RegisterStoreRoutes mounts the public search route on r.
func (h *handler) RegisterStoreRoutes(r chi.Router) {
	r.Get("/search", h.search)
}

// search handles GET /api/v1/store/search
//
// Query params:
//   q      – search query (required)
//   locale – BCP-47 locale (default: de-DE)
//   page   – page number (default: 1)
//   limit  – results per page (default: 25, max: 100)
//   type   – comma-separated entity types to filter, e.g. "product,category"
func (h *handler) search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if strings.TrimSpace(q) == "" {
		writeError(w, http.StatusBadRequest, "missing_query", "query parameter 'q' is required")
		return
	}

	locale := r.URL.Query().Get("locale")
	if locale == "" {
		// Fall back to Accept-Language header, then default locale.
		if al := r.Header.Get("Accept-Language"); al != "" {
			// Use the first tag only (e.g. "de-DE,de;q=0.9" → "de-DE").
			locale = strings.SplitN(al, ",", 2)[0]
			locale = strings.TrimSpace(strings.SplitN(locale, ";", 2)[0])
		}
	}

	var types []string
	if raw := r.URL.Query().Get("type"); raw != "" {
		for _, t := range strings.Split(raw, ",") {
			if t = strings.TrimSpace(t); t != "" {
				types = append(types, t)
			}
		}
	}

	req := SearchRequest{
		Query:  q,
		Locale: locale,
		Page:   parseIntQuery(r, "page", 1),
		Limit:  parseIntQuery(r, "limit", 25),
		Types:  types,
	}

	resp, err := h.engine.Search(r.Context(), req)
	if err != nil {
		h.logger.Error().Err(err).Str("query", q).Msg("search failed")
		writeError(w, http.StatusInternalServerError, "search_failed", "search request failed")
		return
	}

	pages := 0
	if resp.Limit > 0 {
		pages = (resp.Total + resp.Limit - 1) / resp.Limit
	}

	writeJSON(w, http.StatusOK, apiResponse{
		Data: resp.Results,
		Meta: &apiMeta{
			Total: resp.Total,
			Page:  resp.Page,
			Limit: resp.Limit,
			Pages: pages,
		},
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
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, detail string) {
	writeJSON(w, status, apiResponse{
		Errors: []apiError{{Code: code, Detail: detail}},
	})
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
