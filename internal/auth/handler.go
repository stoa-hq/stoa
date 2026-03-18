package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type Handler struct {
	pool          *pgxpool.Pool
	jwtManager    *JWTManager
	apiKeyManager *APIKeyManager
	bruteForce    *BruteForceTracker
	tokenStore    *RefreshTokenStore
	blacklist     *TokenBlacklist
	logger        zerolog.Logger
}

type AdminUser struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	FirstName    *string    `json:"first_name,omitempty"`
	LastName     *string    `json:"last_name,omitempty"`
	Role         string     `json:"role"`
	Active       bool       `json:"active"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func NewHandler(pool *pgxpool.Pool, jwtManager *JWTManager, apiKeyManager *APIKeyManager, bruteForce *BruteForceTracker, tokenStore *RefreshTokenStore, blacklist *TokenBlacklist, logger zerolog.Logger) *Handler {
	return &Handler{pool: pool, jwtManager: jwtManager, apiKeyManager: apiKeyManager, bruteForce: bruteForce, tokenStore: tokenStore, blacklist: blacklist, logger: logger}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/api/v1/auth/login", h.HandleLogin)
	r.Post("/api/v1/auth/refresh", h.HandleRefresh)
	r.Post("/api/v1/auth/logout", h.HandleLogout)
}

func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]interface{}{
				"errors": []map[string]string{{"code": "body_too_large", "detail": "request body exceeds size limit"}},
			})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"errors": []map[string]string{{"code": "invalid_request", "detail": "invalid request body"}},
		})
		return
	}

	if req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"errors": []map[string]string{{"code": "validation_error", "detail": "email and password are required"}},
		})
		return
	}

	if locked, retryAfter := h.bruteForce.IsLocked(req.Email); locked {
		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
		writeJSON(w, http.StatusTooManyRequests, map[string]interface{}{
			"errors": []map[string]string{{"code": "account_locked", "detail": "too many failed login attempts, please try again later"}},
		})
		return
	}

	// Look up admin user
	var user AdminUser
	err := h.pool.QueryRow(r.Context(),
		`SELECT id, email, password_hash, first_name, last_name, role, active
		 FROM admin_users WHERE email = $1`, req.Email).
		Scan(&user.ID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName, &user.Role, &user.Active)
	if err != nil {
		// Also check customers table
		err = h.pool.QueryRow(r.Context(),
			`SELECT id, email, password_hash, first_name, last_name, active
			 FROM customers WHERE email = $1`, req.Email).
			Scan(&user.ID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName, &user.Active)
		if err != nil {
			// User not found — run dummy hash comparison to prevent timing-based user enumeration
			VerifyPassword(req.Password, dummyHash)
			h.bruteForce.RecordFailure(req.Email)
			writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
				"errors": []map[string]string{{"code": "invalid_credentials", "detail": "invalid email or password"}},
			})
			return
		}
		user.Role = string(RoleCustomer)
	}

	if !user.Active {
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"errors": []map[string]string{{"code": "account_disabled", "detail": "account is disabled"}},
		})
		return
	}

	match, err := VerifyPassword(req.Password, user.PasswordHash)
	if err != nil || !match {
		h.bruteForce.RecordFailure(req.Email)
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"errors": []map[string]string{{"code": "invalid_credentials", "detail": "invalid email or password"}},
		})
		return
	}

	h.bruteForce.RecordSuccess(req.Email)

	userType := "admin"
	if user.Role == string(RoleCustomer) {
		userType = "customer"
	}

	accessToken, err := h.jwtManager.GenerateAccessToken(user.ID, user.Email, userType, user.Role)
	if err != nil {
		h.logger.Error().Err(err).Msg("generating access token")
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"errors": []map[string]string{{"code": "internal_error", "detail": "failed to generate token"}},
		})
		return
	}

	refreshToken, err := h.jwtManager.GenerateRefreshToken(user.ID, user.Email, userType, user.Role)
	if err != nil {
		h.logger.Error().Err(err).Msg("generating refresh token")
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"errors": []map[string]string{{"code": "internal_error", "detail": "failed to generate token"}},
		})
		return
	}

	// Persist refresh token for rotation tracking.
	refreshClaims, err := h.jwtManager.ValidateToken(refreshToken)
	if err != nil {
		h.logger.Error().Err(err).Msg("parsing refresh token claims")
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"errors": []map[string]string{{"code": "internal_error", "detail": "failed to generate token"}},
		})
		return
	}
	familyID := uuid.New()
	if err := h.tokenStore.Store(r.Context(), refreshClaims.ID, user.ID, familyID, refreshClaims.ExpiresAt.Time); err != nil {
		h.logger.Error().Err(err).Msg("storing refresh token")
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"errors": []map[string]string{{"code": "internal_error", "detail": "failed to generate token"}},
		})
		return
	}

	// Update last login
	h.updateLastLogin(r.Context(), user.ID, userType)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": TokenResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresIn:    900, // 15 minutes
			TokenType:    "Bearer",
		},
	})
}

func (h *Handler) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]interface{}{
				"errors": []map[string]string{{"code": "body_too_large", "detail": "request body exceeds size limit"}},
			})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"errors": []map[string]string{{"code": "invalid_request", "detail": "invalid request body"}},
		})
		return
	}

	claims, err := h.jwtManager.ValidateToken(req.RefreshToken)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"errors": []map[string]string{{"code": "invalid_token", "detail": "invalid refresh token"}},
		})
		return
	}

	if claims.Type != RefreshToken {
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"errors": []map[string]string{{"code": "invalid_token", "detail": "not a refresh token"}},
		})
		return
	}

	// Consume the old refresh token (marks it as revoked).
	// If the token was already used, reuse detection kicks in and
	// invalidates the entire token family.
	rec, err := h.tokenStore.Consume(r.Context(), claims.ID)
	if err != nil {
		code := "invalid_token"
		detail := "refresh token is invalid"
		if err == ErrTokenReuse {
			h.logger.Warn().Str("user_id", claims.UserID.String()).Msg("refresh token reuse detected — family revoked")
			detail = "token reuse detected, please login again"
		}
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"errors": []map[string]string{{"code": code, "detail": detail}},
		})
		return
	}

	accessToken, err := h.jwtManager.GenerateAccessToken(claims.UserID, claims.Email, claims.UserType, claims.Role)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"errors": []map[string]string{{"code": "internal_error", "detail": "failed to generate token"}},
		})
		return
	}

	newRefreshToken, err := h.jwtManager.GenerateRefreshToken(claims.UserID, claims.Email, claims.UserType, claims.Role)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"errors": []map[string]string{{"code": "internal_error", "detail": "failed to generate token"}},
		})
		return
	}

	// Persist the new refresh token in the same family.
	newClaims, _ := h.jwtManager.ValidateToken(newRefreshToken)
	if err := h.tokenStore.Store(r.Context(), newClaims.ID, claims.UserID, rec.FamilyID, newClaims.ExpiresAt.Time); err != nil {
		h.logger.Error().Err(err).Msg("storing rotated refresh token")
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"errors": []map[string]string{{"code": "internal_error", "detail": "failed to generate token"}},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": TokenResponse{
			AccessToken:  accessToken,
			RefreshToken: newRefreshToken,
			ExpiresIn:    900,
			TokenType:    "Bearer",
		},
	})
}

// RegisterAdminRoutes mounts API key management routes under the admin router.
func (h *Handler) RegisterAdminRoutes(r chi.Router) {
	r.Route("/api-keys", func(r chi.Router) {
		r.Post("/", h.handleCreateAPIKey)
		r.Get("/", h.handleListAPIKeys)
		r.Delete("/{id}", h.handleRevokeAPIKey)
	})
}

type CreateAPIKeyRequest struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

func (h *Handler) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]interface{}{
				"errors": []map[string]string{{"code": "body_too_large", "detail": "request body exceeds size limit"}},
			})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"errors": []map[string]string{{"code": "invalid_request", "detail": "invalid request body"}},
		})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"errors": []map[string]string{{"code": "validation_error", "detail": "name is required"}},
		})
		return
	}

	perms := make([]Permission, len(req.Permissions))
	for i, p := range req.Permissions {
		perms[i] = Permission(p)
	}

	rawKey, apiKey, err := h.apiKeyManager.Create(r.Context(), req.Name, perms)
	if err != nil {
		h.logger.Error().Err(err).Msg("creating API key")
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"errors": []map[string]string{{"code": "internal_error", "detail": "failed to create API key"}},
		})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"data": map[string]interface{}{
			"id":          apiKey.ID,
			"name":        apiKey.Name,
			"key":         rawKey,
			"permissions": apiKey.Permissions,
			"created_at":  apiKey.CreatedAt,
		},
	})
}

func (h *Handler) handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := h.apiKeyManager.List(r.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("listing API keys")
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"errors": []map[string]string{{"code": "internal_error", "detail": "failed to list API keys"}},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": keys,
	})
}

func (h *Handler) handleRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"errors": []map[string]string{{"code": "invalid_request", "detail": "invalid API key ID"}},
		})
		return
	}

	if err := h.apiKeyManager.Revoke(r.Context(), id); err != nil {
		h.logger.Error().Err(err).Msg("revoking API key")
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"errors": []map[string]string{{"code": "internal_error", "detail": "failed to revoke API key"}},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": map[string]string{"message": "API key revoked"},
	})
}

func (h *Handler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	// Blacklist the current access token so it cannot be reused.
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			if claims, err := h.jwtManager.ValidateToken(parts[1]); err == nil && claims.Type == AccessToken {
				h.blacklist.Add(claims.ID, claims.ExpiresAt.Time)
			}
		}
	}

	// Accept an optional refresh_token in the body to identify the user
	// even without an Authorization header.
	var req RefreshRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.RefreshToken != "" {
		claims, err := h.jwtManager.ValidateToken(req.RefreshToken)
		if err == nil && claims.Type == RefreshToken {
			_ = h.tokenStore.RevokeAllForUser(r.Context(), claims.UserID)
		}
	} else if uid := UserID(r.Context()); uid != uuid.Nil {
		_ = h.tokenStore.RevokeAllForUser(r.Context(), uid)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": map[string]string{"message": "logged out successfully"},
	})
}

func (h *Handler) updateLastLogin(ctx context.Context, userID uuid.UUID, userType string) {
	if userType == "admin" {
		_, _ = h.pool.Exec(ctx, `UPDATE admin_users SET last_login_at = $1 WHERE id = $2`, time.Now(), userID)
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
