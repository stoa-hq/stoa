package customer

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/internal/auth"
	"github.com/stoa-hq/stoa/pkg/sdk"
)

// Sentinel errors used throughout the customer domain.
var (
	ErrNotFound       = errors.New("not found")
	ErrEmailTaken     = errors.New("email already in use")
	ErrInvalidCreds   = errors.New("invalid credentials")
	ErrAddressOwner   = errors.New("address does not belong to customer")
)

// CustomerService provides business logic for the customer domain.
type CustomerService struct {
	repo   CustomerRepository
	hooks  *sdk.HookRegistry
	logger zerolog.Logger
}

// NewCustomerService creates a new CustomerService.
func NewCustomerService(
	repo CustomerRepository,
	hooks *sdk.HookRegistry,
	logger zerolog.Logger,
) *CustomerService {
	return &CustomerService{
		repo:   repo,
		hooks:  hooks,
		logger: logger,
	}
}

// GetByID retrieves a customer (with addresses) by primary key.
func (s *CustomerService) GetByID(ctx context.Context, id uuid.UUID) (*Customer, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// GetByEmail retrieves a customer by email. Returns ErrNotFound when absent.
func (s *CustomerService) GetByEmail(ctx context.Context, email string) (*Customer, error) {
	email = auth.NormalizeEmail(email)
	c, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("customer with email %q: %w", email, ErrNotFound)
	}
	return c, nil
}

// List returns a paginated list of customers matching the given filter.
func (s *CustomerService) List(ctx context.Context, filter CustomerFilter) ([]Customer, int, error) {
	return s.repo.FindAll(ctx, filter)
}

// Create registers a new customer. The password is hashed before storage.
func (s *CustomerService) Create(ctx context.Context, req CreateCustomerInput) (*Customer, error) {
	req.Email = auth.NormalizeEmail(req.Email)

	// Uniqueness check
	existing, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailTaken
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	c := &Customer{
		Email:        req.Email,
		PasswordHash: hash,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Active:       true,
		CustomFields: req.CustomFields,
	}

	if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
		Name:   sdk.HookBeforeCustomerCreate,
		Entity: c,
	}); err != nil {
		return nil, fmt.Errorf("before-create hook: %w", err)
	}

	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}

	if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
		Name:   sdk.HookAfterCustomerCreate,
		Entity: c,
	}); err != nil {
		s.logger.Warn().Err(err).Str("customer_id", c.ID.String()).
			Msg("after-create hook error (non-fatal)")
	}

	s.logger.Info().Str("customer_id", c.ID.String()).Msg("customer created")
	return c, nil
}

// Update applies the given changes to an existing customer.
func (s *CustomerService) Update(ctx context.Context, id uuid.UUID, req UpdateCustomerInput) (*Customer, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Email uniqueness check when changing email
	req.Email = auth.NormalizeEmail(req.Email)
	if req.Email != "" && req.Email != c.Email {
		existing, err := s.repo.FindByEmail(ctx, req.Email)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return nil, ErrEmailTaken
		}
		c.Email = req.Email
	}

	if req.FirstName != "" {
		c.FirstName = req.FirstName
	}
	if req.LastName != "" {
		c.LastName = req.LastName
	}
	if req.Active != nil {
		c.Active = *req.Active
	}
	if req.DefaultBillingAddressID != nil {
		c.DefaultBillingAddressID = req.DefaultBillingAddressID
	}
	if req.DefaultShippingAddressID != nil {
		c.DefaultShippingAddressID = req.DefaultShippingAddressID
	}
	if req.CustomFields != nil {
		c.CustomFields = req.CustomFields
	}

	if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
		Name:   sdk.HookBeforeCustomerUpdate,
		Entity: c,
	}); err != nil {
		return nil, fmt.Errorf("before-update hook: %w", err)
	}

	if err := s.repo.Update(ctx, c); err != nil {
		return nil, err
	}

	if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
		Name:   sdk.HookAfterCustomerUpdate,
		Entity: c,
	}); err != nil {
		s.logger.Warn().Err(err).Str("customer_id", c.ID.String()).
			Msg("after-update hook error (non-fatal)")
	}

	s.logger.Info().Str("customer_id", c.ID.String()).Msg("customer updated")
	return c, nil
}

// ChangePassword hashes a new password and persists it.
func (s *CustomerService) ChangePassword(ctx context.Context, id uuid.UUID, newPassword string) error {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}
	c.PasswordHash = hash

	return s.repo.Update(ctx, c)
}

// VerifyCredentials checks email/password and returns the matching customer.
func (s *CustomerService) VerifyCredentials(ctx context.Context, email, password string) (*Customer, error) {
	email = auth.NormalizeEmail(email)
	c, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrInvalidCreds
	}

	ok, err := auth.VerifyPassword(password, c.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("verifying password: %w", err)
	}
	if !ok {
		return nil, ErrInvalidCreds
	}

	return c, nil
}

// Delete removes a customer permanently.
func (s *CustomerService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.logger.Info().Str("customer_id", id.String()).Msg("customer deleted")
	return nil
}

// ---- Address methods --------------------------------------------------------

// AddAddress creates a new address for the given customer.
func (s *CustomerService) AddAddress(ctx context.Context, customerID uuid.UUID, req AddressInput) (*CustomerAddress, error) {
	// Ensure customer exists
	if _, err := s.repo.FindByID(ctx, customerID); err != nil {
		return nil, err
	}

	a := &CustomerAddress{
		CustomerID:  customerID,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Company:     req.Company,
		Street:      req.Street,
		City:        req.City,
		Zip:         req.Zip,
		CountryCode: req.CountryCode,
		Phone:       req.Phone,
	}

	if err := s.repo.CreateAddress(ctx, a); err != nil {
		return nil, err
	}

	return a, nil
}

// UpdateAddress applies changes to an existing address, enforcing ownership.
func (s *CustomerService) UpdateAddress(ctx context.Context, customerID, addressID uuid.UUID, req AddressInput) (*CustomerAddress, error) {
	addresses, err := s.repo.FindAddressesByCustomerID(ctx, customerID)
	if err != nil {
		return nil, err
	}

	var target *CustomerAddress
	for i := range addresses {
		if addresses[i].ID == addressID {
			target = &addresses[i]
			break
		}
	}
	if target == nil {
		return nil, ErrAddressOwner
	}

	target.FirstName = req.FirstName
	target.LastName = req.LastName
	target.Company = req.Company
	target.Street = req.Street
	target.City = req.City
	target.Zip = req.Zip
	target.CountryCode = req.CountryCode
	target.Phone = req.Phone

	if err := s.repo.UpdateAddress(ctx, target); err != nil {
		return nil, err
	}

	return target, nil
}

// DeleteAddress removes an address, enforcing customer ownership.
func (s *CustomerService) DeleteAddress(ctx context.Context, customerID, addressID uuid.UUID) error {
	addresses, err := s.repo.FindAddressesByCustomerID(ctx, customerID)
	if err != nil {
		return err
	}

	found := false
	for _, a := range addresses {
		if a.ID == addressID {
			found = true
			break
		}
	}
	if !found {
		return ErrAddressOwner
	}

	return s.repo.DeleteAddress(ctx, addressID)
}

// GetAddresses returns all addresses for a customer.
func (s *CustomerService) GetAddresses(ctx context.Context, customerID uuid.UUID) ([]CustomerAddress, error) {
	return s.repo.FindAddressesByCustomerID(ctx, customerID)
}

// ---- Input types ------------------------------------------------------------

// CreateCustomerInput carries the data needed to register a new customer.
type CreateCustomerInput struct {
	Email        string
	Password     string
	FirstName    string
	LastName     string
	CustomFields map[string]interface{}
}

// UpdateCustomerInput carries the optional fields for a customer update.
// Only non-zero / non-nil fields are applied.
type UpdateCustomerInput struct {
	Email                    string
	FirstName                string
	LastName                 string
	Active                   *bool
	DefaultBillingAddressID  *uuid.UUID
	DefaultShippingAddressID *uuid.UUID
	CustomFields             map[string]interface{}
}

// AddressInput carries the data for creating or updating an address.
type AddressInput struct {
	FirstName   string
	LastName    string
	Company     string
	Street      string
	City        string
	Zip         string
	CountryCode string
	Phone       string
}
