package cart

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/pkg/sdk"
)

// stockChecker is a narrow interface that the service uses to verify product
// stock without importing the full product domain package.
type stockChecker interface {
	// StockAvailable returns true when the given product (or variant, if non-nil)
	// has at least the requested quantity in stock.
	StockAvailable(ctx context.Context, productID uuid.UUID, variantID *uuid.UUID, quantity int) (bool, error)
}

// CartService encapsulates business logic for the cart domain.
type CartService struct {
	repo   CartRepository
	stock  stockChecker
	hooks  *sdk.HookRegistry
	logger zerolog.Logger
}

// NewCartService creates a new CartService.
// stockChecker may be nil; when nil, stock validation is skipped.
func NewCartService(
	repo CartRepository,
	stock stockChecker,
	hooks *sdk.HookRegistry,
	logger zerolog.Logger,
) *CartService {
	return &CartService{
		repo:   repo,
		stock:  stock,
		hooks:  hooks,
		logger: logger,
	}
}

// CreateCart creates a new empty cart with the given currency and optional
// customer or session identifier.
func (s *CartService) CreateCart(ctx context.Context, currency, sessionID string, customerID *uuid.UUID, expiresAt *time.Time) (*Cart, error) {
	if currency == "" {
		currency = "USD"
	}

	c := &Cart{
		ID:         uuid.New(),
		CustomerID: customerID,
		SessionID:  sessionID,
		Currency:   currency,
		ExpiresAt:  expiresAt,
		CreatedAt:  time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, c); err != nil {
		return nil, fmt.Errorf("cart service: create cart: %w", err)
	}

	s.logger.Info().Str("cart_id", c.ID.String()).Msg("cart created")
	return c, nil
}

// GetCart retrieves a cart by ID.
func (s *CartService) GetCart(ctx context.Context, id uuid.UUID) (*Cart, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrCartNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("cart service: get cart: %w", err)
	}
	return c, nil
}

// AddItem adds a product (or variant) to the cart.
// It fires the cart.before_add_item hook before persisting and
// cart.after_add_item afterward. Stock is validated when a stockChecker is set.
func (s *CartService) AddItem(ctx context.Context, cartID, productID uuid.UUID, variantID *uuid.UUID, quantity int, customFields map[string]interface{}) (*CartItem, error) {
	if quantity <= 0 {
		return nil, fmt.Errorf("cart service: add item: quantity must be greater than zero")
	}

	item := &CartItem{
		ID:           uuid.New(),
		CartID:       cartID,
		ProductID:    productID,
		VariantID:    variantID,
		Quantity:     quantity,
		CustomFields: customFields,
	}

	// Before hook – plugins may reject the addition by returning an error.
	if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
		Name:   sdk.HookBeforeCartAdd,
		Entity: item,
	}); err != nil {
		return nil, fmt.Errorf("cart service: hook %s: %w", sdk.HookBeforeCartAdd, err)
	}

	// Stock validation – include any quantity already in the cart so that
	// customers cannot bypass the limit by adding items incrementally.
	if s.stock != nil {
		existingQty := 0
		if cart, err := s.repo.FindByID(ctx, cartID); err == nil {
			for _, existing := range cart.Items {
				if existing.ProductID == productID && uuidPtrEqual(existing.VariantID, variantID) {
					existingQty = existing.Quantity
					break
				}
			}
		}
		available, err := s.stock.StockAvailable(ctx, productID, variantID, existingQty+quantity)
		if err != nil {
			return nil, fmt.Errorf("cart service: stock check: %w", err)
		}
		if !available {
			return nil, ErrInsufficientStock
		}
	}

	if err := s.repo.AddItem(ctx, item); err != nil {
		return nil, fmt.Errorf("cart service: add item: %w", err)
	}

	// After hook – informational; errors are logged but do not roll back.
	if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
		Name:   sdk.HookAfterCartAdd,
		Entity: item,
	}); err != nil {
		s.logger.Warn().Err(err).Str("hook", sdk.HookAfterCartAdd).Msg("cart: after_add_item hook error")
	}

	s.logger.Info().
		Str("cart_id", cartID.String()).
		Str("item_id", item.ID.String()).
		Str("product_id", productID.String()).
		Int("quantity", quantity).
		Msg("cart item added")

	return item, nil
}

// UpdateItemQuantity changes the quantity of an existing cart line item.
// It fires cart.before_update_item and cart.after_update_item hooks.
func (s *CartService) UpdateItemQuantity(ctx context.Context, itemID uuid.UUID, quantity int) error {
	if quantity <= 0 {
		return fmt.Errorf("cart service: update item: quantity must be greater than zero")
	}

	// Stock validation against the new absolute quantity.
	if s.stock != nil {
		item, err := s.repo.FindItemByID(ctx, itemID)
		if err != nil {
			return fmt.Errorf("cart service: update item fetch: %w", err)
		}
		available, err := s.stock.StockAvailable(ctx, item.ProductID, item.VariantID, quantity)
		if err != nil {
			return fmt.Errorf("cart service: update item stock check: %w", err)
		}
		if !available {
			return ErrInsufficientStock
		}
	}

	payload := map[string]interface{}{
		"item_id":  itemID,
		"quantity": quantity,
	}

	// Before hook.
	if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
		Name:    sdk.HookBeforeCartUpdate,
		Changes: payload,
	}); err != nil {
		return fmt.Errorf("cart service: hook %s: %w", sdk.HookBeforeCartUpdate, err)
	}

	if err := s.repo.UpdateItem(ctx, itemID, quantity); err != nil {
		return fmt.Errorf("cart service: update item: %w", err)
	}

	// After hook.
	if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
		Name:    sdk.HookAfterCartUpdate,
		Changes: payload,
	}); err != nil {
		s.logger.Warn().Err(err).Str("hook", sdk.HookAfterCartUpdate).Msg("cart: after_update_item hook error")
	}

	s.logger.Info().
		Str("item_id", itemID.String()).
		Int("quantity", quantity).
		Msg("cart item updated")

	return nil
}

// RemoveItem removes a line item from a cart.
// It fires cart.before_remove_item and cart.after_remove_item hooks.
func (s *CartService) RemoveItem(ctx context.Context, itemID uuid.UUID) error {
	payload := map[string]interface{}{
		"item_id": itemID,
	}

	// Before hook.
	if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
		Name:    sdk.HookBeforeCartRemove,
		Changes: payload,
	}); err != nil {
		return fmt.Errorf("cart service: hook %s: %w", sdk.HookBeforeCartRemove, err)
	}

	if err := s.repo.RemoveItem(ctx, itemID); err != nil {
		return fmt.Errorf("cart service: remove item: %w", err)
	}

	// After hook.
	if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
		Name:    sdk.HookAfterCartRemove,
		Changes: payload,
	}); err != nil {
		s.logger.Warn().Err(err).Str("hook", sdk.HookAfterCartRemove).Msg("cart: after_remove_item hook error")
	}

	s.logger.Info().Str("item_id", itemID.String()).Msg("cart item removed")
	return nil
}

// Additional domain errors.
var ErrInsufficientStock = errors.New("insufficient stock")

// uuidPtrEqual compares two nullable UUID pointers for equality.
func uuidPtrEqual(a, b *uuid.UUID) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
