package order

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/pkg/sdk"
)

// ShippingCostFn resolves the gross shipping cost for a given shipping method ID.
// Returns 0 if the ID is unknown or an error occurs.
type ShippingCostFn func(ctx context.Context, id uuid.UUID) (int, error)

// ProductTaxRateFn resolves the integer basis-point tax rate for a given product ID.
// Returns an error if the product has no tax rule or the lookup fails.
type ProductTaxRateFn func(ctx context.Context, productID uuid.UUID) (int, error)

// PaymentMethodCheckFn checks whether the given payment method ID is valid.
// It returns whether any active payment methods are configured, whether the
// specific ID (if non-nil) references a valid active method, and any error.
type PaymentMethodCheckFn func(ctx context.Context, id *uuid.UUID) (hasActiveMethods bool, methodIsValid bool, err error)

func calcNetFromGross(gross, rate int) int {
	return int(math.Round(float64(gross) * 10000 / float64(10000+rate)))
}

// ---------------------------------------------------------------------------
// API envelope types (local to handler)
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

// Handler handles HTTP requests for the order domain.
type Handler struct {
	service              *Service
	shippingCostFn       ShippingCostFn
	productTaxRateFn     ProductTaxRateFn
	paymentMethodCheckFn PaymentMethodCheckFn
	validator            *validator.Validate
	logger               zerolog.Logger
}

// NewHandler creates a new order Handler.
// shippingCostFn may be nil; if non-nil it is called during checkout to apply shipping costs.
// productTaxRateFn may be nil; if non-nil it is used to look up tax rates per product during checkout.
// paymentMethodCheckFn may be nil; if non-nil it validates payment method selection during checkout.
func NewHandler(service *Service, shippingCostFn ShippingCostFn, productTaxRateFn ProductTaxRateFn, paymentMethodCheckFn PaymentMethodCheckFn, validate *validator.Validate, logger zerolog.Logger) *Handler {
	return &Handler{
		service:              service,
		shippingCostFn:       shippingCostFn,
		productTaxRateFn:     productTaxRateFn,
		paymentMethodCheckFn: paymentMethodCheckFn,
		validator:            validate,
		logger:               logger,
	}
}

// ---------------------------------------------------------------------------
// Route registration
// ---------------------------------------------------------------------------

// RegisterAdminRoutes mounts admin order management endpoints.
// Expected prefix: /orders
func (h *Handler) RegisterAdminRoutes(r chi.Router) {
	r.Get("/orders", h.adminList)
	r.Get("/orders/{id}", h.adminGetByID)
	r.Put("/orders/{id}/status", h.adminUpdateStatus)
}

// RegisterStoreRoutes mounts the customer-facing order endpoints.
// Expected prefix: (root of store router)
func (h *Handler) RegisterStoreRoutes(r chi.Router) {
	r.Post("/checkout", h.storeCheckout)
	r.Get("/account/orders", h.storeListOrders)
}

// ---------------------------------------------------------------------------
// Admin handlers
// ---------------------------------------------------------------------------

// adminList handles GET /orders
// Query params: page, limit, sort, order, status, customer_id
func (h *Handler) adminList(w http.ResponseWriter, r *http.Request) {
	filter, page, limit := h.parseListFilter(r)

	orders, total, err := h.service.List(r.Context(), filter)
	if err != nil {
		h.serverError(w, r, err)
		return
	}

	pages := 0
	if limit > 0 {
		pages = int(math.Ceil(float64(total) / float64(limit)))
	}

	items := make([]OrderResponse, len(orders))
	for i := range orders {
		items[i] = ToResponse(&orders[i])
	}

	h.writeJSON(w, http.StatusOK, apiResponse{
		Data: items,
		Meta: &apiMeta{
			Total: total,
			Page:  page,
			Limit: limit,
			Pages: pages,
		},
	})
}

// adminGetByID handles GET /orders/{id}
func (h *Handler) adminGetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUID(w, r, "id")
	if !ok {
		return
	}

	o, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if h.isNotFound(err) {
			h.notFound(w, "order not found")
			return
		}
		h.serverError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusOK, apiResponse{Data: ToResponse(o)})
}

// adminUpdateStatus handles PUT /orders/{id}/status
func (h *Handler) adminUpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUID(w, r, "id")
	if !ok {
		return
	}

	var req UpdateStatusRequest
	if !h.decodeJSON(w, r, &req) {
		return
	}
	if !h.validate(w, &req) {
		return
	}

	if err := h.service.UpdateStatus(r.Context(), id, req.Status, req.Comment); err != nil {
		if h.isNotFound(err) {
			h.notFound(w, "order not found")
			return
		}
		// Invalid transition is a domain-level validation error.
		if strings.Contains(err.Error(), "invalid status transition") ||
			strings.Contains(err.Error(), "unknown order status") {
			h.writeError(w, http.StatusUnprocessableEntity, "invalid_transition", err.Error(), "status")
			return
		}
		h.serverError(w, r, err)
		return
	}

	// Return the updated order so the caller gets the new status immediately.
	updated, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		// Non-fatal: return 204 if we can't re-fetch.
		w.WriteHeader(http.StatusNoContent)
		return
	}

	h.writeJSON(w, http.StatusOK, apiResponse{Data: ToResponse(updated)})
}

// ---------------------------------------------------------------------------
// Store handlers
// ---------------------------------------------------------------------------

// storeCheckout handles POST /checkout
func (h *Handler) storeCheckout(w http.ResponseWriter, r *http.Request) {
	var req CheckoutRequest
	if !h.decodeJSON(w, r, &req) {
		return
	}
	if !h.validate(w, &req) {
		return
	}

	// Validate payment method selection against active payment methods.
	if h.paymentMethodCheckFn != nil {
		hasActive, methodValid, err := h.paymentMethodCheckFn(r.Context(), req.PaymentMethodID)
		if err != nil {
			h.serverError(w, r, err)
			return
		}
		if hasActive && req.PaymentMethodID == nil {
			h.writeError(w, http.StatusUnprocessableEntity, "payment_method_required",
				"a payment method must be selected", "payment_method_id")
			return
		}
		if req.PaymentMethodID != nil && !methodValid {
			h.writeError(w, http.StatusUnprocessableEntity, "invalid_payment_method",
				"the selected payment method is not available", "payment_method_id")
			return
		}
	}

	// Attach the authenticated customer when present; guest checkout is
	// supported, so a missing customer ID is not an error.
	customerID := h.optionalCustomerID(r)

	o := FromCheckoutRequest(&req, customerID)

	// Generate a guest token for unauthenticated orders so that the
	// browser session can prove ownership without a JWT.
	if customerID == nil {
		o.GuestToken = uuid.New().String()
	}

	// Look up the shipping method price and apply it to the order.
	if req.ShippingMethodID != nil && h.shippingCostFn != nil {
		if cost, err := h.shippingCostFn(r.Context(), *req.ShippingMethodID); err == nil {
			o.ShippingCost = cost
			o.Total = o.SubtotalGross + o.ShippingCost
		}
	}

	// Server-side tax rate lookup: replace client-supplied tax_rate with the
	// authoritative rate from the product's tax rule and recalculate net prices.
	if h.productTaxRateFn != nil {
		for i := range o.Items {
			if o.Items[i].ProductID != nil {
				if rate, err := h.productTaxRateFn(r.Context(), *o.Items[i].ProductID); err == nil && rate > 0 {
					o.Items[i].TaxRate = rate
					if o.Items[i].UnitPriceGross > 0 && o.Items[i].UnitPriceNet == 0 {
						o.Items[i].UnitPriceNet = calcNetFromGross(o.Items[i].UnitPriceGross, rate)
						o.Items[i].TotalNet = o.Items[i].UnitPriceNet * o.Items[i].Quantity
					}
				}
			}
		}
		// Recalculate subtotal and tax total from item data.
		o.SubtotalNet, o.TaxTotal = 0, 0
		for _, item := range o.Items {
			o.SubtotalNet += item.TotalNet
			o.TaxTotal += item.TotalGross - item.TotalNet
		}
	}

	// Dispatch checkout before-hook — plugins can reject the checkout.
	if err := h.service.DispatchHook(r.Context(), sdk.HookBeforeCheckout, o); err != nil {
		h.writeError(w, http.StatusUnprocessableEntity, "checkout_rejected", err.Error(), "")
		return
	}

	if err := h.service.Create(r.Context(), o); err != nil {
		h.serverError(w, r, err)
		return
	}

	// Dispatch checkout after-hook — non-fatal, log errors.
	if err := h.service.DispatchHook(r.Context(), sdk.HookAfterCheckout, o); err != nil {
		h.logger.Warn().Err(err).Str("order_id", o.ID.String()).Msg("after_checkout hook returned error")
	}

	h.writeJSON(w, http.StatusCreated, apiResponse{Data: ToResponse(o)})
}

// storeListOrders handles GET /account/orders
// Returns all orders for the authenticated customer.
func (h *Handler) storeListOrders(w http.ResponseWriter, r *http.Request) {
	customerID, ok := h.customerIDFromContext(w, r)
	if !ok {
		return
	}

	orders, err := h.service.GetByCustomerID(r.Context(), customerID)
	if err != nil {
		h.serverError(w, r, err)
		return
	}

	items := make([]OrderResponse, len(orders))
	for i := range orders {
		items[i] = ToResponse(&orders[i])
	}

	h.writeJSON(w, http.StatusOK, apiResponse{
		Data: items,
		Meta: &apiMeta{
			Total: len(items),
			Page:  1,
			Limit: len(items),
			Pages: 1,
		},
	})
}

// ---------------------------------------------------------------------------
// Parsing helpers
// ---------------------------------------------------------------------------

// parseListFilter builds an OrderFilter from URL query parameters.
func (h *Handler) parseListFilter(r *http.Request) (OrderFilter, int, int) {
	q := r.URL.Query()

	page := 1
	if v := q.Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}

	limit := 25
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	filter := OrderFilter{
		Page:   page,
		Limit:  limit,
		Status: q.Get("status"),
		Search: q.Get("search"),
		Sort:   q.Get("sort"),
		Order:  q.Get("order"),
	}

	if v := q.Get("customer_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			filter.CustomerID = &id
		}
	}

	return filter, page, limit
}

// customerIDFromContext extracts the authenticated customer's UUID from the
// request context. It writes a 401 and returns false when the ID is absent.
func (h *Handler) customerIDFromContext(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	type contextKeyType string
	const ctxKeyUserID contextKeyType = "user_id"

	id, ok := r.Context().Value(ctxKeyUserID).(uuid.UUID)
	if !ok || id == uuid.Nil {
		h.writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required", "")
		return uuid.Nil, false
	}
	return id, true
}

// optionalCustomerID extracts the customer UUID from context without failing.
// Returns nil when the user is unauthenticated (guest checkout).
func (h *Handler) optionalCustomerID(r *http.Request) *uuid.UUID {
	type contextKeyType string
	const ctxKeyUserID contextKeyType = "user_id"

	id, ok := r.Context().Value(ctxKeyUserID).(uuid.UUID)
	if !ok || id == uuid.Nil {
		return nil
	}
	cp := id
	return &cp
}

// parseLocale extracts the primary locale tag from the Accept-Language header,
// defaulting to "en" when the header is absent or malformed.
func parseLocale(r *http.Request) string {
	al := r.Header.Get("Accept-Language")
	if al == "" {
		return "en"
	}
	parts := strings.SplitN(al, ",", 2)
	lang := strings.TrimSpace(parts[0])
	if idx := strings.Index(lang, ";"); idx != -1 {
		lang = lang[:idx]
	}
	lang = strings.TrimSpace(lang)
	if lang == "" {
		return "en"
	}
	return lang
}

// isNotFound reports whether an error indicates a missing resource.
// The order domain does not define a sentinel ErrNotFound; it wraps the
// pgx.ErrNoRows message in a descriptive string instead.
func (h *Handler) isNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "not found") || errors.Is(err, errOrderNotFound)
}

// errOrderNotFound is a local sentinel that handler code may check against.
// The postgres repository currently does not export a typed sentinel, so we
// rely on the string check in isNotFound above for now.
var errOrderNotFound = errors.New("order not found")

// parseUUID reads a chi URL parameter as a UUID, writing a 400 on failure.
func (h *Handler) parseUUID(w http.ResponseWriter, r *http.Request, param string) (uuid.UUID, bool) {
	raw := chi.URLParam(r, param)
	id, err := uuid.Parse(raw)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_param", param+" must be a valid UUID", param)
		return uuid.Nil, false
	}
	return id, true
}

// decodeJSON decodes the request body into dst, writing a 400 on failure.
func (h *Handler) decodeJSON(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_body", "request body is not valid JSON: "+err.Error(), "")
		return false
	}
	return true
}

// validate runs the go-playground validator and writes validation errors on failure.
func (h *Handler) validate(w http.ResponseWriter, v interface{}) bool {
	if err := h.validator.Struct(v); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			errs := make([]apiError, 0, len(ve))
			for _, fe := range ve {
				errs = append(errs, apiError{
					Code:   "validation_error",
					Detail: fe.Tag() + " constraint violated on " + fe.Field(),
					Field:  fe.Field(),
				})
			}
			h.writeJSON(w, http.StatusUnprocessableEntity, apiResponse{Errors: errs})
			return false
		}
		h.writeError(w, http.StatusUnprocessableEntity, "validation_error", err.Error(), "")
		return false
	}
	return true
}

// ---------------------------------------------------------------------------
// Response helpers
// ---------------------------------------------------------------------------

func (h *Handler) writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode JSON response")
	}
}

func (h *Handler) writeError(w http.ResponseWriter, status int, code, detail, field string) {
	h.writeJSON(w, status, apiResponse{
		Errors: []apiError{{Code: code, Detail: detail, Field: field}},
	})
}

func (h *Handler) notFound(w http.ResponseWriter, detail string) {
	h.writeError(w, http.StatusNotFound, "not_found", detail, "")
}

func (h *Handler) serverError(w http.ResponseWriter, r *http.Request, err error) {
	h.logger.Error().Err(err).Str("method", r.Method).Str("path", r.URL.Path).Msg("internal server error")
	h.writeError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred", "")
}

// Ensure parseLocale is used (available for store route locale work).
var _ = parseLocale
