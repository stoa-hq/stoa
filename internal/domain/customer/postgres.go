package customer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// PostgresRepository is the PostgreSQL implementation of CustomerRepository.
type PostgresRepository struct {
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

// NewPostgresRepository creates a new PostgresRepository.
func NewPostgresRepository(pool *pgxpool.Pool, logger zerolog.Logger) *PostgresRepository {
	return &PostgresRepository{
		pool:   pool,
		logger: logger,
	}
}

// FindByID retrieves a customer and their addresses by primary key.
func (r *PostgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*Customer, error) {
	const query = `
		SELECT
			id, email, password_hash, first_name, last_name, active,
			default_billing_address_id, default_shipping_address_id,
			custom_fields, created_at, updated_at
		FROM customers
		WHERE id = $1`

	c, err := r.scanCustomer(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("customer not found: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("finding customer by id: %w", err)
	}

	addresses, err := r.FindAddressesByCustomerID(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	c.Addresses = addresses

	return c, nil
}

// FindByEmail retrieves a customer by their email address.
func (r *PostgresRepository) FindByEmail(ctx context.Context, email string) (*Customer, error) {
	const query = `
		SELECT
			id, email, password_hash, first_name, last_name, active,
			default_billing_address_id, default_shipping_address_id,
			custom_fields, created_at, updated_at
		FROM customers
		WHERE email = $1`

	c, err := r.scanCustomer(r.pool.QueryRow(ctx, query, email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("finding customer by email: %w", err)
	}

	return c, nil
}

// FindAll retrieves a paginated, filtered list of customers.
func (r *PostgresRepository) FindAll(ctx context.Context, filter CustomerFilter) ([]Customer, int, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 20
	}

	args := []interface{}{}
	conditions := []string{}
	argIdx := 1

	if filter.Search != "" {
		conditions = append(conditions,
			fmt.Sprintf(
				"(first_name ILIKE $%d OR last_name ILIKE $%d OR email ILIKE $%d)",
				argIdx, argIdx+1, argIdx+2,
			),
		)
		pattern := "%" + filter.Search + "%"
		args = append(args, pattern, pattern, pattern)
		argIdx += 3
	}

	if filter.Active != nil {
		conditions = append(conditions, fmt.Sprintf("active = $%d", argIdx))
		args = append(args, *filter.Active)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM customers %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting customers: %w", err)
	}

	offset := (filter.Page - 1) * filter.Limit
	selectQuery := fmt.Sprintf(`
		SELECT
			id, email, password_hash, first_name, last_name, active,
			default_billing_address_id, default_shipping_address_id,
			custom_fields, created_at, updated_at
		FROM customers
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, argIdx, argIdx+1,
	)
	args = append(args, filter.Limit, offset)

	rows, err := r.pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying customers: %w", err)
	}
	defer rows.Close()

	var customers []Customer
	for rows.Next() {
		c, err := r.scanCustomer(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning customer row: %w", err)
		}
		customers = append(customers, *c)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterating customer rows: %w", err)
	}

	return customers, total, nil
}

// Create inserts a new customer into the database.
func (r *PostgresRepository) Create(ctx context.Context, c *Customer) error {
	c.ID = uuid.New()
	now := time.Now().UTC()
	c.CreatedAt = now
	c.UpdatedAt = now

	cfJSON, err := marshalCustomFields(c.CustomFields)
	if err != nil {
		return err
	}

	const query = `
		INSERT INTO customers (
			id, email, password_hash, first_name, last_name, active,
			default_billing_address_id, default_shipping_address_id,
			custom_fields, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err = r.pool.Exec(ctx, query,
		c.ID, c.Email, c.PasswordHash, c.FirstName, c.LastName, c.Active,
		c.DefaultBillingAddressID, c.DefaultShippingAddressID,
		cfJSON, c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("creating customer: %w", err)
	}

	return nil
}

// Update persists changes to an existing customer.
func (r *PostgresRepository) Update(ctx context.Context, c *Customer) error {
	c.UpdatedAt = time.Now().UTC()

	cfJSON, err := marshalCustomFields(c.CustomFields)
	if err != nil {
		return err
	}

	const query = `
		UPDATE customers SET
			email = $2,
			password_hash = $3,
			first_name = $4,
			last_name = $5,
			active = $6,
			default_billing_address_id = $7,
			default_shipping_address_id = $8,
			custom_fields = $9,
			updated_at = $10
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, query,
		c.ID, c.Email, c.PasswordHash, c.FirstName, c.LastName, c.Active,
		c.DefaultBillingAddressID, c.DefaultShippingAddressID,
		cfJSON, c.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("updating customer: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("customer not found: %w", ErrNotFound)
	}

	return nil
}

// Delete removes a customer record by ID.
func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM customers WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("deleting customer: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("customer not found: %w", ErrNotFound)
	}
	return nil
}

// CreateAddress inserts a new customer address.
func (r *PostgresRepository) CreateAddress(ctx context.Context, a *CustomerAddress) error {
	a.ID = uuid.New()
	now := time.Now().UTC()
	a.CreatedAt = now
	a.UpdatedAt = now

	const query = `
		INSERT INTO customer_addresses (
			id, customer_id, first_name, last_name, company,
			street, city, zip, country_code, phone,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err := r.pool.Exec(ctx, query,
		a.ID, a.CustomerID, a.FirstName, a.LastName, a.Company,
		a.Street, a.City, a.Zip, a.CountryCode, a.Phone,
		a.CreatedAt, a.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("creating customer address: %w", err)
	}

	return nil
}

// UpdateAddress persists changes to an existing customer address.
func (r *PostgresRepository) UpdateAddress(ctx context.Context, a *CustomerAddress) error {
	a.UpdatedAt = time.Now().UTC()

	const query = `
		UPDATE customer_addresses SET
			first_name = $2,
			last_name = $3,
			company = $4,
			street = $5,
			city = $6,
			zip = $7,
			country_code = $8,
			phone = $9,
			updated_at = $10
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, query,
		a.ID, a.FirstName, a.LastName, a.Company,
		a.Street, a.City, a.Zip, a.CountryCode, a.Phone,
		a.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("updating customer address: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("address not found: %w", ErrNotFound)
	}

	return nil
}

// DeleteAddress removes a customer address by ID.
func (r *PostgresRepository) DeleteAddress(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM customer_addresses WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("deleting customer address: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("address not found: %w", ErrNotFound)
	}
	return nil
}

// FindAddressesByCustomerID retrieves all addresses for a given customer.
func (r *PostgresRepository) FindAddressesByCustomerID(ctx context.Context, customerID uuid.UUID) ([]CustomerAddress, error) {
	const query = `
		SELECT
			id, customer_id, first_name, last_name, company,
			street, city, zip, country_code, phone,
			created_at, updated_at
		FROM customer_addresses
		WHERE customer_id = $1
		ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, query, customerID)
	if err != nil {
		return nil, fmt.Errorf("querying customer addresses: %w", err)
	}
	defer rows.Close()

	var addresses []CustomerAddress
	for rows.Next() {
		var a CustomerAddress
		err := rows.Scan(
			&a.ID, &a.CustomerID, &a.FirstName, &a.LastName, &a.Company,
			&a.Street, &a.City, &a.Zip, &a.CountryCode, &a.Phone,
			&a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning address row: %w", err)
		}
		addresses = append(addresses, a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating address rows: %w", err)
	}

	return addresses, nil
}

// ---- helpers ----------------------------------------------------------------

// rowScanner is satisfied by both *pgx.Row and pgx.Rows so we can share
// scanCustomer between single-row and multi-row queries.
type rowScanner interface {
	Scan(dest ...interface{}) error
}

func (r *PostgresRepository) scanCustomer(row rowScanner) (*Customer, error) {
	var c Customer
	var cfRaw []byte

	err := row.Scan(
		&c.ID, &c.Email, &c.PasswordHash, &c.FirstName, &c.LastName, &c.Active,
		&c.DefaultBillingAddressID, &c.DefaultShippingAddressID,
		&cfRaw, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if len(cfRaw) > 0 && string(cfRaw) != "null" {
		if err := json.Unmarshal(cfRaw, &c.CustomFields); err != nil {
			return nil, fmt.Errorf("unmarshaling custom_fields: %w", err)
		}
	}

	return &c, nil
}

func marshalCustomFields(cf map[string]interface{}) ([]byte, error) {
	if cf == nil {
		return []byte("{}"), nil
	}
	b, err := json.Marshal(cf)
	if err != nil {
		return nil, fmt.Errorf("marshaling custom_fields: %w", err)
	}
	return b, nil
}
