package customer

import (
	"time"

	"github.com/google/uuid"
)

// ---- Request DTOs -----------------------------------------------------------

// CreateCustomerRequest is the admin-facing payload for creating a customer.
type CreateCustomerRequest struct {
	Email     string `json:"email"      validate:"required,email,max=255"`
	Password  string `json:"password"   validate:"required,min=8,max=128"`
	FirstName string `json:"first_name" validate:"required,max=100"`
	LastName  string `json:"last_name"  validate:"required,max=100"`
}

// UpdateCustomerRequest is the admin-facing payload for updating a customer.
// All fields are optional; only provided (non-zero) fields are applied.
type UpdateCustomerRequest struct {
	Email                    string                 `json:"email,omitempty"                       validate:"omitempty,email,max=255"`
	FirstName                string                 `json:"first_name,omitempty"                  validate:"omitempty,max=100"`
	LastName                 string                 `json:"last_name,omitempty"                   validate:"omitempty,max=100"`
	Active                   *bool                  `json:"active,omitempty"`
	DefaultBillingAddressID  *uuid.UUID             `json:"default_billing_address_id,omitempty"`
	DefaultShippingAddressID *uuid.UUID             `json:"default_shipping_address_id,omitempty"`
	CustomFields             map[string]interface{} `json:"custom_fields,omitempty"`
}

// RegisterRequest is the store-facing payload for self-registration.
type RegisterRequest struct {
	Email     string `json:"email"      validate:"required,email,max=255"`
	Password  string `json:"password"   validate:"required,min=8,max=128"`
	FirstName string `json:"first_name" validate:"required,max=100"`
	LastName  string `json:"last_name"  validate:"required,max=100"`
}

// UpdateAccountRequest is the store-facing payload for account self-update.
type UpdateAccountRequest struct {
	FirstName string `json:"first_name,omitempty" validate:"omitempty,max=100"`
	LastName  string `json:"last_name,omitempty"  validate:"omitempty,max=100"`
	Email     string `json:"email,omitempty"      validate:"omitempty,email,max=255"`
}

// AddressRequest is the payload for creating or updating an address.
type AddressRequest struct {
	FirstName   string `json:"first_name"   validate:"required,max=100"`
	LastName    string `json:"last_name"    validate:"required,max=100"`
	Company     string `json:"company,omitempty"      validate:"omitempty,max=255"`
	Street      string `json:"street"       validate:"required,max=255"`
	City        string `json:"city"         validate:"required,max=100"`
	Zip         string `json:"zip"          validate:"required,max=20"`
	CountryCode string `json:"country_code" validate:"required,len=2"`
	Phone       string `json:"phone,omitempty"        validate:"omitempty,max=50"`
}

// ---- Response DTOs ----------------------------------------------------------

// CustomerResponse is the safe outward representation of a customer.
// PasswordHash is intentionally omitted.
type CustomerResponse struct {
	ID                       uuid.UUID         `json:"id"`
	Email                    string            `json:"email"`
	FirstName                string            `json:"first_name"`
	LastName                 string            `json:"last_name"`
	Active                   bool              `json:"active"`
	DefaultBillingAddressID  *uuid.UUID        `json:"default_billing_address_id,omitempty"`
	DefaultShippingAddressID *uuid.UUID        `json:"default_shipping_address_id,omitempty"`
	CustomFields             map[string]interface{} `json:"custom_fields,omitempty"`
	CreatedAt                time.Time         `json:"created_at"`
	UpdatedAt                time.Time         `json:"updated_at"`
	Addresses                []AddressResponse `json:"addresses,omitempty"`
}

// AddressResponse is the outward representation of a customer address.
type AddressResponse struct {
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

// ---- Mappers ----------------------------------------------------------------

// ToResponse converts a Customer entity to a CustomerResponse DTO.
func ToResponse(c *Customer) CustomerResponse {
	resp := CustomerResponse{
		ID:                       c.ID,
		Email:                    c.Email,
		FirstName:                c.FirstName,
		LastName:                 c.LastName,
		Active:                   c.Active,
		DefaultBillingAddressID:  c.DefaultBillingAddressID,
		DefaultShippingAddressID: c.DefaultShippingAddressID,
		CustomFields:             c.CustomFields,
		CreatedAt:                c.CreatedAt,
		UpdatedAt:                c.UpdatedAt,
	}

	if len(c.Addresses) > 0 {
		resp.Addresses = make([]AddressResponse, len(c.Addresses))
		for i, a := range c.Addresses {
			resp.Addresses[i] = ToAddressResponse(&a)
		}
	}

	return resp
}

// ToAddressResponse converts a CustomerAddress entity to an AddressResponse DTO.
func ToAddressResponse(a *CustomerAddress) AddressResponse {
	return AddressResponse{
		ID:          a.ID,
		CustomerID:  a.CustomerID,
		FirstName:   a.FirstName,
		LastName:    a.LastName,
		Company:     a.Company,
		Street:      a.Street,
		City:        a.City,
		Zip:         a.Zip,
		CountryCode: a.CountryCode,
		Phone:       a.Phone,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
	}
}
