package audit

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/epoxx-arch/stoa/internal/auth"
)

// Middleware returns a chi-compatible middleware that records every successful
// admin or store mutation to the audit log.
//
// It must be placed after the Authenticate middleware so that user information
// is available in the request context.
//
// Audit entries are written asynchronously (fire-and-forget) so that a slow or
// failing audit write never delays the HTTP response.
func Middleware(svc AuditService, logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Read-only methods are never audited.
			if r.Method == http.MethodGet ||
				r.Method == http.MethodHead ||
				r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			// Wrap the response writer to capture the HTTP status code that the
			// handler will write. We need this to decide whether to audit.
			crw := &captureResponseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(crw, r)

			// Only audit successful mutations (2xx).
			if crw.status < 200 || crw.status >= 300 {
				return
			}

			action, entityType, entityID := routeAuditInfo(r)
			if action == "" || entityType == "" {
				return
			}

			userID := auth.UserID(r.Context())
			userType := auth.UserType(r.Context())
			ip := clientIP(r)

			entry := &AuditLog{
				Action:     action,
				EntityType: entityType,
				UserType:   userType,
				IPAddress:  ip,
			}
			if userID != uuid.Nil {
				id := userID
				entry.UserID = &id
			}
			if entityID != uuid.Nil {
				id := entityID
				entry.EntityID = &id
			}

			// Fire-and-forget: audit errors must never fail the HTTP response.
			go func() {
				if err := svc.Log(context.Background(), entry); err != nil {
					logger.Warn().Err(err).
						Str("action", action).
						Str("entity_type", entityType).
						Msg("audit log write failed")
				}
			}()
		})
	}
}

// routeAuditInfo derives the audit action, entity type, and optional entity ID
// from the request URL path and HTTP method.
//
// Returns empty strings for paths that should not be audited (reads, unknowns).
func routeAuditInfo(r *http.Request) (action, entityType string, entityID uuid.UUID) {
	path := r.URL.Path

	switch {
	case strings.HasPrefix(path, "/api/v1/admin/"):
		return adminRouteInfo(r, strings.TrimPrefix(path, "/api/v1/admin/"))
	case strings.HasPrefix(path, "/api/v1/store/"):
		return storeRouteInfo(r, strings.TrimPrefix(path, "/api/v1/store/"))
	}
	return
}

// adminRouteInfo handles paths relative to /api/v1/admin/.
//
// Patterns:
//   POST   {entity}                → create
//   PUT    {entity}/{id}           → update
//   DELETE {entity}/{id}           → delete
//   POST   {entity}/{id}/{sub}     → {sub} (e.g. generate_variants, apply)
//   PUT    {entity}/{id}/{sub}     → update_{sub} (e.g. update_status)
//   POST   discounts/validate      → skipped (not an entity mutation)
func adminRouteInfo(r *http.Request, rel string) (action, entityType string, entityID uuid.UUID) {
	parts := strings.SplitN(rel, "/", 3)

	entityType = adminEntityType(parts[0])
	if entityType == "" {
		return
	}

	switch len(parts) {
	case 1:
		// e.g. POST /products
		if r.Method == http.MethodPost {
			action = "create"
		}

	case 2:
		// e.g. PUT /products/{id}, DELETE /products/{id}
		entityID, _ = uuid.Parse(parts[1])
		switch r.Method {
		case http.MethodPut, http.MethodPatch:
			action = "update"
		case http.MethodDelete:
			action = "delete"
		}

	case 3:
		// e.g. POST /products/{id}/variants
		//      PUT  /orders/{id}/status
		//      POST /discounts/validate  (skip)
		entityID, _ = uuid.Parse(parts[1])
		sub := parts[2]

		// /discounts/validate is a query, not a mutation.
		if entityType == "discount" && sub == "validate" {
			entityType = ""
			return
		}

		switch r.Method {
		case http.MethodPost:
			action = strings.ReplaceAll(sub, "-", "_")
		case http.MethodPut, http.MethodPatch:
			action = "update_" + strings.ReplaceAll(sub, "-", "_")
		}
	}
	return
}

// storeRouteInfo handles a curated list of store mutations worth auditing.
func storeRouteInfo(r *http.Request, rel string) (action, entityType string, entityID uuid.UUID) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch {
		return
	}
	switch rel {
	case "checkout":
		action, entityType = "checkout", "order"
	case "register":
		action, entityType = "register", "customer"
	case "account":
		if r.Method == http.MethodPut || r.Method == http.MethodPatch {
			action, entityType = "update", "customer"
		}
	}
	return
}

// adminEntityType maps a URL path segment to a canonical entity type name.
func adminEntityType(segment string) string {
	switch segment {
	case "products":
		return "product"
	case "categories":
		return "category"
	case "customers":
		return "customer"
	case "orders":
		return "order"
	case "tax-rules":
		return "tax_rule"
	case "shipping-methods":
		return "shipping_method"
	case "payment-methods":
		return "payment_method"
	case "discounts":
		return "discount"
	case "tags":
		return "tag"
	case "media":
		return "media"
	default:
		return ""
	}
}

// captureResponseWriter wraps http.ResponseWriter to capture the status code
// written by the handler.
type captureResponseWriter struct {
	http.ResponseWriter
	status int
}

func (c *captureResponseWriter) WriteHeader(code int) {
	c.status = code
	c.ResponseWriter.WriteHeader(code)
}

// Write captures an implicit 200 OK when the handler writes a body without
// calling WriteHeader first.
func (c *captureResponseWriter) Write(b []byte) (int, error) {
	if c.status == 0 {
		c.status = http.StatusOK
	}
	return c.ResponseWriter.Write(b)
}

// clientIP extracts the real client IP from the request, checking the
// X-Real-IP and X-Forwarded-For headers before falling back to RemoteAddr.
func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.TrimSpace(strings.SplitN(fwd, ",", 2)[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
