package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type Handler struct {
	pool       *pgxpool.Pool
	jwtManager *JWTManager
	logger     zerolog.Logger
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

func NewHandler(pool *pgxpool.Pool, jwtManager *JWTManager, logger zerolog.Logger) *Handler {
	return &Handler{pool: pool, jwtManager: jwtManager, logger: logger}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/api/v1/auth/login", h.handleLogin)
	r.Post("/api/v1/auth/refresh", h.handleRefresh)
	r.Post("/api/v1/auth/logout", h.handleLogout)
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"errors": []map[string]string{{"code": "invalid_credentials", "detail": "invalid email or password"}},
		})
		return
	}

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

func (h *Handler) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": TokenResponse{
			AccessToken:  accessToken,
			RefreshToken: newRefreshToken,
			ExpiresIn:    900,
			TokenType:    "Bearer",
		},
	})
}

func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	// For stateless JWT, logout is handled client-side by discarding the token.
	// If token blacklisting is needed, it can be added here.
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
