package order

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/pkg/sdk"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func newTestHandler(repo OrderRepository, paymentCheckFn PaymentMethodCheckFn, hooks *sdk.HookRegistry) *Handler {
	if hooks == nil {
		hooks = sdk.NewHookRegistry()
	}
	svc := NewService(repo, hooks, zerolog.Nop())
	return NewHandler(svc, nil, nil, paymentCheckFn, validator.New(), zerolog.Nop())
}

func checkoutBody(t *testing.T, pmID *uuid.UUID) *bytes.Buffer {
	t.Helper()
	req := CheckoutRequest{
		Currency:        "EUR",
		BillingAddress:  map[string]interface{}{"city": "Berlin"},
		ShippingAddress: map[string]interface{}{"city": "Berlin"},
		PaymentMethodID: pmID,
		Items: []CheckoutItemRequest{
			{
				SKU:            "TEST-001",
				Name:           "Test Item",
				Quantity:       1,
				UnitPriceNet:   1000,
				UnitPriceGross: 1190,
				TaxRate:        1900,
			},
		},
	}
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal checkout request: %v", err)
	}
	return bytes.NewBuffer(body)
}

func doCheckout(h *Handler, body *bytes.Buffer) *httptest.ResponseRecorder {
	r := chi.NewRouter()
	r.Post("/checkout", h.storeCheckout)
	req := httptest.NewRequest(http.MethodPost, "/checkout", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestStoreCheckout_NoActivePaymentMethods_OK(t *testing.T) {
	repo := &mockOrderRepo{}
	checkFn := PaymentMethodCheckFn(func(_ context.Context, _ *uuid.UUID) (bool, bool, error) {
		return false, false, nil // no active methods
	})
	h := newTestHandler(repo, checkFn, nil)

	rr := doCheckout(h, checkoutBody(t, nil))
	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestStoreCheckout_ActiveMethods_NoID_422(t *testing.T) {
	repo := &mockOrderRepo{}
	checkFn := PaymentMethodCheckFn(func(_ context.Context, _ *uuid.UUID) (bool, bool, error) {
		return true, false, nil // active methods exist, no ID given
	})
	h := newTestHandler(repo, checkFn, nil)

	rr := doCheckout(h, checkoutBody(t, nil))
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp apiResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp.Errors) == 0 || resp.Errors[0].Code != "payment_method_required" {
		t.Errorf("expected payment_method_required error, got %+v", resp.Errors)
	}
}

func TestStoreCheckout_ActiveMethods_InvalidID_422(t *testing.T) {
	repo := &mockOrderRepo{}
	checkFn := PaymentMethodCheckFn(func(_ context.Context, _ *uuid.UUID) (bool, bool, error) {
		return true, false, nil // active methods exist, ID is invalid
	})
	h := newTestHandler(repo, checkFn, nil)

	badID := uuid.New()
	rr := doCheckout(h, checkoutBody(t, &badID))
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp apiResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp.Errors) == 0 || resp.Errors[0].Code != "invalid_payment_method" {
		t.Errorf("expected invalid_payment_method error, got %+v", resp.Errors)
	}
}

func TestStoreCheckout_ActiveMethods_ValidID_OK(t *testing.T) {
	repo := &mockOrderRepo{}
	validID := uuid.New()
	checkFn := PaymentMethodCheckFn(func(_ context.Context, id *uuid.UUID) (bool, bool, error) {
		return true, id != nil && *id == validID, nil
	})
	h := newTestHandler(repo, checkFn, nil)

	rr := doCheckout(h, checkoutBody(t, &validID))
	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestStoreCheckout_NilPaymentCheckFn_OK(t *testing.T) {
	repo := &mockOrderRepo{}
	h := newTestHandler(repo, nil, nil)

	rr := doCheckout(h, checkoutBody(t, nil))
	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestStoreCheckout_BeforeHookRejects_422(t *testing.T) {
	repo := &mockOrderRepo{}
	hooks := sdk.NewHookRegistry()
	hooks.On(sdk.HookBeforeCheckout, func(_ context.Context, _ *sdk.HookEvent) error {
		return errors.New("payment plugin rejected checkout")
	})
	h := newTestHandler(repo, nil, hooks)

	rr := doCheckout(h, checkoutBody(t, nil))
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp apiResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp.Errors) == 0 || resp.Errors[0].Code != "checkout_rejected" {
		t.Errorf("expected checkout_rejected error, got %+v", resp.Errors)
	}
}

func TestStoreCheckout_AfterHookError_OrderStillCreated(t *testing.T) {
	var created bool
	repo := &mockOrderRepo{
		create: func(_ context.Context, _ *Order) error {
			created = true
			return nil
		},
	}
	hooks := sdk.NewHookRegistry()
	hooks.On(sdk.HookAfterCheckout, func(_ context.Context, _ *sdk.HookEvent) error {
		return errors.New("after hook failed")
	})
	h := newTestHandler(repo, nil, hooks)

	rr := doCheckout(h, checkoutBody(t, nil))
	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	if !created {
		t.Error("expected order to be created despite after-hook error")
	}
}
