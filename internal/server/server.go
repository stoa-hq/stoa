package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/internal/config"
	"github.com/stoa-hq/stoa/internal/database"
)

type Server struct {
	router chi.Router
	http   *http.Server
	db     *database.DB
	logger zerolog.Logger
	cfg    *config.Config
}

func New(cfg *config.Config, db *database.DB, logger zerolog.Logger) *Server {
	r := chi.NewRouter()

	s := &Server{
		router: r,
		db:     db,
		logger: logger,
		cfg:    cfg,
	}

	s.setupMiddleware()
	s.setupRoutes()

	s.http = &http.Server{
		Addr:           cfg.Addr(),
		Handler:        r,
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		MaxHeaderBytes: 1 << 20,
	}

	return s
}

func (s *Server) setupMiddleware() {
	r := s.router

	// 1. Recovery
	r.Use(chimw.Recoverer)

	// 2. Max request body size
	r.Use(MaxBytes(s.cfg.Server.MaxBodySize, s.cfg.Server.MaxUploadSize))

	// 3. Request ID
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := r.Header.Get("X-Request-ID")
			if reqID == "" {
				reqID = uuid.New().String()
			}
			ctx := context.WithValue(r.Context(), contextKeyRequestID, reqID)
			w.Header().Set("X-Request-ID", reqID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})

	// 4. Structured Logging
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			s.logger.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", ww.Status()).
				Dur("duration", time.Since(start)).
				Str("request_id", RequestID(r.Context())).
				Msg("request")
		})
	})

	// 5. CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   s.cfg.Server.CORS.AllowedOrigins,
		AllowedMethods:   s.cfg.Server.CORS.AllowedMethods,
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Request-ID", "Accept-Language"},
		ExposedHeaders:   []string{"Link", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// 6. Security Headers
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Content-Security-Policy", "default-src 'self'")
			next.ServeHTTP(w, r)
		})
	})

	// 7. Rate Limiter
	r.Use(httprate.LimitByIP(
		s.cfg.Security.RateLimit.RequestsPerMinute,
		time.Minute,
	))

	// 8. CSRF – Double Submit Cookie (exempt when Authorization header is present).
	// Plugin webhook paths are exempt because they authenticate via provider
	// signatures (e.g. Stripe HMAC), not cookies or CSRF tokens.
	r.Use(CSRF(s.cfg.Security.CSRF.Secure, "/plugins/"))

	// Content-Type enforcement for mutations
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
				ct := r.Header.Get("Content-Type")
				if r.ContentLength > 0 && ct != "" && !strings.HasPrefix(ct, "application/json") && !strings.HasPrefix(ct, "multipart/form-data") {
					writeJSON(w, http.StatusUnsupportedMediaType, APIResponse{
						Errors: []APIError{{Code: "unsupported_media_type", Detail: "Content-Type must be application/json or multipart/form-data"}},
					})
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	})
}

func (s *Server) setupRoutes() {
	r := s.router

	// Health check
	r.Get("/api/v1/health", s.handleHealth)
}

func (s *Server) Router() chi.Router {
	return s.router
}

func (s *Server) Start() error {
	s.logger.Info().Str("addr", s.http.Addr).Msg("starting HTTP server")
	if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("starting server: %w", err)
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info().Msg("shutting down HTTP server")
	return s.http.Shutdown(ctx)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	if err := s.db.Health(r.Context()); err != nil {
		status = "degraded"
	}
	writeJSON(w, http.StatusOK, APIResponse{
		Data: map[string]string{"status": status},
	})
}

// Context keys
type contextKey string

const contextKeyRequestID contextKey = "request_id"

func RequestID(ctx context.Context) string {
	if id, ok := ctx.Value(contextKeyRequestID).(string); ok {
		return id
	}
	return ""
}

// API Response types
type APIResponse struct {
	Data   interface{} `json:"data,omitempty"`
	Meta   *APIMeta    `json:"meta,omitempty"`
	Errors []APIError  `json:"errors,omitempty"`
}

type APIMeta struct {
	Total int `json:"total"`
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Pages int `json:"pages"`
}

type APIError struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
	Field  string `json:"field,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
