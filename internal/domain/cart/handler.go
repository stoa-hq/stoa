package cart

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Handler holds the HTTP handlers for the cart domain.
type Handler struct {
	service *CartService
	logger  zerolog.Logger
}

// NewHandler creates a new cart HTTP Handler.
func NewHandler(service *CartService, logger zerolog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// RegisterStoreRoutes mounts the store-facing cart routes onto the given router.
//
//	POST   /cart                       – create a cart
//	GET    /cart/:id                   – get a cart by ID
//	POST   /cart/:id/items             – add an item to the cart
//	PUT    /cart/:id/items/:itemId     – update item quantity
//	DELETE /cart/:id/items/:itemId     – remove item from cart
func (h *Handler) RegisterStoreRoutes(r chi.Router) {
	r.Post("/cart", h.handleCreateCart)
	r.Get("/cart/{id}", h.handleGetCart)
	r.Post("/cart/{id}/items", h.handleAddItem)
	r.Put("/cart/{id}/items/{itemId}", h.handleUpdateItem)
	r.Delete("/cart/{id}/items/{itemId}", h.handleRemoveItem)
}

// handleCreateCart creates a new cart.
// POST /cart
func (h *Handler) handleCreateCart(w http.ResponseWriter, r *http.Request) {
	var req CreateCartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "request body is not valid JSON")
		return
	}

	c, err := h.service.CreateCart(r.Context(), req.Currency, req.SessionID, nil, req.ExpiresAt)
	if err != nil {
		h.logger.Error().Err(err).Msg("cart handler: create cart")
		h.writeError(w, http.StatusInternalServerError, "internal_error", "failed to create cart")
		return
	}

	writeJSON(w, http.StatusCreated, apiResponse{Data: toCartResponse(c)})
}

// handleGetCart returns a cart by its ID.
// GET /cart/:id
func (h *Handler) handleGetCart(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUID(w, chi.URLParam(r, "id"), "id")
	if !ok {
		return
	}

	c, err := h.service.GetCart(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrCartNotFound) {
			h.writeError(w, http.StatusNotFound, "not_found", "cart not found")
			return
		}
		h.logger.Error().Err(err).Str("cart_id", id.String()).Msg("cart handler: get cart")
		h.writeError(w, http.StatusInternalServerError, "internal_error", "failed to retrieve cart")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: toCartResponse(c)})
}

// handleAddItem adds a line item to a cart.
// POST /cart/:id/items
func (h *Handler) handleAddItem(w http.ResponseWriter, r *http.Request) {
	cartID, ok := h.parseUUID(w, chi.URLParam(r, "id"), "id")
	if !ok {
		return
	}

	var req AddItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "request body is not valid JSON")
		return
	}

	if req.ProductID == uuid.Nil {
		h.writeError(w, http.StatusBadRequest, "validation_error", "product_id is required")
		return
	}
	if req.Quantity <= 0 {
		h.writeError(w, http.StatusBadRequest, "validation_error", "quantity must be greater than zero")
		return
	}

	_, err := h.service.AddItem(r.Context(), cartID, req.ProductID, req.VariantID, req.Quantity, req.CustomFields)
	if err != nil {
		switch {
		case errors.Is(err, ErrCartNotFound):
			h.writeError(w, http.StatusNotFound, "not_found", "cart not found")
		case errors.Is(err, ErrInsufficientStock):
			h.writeError(w, http.StatusUnprocessableEntity, "insufficient_stock", "requested quantity exceeds available stock")
		default:
			h.logger.Error().Err(err).Str("cart_id", cartID.String()).Msg("cart handler: add item")
			h.writeError(w, http.StatusInternalServerError, "internal_error", "failed to add item to cart")
		}
		return
	}

	cart, err := h.service.GetCart(r.Context(), cartID)
	if err != nil {
		h.logger.Error().Err(err).Str("cart_id", cartID.String()).Msg("cart handler: get cart after add item")
		h.writeError(w, http.StatusInternalServerError, "internal_error", "failed to retrieve cart")
		return
	}

	writeJSON(w, http.StatusCreated, apiResponse{Data: toCartResponse(cart)})
}

// handleUpdateItem updates the quantity of a cart line item.
// PUT /cart/:id/items/:itemId
func (h *Handler) handleUpdateItem(w http.ResponseWriter, r *http.Request) {
	cartID, ok := h.parseUUID(w, chi.URLParam(r, "id"), "id")
	if !ok {
		return
	}

	itemID, ok := h.parseUUID(w, chi.URLParam(r, "itemId"), "itemId")
	if !ok {
		return
	}

	var req UpdateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "request body is not valid JSON")
		return
	}

	if req.Quantity <= 0 {
		h.writeError(w, http.StatusBadRequest, "validation_error", "quantity must be greater than zero")
		return
	}

	if err := h.service.UpdateItemQuantity(r.Context(), itemID, req.Quantity); err != nil {
		switch {
		case errors.Is(err, ErrItemNotFound):
			h.writeError(w, http.StatusNotFound, "not_found", "cart item not found")
		case errors.Is(err, ErrInsufficientStock):
			h.writeError(w, http.StatusUnprocessableEntity, "insufficient_stock", "requested quantity exceeds available stock")
		default:
			h.logger.Error().Err(err).Str("item_id", itemID.String()).Msg("cart handler: update item")
			h.writeError(w, http.StatusInternalServerError, "internal_error", "failed to update cart item")
		}
		return
	}

	cart, err := h.service.GetCart(r.Context(), cartID)
	if err != nil {
		h.logger.Error().Err(err).Str("cart_id", cartID.String()).Msg("cart handler: get cart after update item")
		h.writeError(w, http.StatusInternalServerError, "internal_error", "failed to retrieve cart")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: toCartResponse(cart)})
}

// handleRemoveItem removes a line item from a cart.
// DELETE /cart/:id/items/:itemId
func (h *Handler) handleRemoveItem(w http.ResponseWriter, r *http.Request) {
	cartID, ok := h.parseUUID(w, chi.URLParam(r, "id"), "id")
	if !ok {
		return
	}

	itemID, ok := h.parseUUID(w, chi.URLParam(r, "itemId"), "itemId")
	if !ok {
		return
	}

	if err := h.service.RemoveItem(r.Context(), itemID); err != nil {
		switch {
		case errors.Is(err, ErrItemNotFound):
			h.writeError(w, http.StatusNotFound, "not_found", "cart item not found")
		default:
			h.logger.Error().Err(err).Str("item_id", itemID.String()).Msg("cart handler: remove item")
			h.writeError(w, http.StatusInternalServerError, "internal_error", "failed to remove cart item")
		}
		return
	}

	cart, err := h.service.GetCart(r.Context(), cartID)
	if err != nil {
		h.logger.Error().Err(err).Str("cart_id", cartID.String()).Msg("cart handler: get cart after remove item")
		h.writeError(w, http.StatusInternalServerError, "internal_error", "failed to retrieve cart")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: toCartResponse(cart)})
}

// --- helpers ----------------------------------------------------------------

type apiResponse struct {
	Data   interface{}  `json:"data,omitempty"`
	Errors []apiError   `json:"errors,omitempty"`
}

type apiError struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
	Field  string `json:"field,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func (h *Handler) writeError(w http.ResponseWriter, status int, code, detail string) {
	writeJSON(w, status, apiResponse{
		Errors: []apiError{{Code: code, Detail: detail}},
	})
}

// parseUUID parses a URL parameter as a UUID, writing a 400 error and
// returning false on failure.
func (h *Handler) parseUUID(w http.ResponseWriter, raw, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(raw)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_uuid", param+" must be a valid UUID")
		return uuid.Nil, false
	}
	return id, true
}
