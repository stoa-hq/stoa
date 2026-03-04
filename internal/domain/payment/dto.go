package payment

// PaymentTranslationInput holds locale-specific fields for create/update requests.
type PaymentTranslationInput struct {
	Locale      string `json:"locale"      validate:"required,min=2,max=10"`
	Name        string `json:"name"        validate:"required,min=1,max=255"`
	Description string `json:"description"`
}

// CreatePaymentMethodRequest is the request body for creating a payment method.
type CreatePaymentMethodRequest struct {
	Provider     string                   `json:"provider"      validate:"required,min=1,max=100"`
	Active       bool                     `json:"active"`
	Config       []byte                   `json:"config,omitempty"`
	CustomFields map[string]interface{}   `json:"custom_fields,omitempty"`
	Translations []PaymentTranslationInput `json:"translations" validate:"dive"`
}

// UpdatePaymentMethodRequest is the request body for updating a payment method.
type UpdatePaymentMethodRequest struct {
	Provider     string                   `json:"provider"      validate:"required,min=1,max=100"`
	Active       bool                     `json:"active"`
	Config       []byte                   `json:"config,omitempty"`
	CustomFields map[string]interface{}   `json:"custom_fields,omitempty"`
	Translations []PaymentTranslationInput `json:"translations" validate:"dive"`
}

// ListPaymentMethodsRequest holds query parameters for the List endpoint.
type ListPaymentMethodsRequest struct {
	Page   int   `json:"page"`
	Limit  int   `json:"limit"`
	Active *bool `json:"active,omitempty"`
}

// CreateTransactionRequest is the request body for creating a payment transaction.
type CreateTransactionRequest struct {
	OrderID           string `json:"order_id"            validate:"required,uuid"`
	PaymentMethodID   string `json:"payment_method_id"   validate:"required,uuid"`
	Status            string `json:"status"              validate:"required,min=1,max=50"`
	Currency          string `json:"currency"            validate:"required,len=3"`
	Amount            int    `json:"amount"              validate:"required,min=0"`
	ProviderReference string `json:"provider_reference"`
}
