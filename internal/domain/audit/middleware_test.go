package audit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestRouteAuditInfo(t *testing.T) {
	id := uuid.New()

	tests := []struct {
		method         string
		path           string
		wantAction     string
		wantEntityType string
		wantID         bool
	}{
		// Admin create
		{"POST", "/api/v1/admin/products", "create", "product", false},
		{"POST", "/api/v1/admin/categories", "create", "category", false},
		{"POST", "/api/v1/admin/customers", "create", "customer", false},
		{"POST", "/api/v1/admin/tax-rules", "create", "tax_rule", false},
		{"POST", "/api/v1/admin/shipping-methods", "create", "shipping_method", false},
		{"POST", "/api/v1/admin/payment-methods", "create", "payment_method", false},
		{"POST", "/api/v1/admin/discounts", "create", "discount", false},
		{"POST", "/api/v1/admin/tags", "create", "tag", false},
		{"POST", "/api/v1/admin/media", "create", "media", false},

		// Admin update
		{"PUT", "/api/v1/admin/products/" + id.String(), "update", "product", true},
		{"PUT", "/api/v1/admin/categories/" + id.String(), "update", "category", true},
		{"PUT", "/api/v1/admin/customers/" + id.String(), "update", "customer", true},

		// Admin delete
		{"DELETE", "/api/v1/admin/products/" + id.String(), "delete", "product", true},
		{"DELETE", "/api/v1/admin/media/" + id.String(), "delete", "media", true},

		// Admin sub-actions
		{"POST", "/api/v1/admin/products/" + id.String() + "/variants", "variants", "product", true},
		{"PUT", "/api/v1/admin/orders/" + id.String() + "/status", "update_status", "order", true},
		{"POST", "/api/v1/admin/discounts/" + id.String() + "/apply", "apply", "discount", true},

		// Admin sub-actions with nested sub-resource ID (e.g. PUT /products/{id}/variants/{variantId})
		{"PUT", "/api/v1/admin/products/" + id.String() + "/variants/" + uuid.New().String(), "update_variants", "product", true},

		// Admin: discounts/validate should be skipped
		{"POST", "/api/v1/admin/discounts/validate", "", "discount", false},

		// Store
		{"POST", "/api/v1/store/checkout", "checkout", "order", false},
		{"POST", "/api/v1/store/register", "register", "customer", false},
		{"PUT", "/api/v1/store/account", "update", "customer", false},

		// GET should return nothing (caller skips, but testing the function)
		{"GET", "/api/v1/admin/products", "", "product", false},

		// Unknown path
		{"POST", "/api/v1/admin/unknown-entity", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, tt.path, nil)
			action, entityType, entityID := routeAuditInfo(r)

			if action != tt.wantAction {
				t.Errorf("action: got %q, want %q", action, tt.wantAction)
			}
			if entityType != tt.wantEntityType {
				t.Errorf("entityType: got %q, want %q", entityType, tt.wantEntityType)
			}
			gotID := entityID != uuid.Nil
			if gotID != tt.wantID {
				t.Errorf("entityID presence: got %v, want %v (id=%s)", gotID, tt.wantID, entityID)
			}
			if tt.wantID && gotID && entityID != id {
				t.Errorf("entityID value: got %s, want %s", entityID, id)
			}
		})
	}
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*http.Request)
		wantIP     string
	}{
		{
			name: "X-Real-IP header",
			setup: func(r *http.Request) {
				r.Header.Set("X-Real-IP", "1.2.3.4")
			},
			wantIP: "1.2.3.4",
		},
		{
			name: "X-Forwarded-For single",
			setup: func(r *http.Request) {
				r.Header.Set("X-Forwarded-For", "5.6.7.8")
			},
			wantIP: "5.6.7.8",
		},
		{
			name: "X-Forwarded-For chain",
			setup: func(r *http.Request) {
				r.Header.Set("X-Forwarded-For", "9.10.11.12, 13.14.15.16")
			},
			wantIP: "9.10.11.12",
		},
		{
			name:   "RemoteAddr fallback",
			setup:  func(r *http.Request) { r.RemoteAddr = "192.168.1.1:54321" },
			wantIP: "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			tt.setup(r)
			got := clientIP(r)
			if got != tt.wantIP {
				t.Errorf("clientIP: got %q, want %q", got, tt.wantIP)
			}
		})
	}
}
