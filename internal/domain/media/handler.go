package media

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

const maxUploadSize = 32 << 20 // 32 MiB

type handler struct {
	svc      MediaService
	validate *validator.Validate
	logger   zerolog.Logger
}

// NewHandler creates a new media HTTP handler.
func NewHandler(svc MediaService, logger zerolog.Logger) *handler {
	return &handler{
		svc:      svc,
		validate: validator.New(),
		logger:   logger,
	}
}

// RegisterAdminRoutes mounts the admin-only media routes on r.
func (h *handler) RegisterAdminRoutes(r chi.Router) {
	r.Route("/media", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.upload)
		r.Get("/{id}", h.getByID)
		r.Delete("/{id}", h.delete)
	})
}

// --- handlers ---

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	filter := MediaFilter{
		Page:     parseIntQuery(r, "page", 1),
		Limit:    parseIntQuery(r, "limit", 20),
		MimeType: r.URL.Query().Get("mime_type"),
	}

	items, total, err := h.svc.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	pages := 0
	if filter.Limit > 0 {
		pages = (total + filter.Limit - 1) / filter.Limit
	}
	writeJSON(w, http.StatusOK, apiResponse{
		Data: items,
		Meta: &apiMeta{Total: total, Page: filter.Page, Limit: filter.Limit, Pages: pages},
	})
}

func (h *handler) upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeError(w, http.StatusBadRequest, "parse_failed", "failed to parse multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing_file", "file field is required")
		return
	}
	defer file.Close()

	altText := r.FormValue("alt_text")
	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	m, err := h.svc.Upload(r.Context(), header.Filename, mimeType, altText, header.Size, file)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "upload_failed", err.Error())
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

	m, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "media not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
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

	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "media not found")
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
