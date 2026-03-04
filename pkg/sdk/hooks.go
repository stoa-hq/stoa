package sdk

import (
	"context"
	"sync"
)

// Hook names follow the pattern: entity.before_action / entity.after_action
const (
	HookBeforeProductCreate  = "product.before_create"
	HookAfterProductCreate   = "product.after_create"
	HookBeforeProductUpdate  = "product.before_update"
	HookAfterProductUpdate   = "product.after_update"
	HookBeforeProductDelete  = "product.before_delete"
	HookAfterProductDelete   = "product.after_delete"

	HookBeforeCategoryCreate = "category.before_create"
	HookAfterCategoryCreate  = "category.after_create"
	HookBeforeCategoryUpdate = "category.before_update"
	HookAfterCategoryUpdate  = "category.after_update"
	HookBeforeCategoryDelete = "category.before_delete"
	HookAfterCategoryDelete  = "category.after_delete"

	HookBeforeOrderCreate    = "order.before_create"
	HookAfterOrderCreate     = "order.after_create"
	HookBeforeOrderUpdate    = "order.before_update"
	HookAfterOrderUpdate     = "order.after_update"

	HookBeforeCartAdd        = "cart.before_add_item"
	HookAfterCartAdd         = "cart.after_add_item"
	HookBeforeCartUpdate     = "cart.before_update_item"
	HookAfterCartUpdate      = "cart.after_update_item"
	HookBeforeCartRemove     = "cart.before_remove_item"
	HookAfterCartRemove      = "cart.after_remove_item"

	HookBeforeCustomerCreate = "customer.before_create"
	HookAfterCustomerCreate  = "customer.after_create"
	HookBeforeCustomerUpdate = "customer.before_update"
	HookAfterCustomerUpdate  = "customer.after_update"

	HookAfterPaymentComplete = "payment.after_complete"
	HookAfterPaymentFailed   = "payment.after_failed"

	HookBeforeCheckout       = "checkout.before"
	HookAfterCheckout        = "checkout.after"
)

// HookHandler is the function signature for hook handlers.
type HookHandler func(ctx context.Context, event *HookEvent) error

// HookEvent contains the data passed to hook handlers.
type HookEvent struct {
	Name     string
	Entity   interface{}
	Changes  map[string]interface{}
	Metadata map[string]interface{}
}

// HookRegistry manages hook registrations and dispatching.
type HookRegistry struct {
	mu       sync.RWMutex
	handlers map[string][]HookHandler
}

func NewHookRegistry() *HookRegistry {
	return &HookRegistry{
		handlers: make(map[string][]HookHandler),
	}
}

// On registers a handler for the given hook name.
func (r *HookRegistry) On(name string, handler HookHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[name] = append(r.handlers[name], handler)
}

// Dispatch fires all handlers for the given hook name.
// Before-hooks can cancel the operation by returning an error.
func (r *HookRegistry) Dispatch(ctx context.Context, event *HookEvent) error {
	r.mu.RLock()
	handlers := r.handlers[event.Name]
	r.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			return err
		}
	}
	return nil
}
