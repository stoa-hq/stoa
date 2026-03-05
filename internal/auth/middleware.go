package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type contextKeyType string

const (
	ctxKeyUserID      contextKeyType = "user_id"
	ctxKeyUserType    contextKeyType = "user_type"
	ctxKeyRole        contextKeyType = "role"
	ctxKeyPermissions contextKeyType = "permissions"
)

type Middleware struct {
	jwtManager    *JWTManager
	apiKeyManager *APIKeyManager
}

func NewMiddleware(jwtManager *JWTManager, apiKeyManager *APIKeyManager) *Middleware {
	return &Middleware{
		jwtManager:    jwtManager,
		apiKeyManager: apiKeyManager,
	}
}

// Authenticate extracts and validates the token from the request.
// It supports both JWT Bearer tokens and API keys.
func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeAuthError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 {
			writeAuthError(w, http.StatusUnauthorized, "invalid authorization header format")
			return
		}

		var ctx context.Context

		switch strings.ToLower(parts[0]) {
		case "bearer":
			claims, err := m.jwtManager.ValidateToken(parts[1])
			if err != nil {
				writeAuthError(w, http.StatusUnauthorized, "invalid token")
				return
			}
			if claims.Type != AccessToken {
				writeAuthError(w, http.StatusUnauthorized, "invalid token type")
				return
			}
			ctx = context.WithValue(r.Context(), ctxKeyUserID, claims.UserID)
			ctx = context.WithValue(ctx, ctxKeyUserType, claims.UserType)
			ctx = context.WithValue(ctx, ctxKeyRole, Role(claims.Role))

		case "apikey":
			if m.apiKeyManager == nil {
				writeAuthError(w, http.StatusUnauthorized, "API key authentication not available")
				return
			}
			apiKey, err := m.apiKeyManager.Validate(r.Context(), parts[1])
			if err != nil {
				writeAuthError(w, http.StatusUnauthorized, "invalid API key")
				return
			}
			ctx = context.WithValue(r.Context(), ctxKeyUserID, apiKey.ID)
			ctx = context.WithValue(ctx, ctxKeyUserType, "api_key")
			ctx = context.WithValue(ctx, ctxKeyRole, RoleAPIClient)
			ctx = context.WithValue(ctx, ctxKeyPermissions, apiKey.Permissions)

		default:
			writeAuthError(w, http.StatusUnauthorized, "unsupported authorization scheme")
			return
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole checks that the user has a specific role.
// API clients (RoleAPIClient) are allowed through if they have any permissions
// at all — fine-grained access control is enforced by RequirePermission.
func (m *Middleware) RequireRole(roles ...Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := UserRole(r.Context())
			for _, allowed := range roles {
				if role == allowed {
					next.ServeHTTP(w, r)
					return
				}
			}
			// Allow API clients with permissions to access admin routes.
			if role == RoleAPIClient {
				if perms := UserPermissions(r.Context()); len(perms) > 0 {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeAuthError(w, http.StatusForbidden, "insufficient permissions")
		})
	}
}

// RequirePermission checks that the user has a specific permission.
// For API clients, it checks context-stored per-key permissions.
func (m *Middleware) RequirePermission(perm Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := UserRole(r.Context())
			if !HasPermissionCtx(r.Context(), role, perm) {
				writeAuthError(w, http.StatusForbidden, "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// OptionalAuth tries to extract auth info but never blocks the request.
// If the token is absent, invalid, or expired the request continues without
// auth context so that public store routes remain accessible.
func (m *Middleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 {
			next.ServeHTTP(w, r)
			return
		}

		switch strings.ToLower(parts[0]) {
		case "bearer":
			claims, err := m.jwtManager.ValidateToken(parts[1])
			if err != nil || claims.Type != AccessToken {
				// Token present but invalid/expired – continue as anonymous.
				next.ServeHTTP(w, r)
				return
			}
			ctx := context.WithValue(r.Context(), ctxKeyUserID, claims.UserID)
			ctx = context.WithValue(ctx, ctxKeyUserType, claims.UserType)
			ctx = context.WithValue(ctx, ctxKeyRole, Role(claims.Role))
			next.ServeHTTP(w, r.WithContext(ctx))

		case "apikey":
			if m.apiKeyManager == nil {
				next.ServeHTTP(w, r)
				return
			}
			apiKey, err := m.apiKeyManager.Validate(r.Context(), parts[1])
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			ctx := context.WithValue(r.Context(), ctxKeyUserID, apiKey.ID)
			ctx = context.WithValue(ctx, ctxKeyUserType, "api_key")
			ctx = context.WithValue(ctx, ctxKeyRole, RoleAPIClient)
			ctx = context.WithValue(ctx, ctxKeyPermissions, apiKey.Permissions)
			next.ServeHTTP(w, r.WithContext(ctx))

		default:
			next.ServeHTTP(w, r)
		}
	})
}

// Context helpers
func UserID(ctx context.Context) uuid.UUID {
	if id, ok := ctx.Value(ctxKeyUserID).(uuid.UUID); ok {
		return id
	}
	return uuid.Nil
}

func UserType(ctx context.Context) string {
	if ut, ok := ctx.Value(ctxKeyUserType).(string); ok {
		return ut
	}
	return ""
}

func UserRole(ctx context.Context) Role {
	if role, ok := ctx.Value(ctxKeyRole).(Role); ok {
		return role
	}
	return ""
}

func UserPermissions(ctx context.Context) []Permission {
	if perms, ok := ctx.Value(ctxKeyPermissions).([]Permission); ok {
		return perms
	}
	return nil
}

func writeAuthError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"errors": []map[string]string{
			{"code": "unauthorized", "detail": detail},
		},
	})
}
