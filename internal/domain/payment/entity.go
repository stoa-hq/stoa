package payment

import (
	"time"

	"github.com/google/uuid"
)

// PaymentMethod represents a payment option available in the store.
// Config stores provider-specific credentials/settings as raw JSON bytes.
type PaymentMethod struct {
	ID           uuid.UUID                   `json:"id"`
	Provider     string                      `json:"provider"`
	Active       bool                        `json:"active"`
	Config       []byte                      `json:"-"` // never expose raw config
	CustomFields map[string]interface{}      `json:"custom_fields,omitempty"`
	CreatedAt    time.Time                   `json:"created_at"`
	UpdatedAt    time.Time                   `json:"updated_at"`
	Translations []PaymentMethodTranslation  `json:"translations,omitempty"`
}

// PaymentMethodTranslation holds locale-specific content for a PaymentMethod.
type PaymentMethodTranslation struct {
	PaymentMethodID uuid.UUID `json:"payment_method_id"`
	Locale          string    `json:"locale"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
}

// PaymentTransaction records a payment attempt or completion for an order.
// Amount is stored as integer cents (e.g. 4999 = €49.99).
type PaymentTransaction struct {
	ID                uuid.UUID `json:"id"`
	OrderID           uuid.UUID `json:"order_id"`
	PaymentMethodID   uuid.UUID `json:"payment_method_id"`
	Status            string    `json:"status"`
	Currency          string    `json:"currency"`
	Amount            int       `json:"amount"`
	ProviderReference string    `json:"provider_reference"`
	CreatedAt         time.Time `json:"created_at"`
}
