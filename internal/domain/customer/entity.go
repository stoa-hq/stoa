package customer

import (
	"time"

	"github.com/google/uuid"
)

// Customer is the aggregate root for the customer domain.
type Customer struct {
	ID                       uuid.UUID              `json:"id"`
	Email                    string                 `json:"email"`
	PasswordHash             string                 `json:"-"`
	FirstName                string                 `json:"first_name"`
	LastName                 string                 `json:"last_name"`
	Active                   bool                   `json:"active"`
	DefaultBillingAddressID  *uuid.UUID             `json:"default_billing_address_id,omitempty"`
	DefaultShippingAddressID *uuid.UUID             `json:"default_shipping_address_id,omitempty"`
	CustomFields             map[string]interface{} `json:"custom_fields,omitempty"`
	CreatedAt                time.Time              `json:"created_at"`
	UpdatedAt                time.Time              `json:"updated_at"`

	// Relations (populated on demand)
	Addresses []CustomerAddress `json:"addresses,omitempty"`
}

// CustomerAddress represents a postal address belonging to a customer.
type CustomerAddress struct {
	ID          uuid.UUID `json:"id"`
	CustomerID  uuid.UUID `json:"customer_id"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	Company     string    `json:"company,omitempty"`
	Street      string    `json:"street"`
	City        string    `json:"city"`
	Zip         string    `json:"zip"`
	CountryCode string    `json:"country_code"`
	Phone       string    `json:"phone,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
