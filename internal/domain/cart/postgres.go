package cart

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// postgresRepository is the PostgreSQL-backed implementation of CartRepository.
type postgresRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewPostgresRepository creates a CartRepository backed by a pgxpool connection pool.
func NewPostgresRepository(db *pgxpool.Pool, logger zerolog.Logger) CartRepository {
	return &postgresRepository{db: db, logger: logger}
}

// FindByID retrieves a cart and all its line items by primary key.
func (r *postgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*Cart, error) {
	const q = `
		SELECT id, customer_id, session_id, currency, expires_at, created_at
		FROM carts
		WHERE id = $1`

	c, err := r.scanCart(ctx, q, id)
	if err != nil {
		return nil, err
	}

	if err := r.loadItems(ctx, c); err != nil {
		return nil, err
	}

	return c, nil
}

// FindBySessionID retrieves the active cart for a guest session.
func (r *postgresRepository) FindBySessionID(ctx context.Context, sessionID string) (*Cart, error) {
	const q = `
		SELECT id, customer_id, session_id, currency, expires_at, created_at
		FROM carts
		WHERE session_id = $1
		  AND (expires_at IS NULL OR expires_at > now())
		LIMIT 1`

	c, err := r.scanCart(ctx, q, sessionID)
	if err != nil {
		return nil, err
	}

	if err := r.loadItems(ctx, c); err != nil {
		return nil, err
	}

	return c, nil
}

// FindByCustomerID retrieves the active cart for a registered customer.
func (r *postgresRepository) FindByCustomerID(ctx context.Context, customerID uuid.UUID) (*Cart, error) {
	const q = `
		SELECT id, customer_id, session_id, currency, expires_at, created_at
		FROM carts
		WHERE customer_id = $1
		  AND (expires_at IS NULL OR expires_at > now())
		LIMIT 1`

	c, err := r.scanCart(ctx, q, customerID)
	if err != nil {
		return nil, err
	}

	if err := r.loadItems(ctx, c); err != nil {
		return nil, err
	}

	return c, nil
}

// Create persists a new cart.
func (r *postgresRepository) Create(ctx context.Context, c *Cart) error {
	const q = `
		INSERT INTO carts (id, customer_id, session_id, currency, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}

	_, err := r.db.Exec(ctx, q,
		c.ID,
		c.CustomerID,
		c.SessionID,
		c.Currency,
		c.ExpiresAt,
		c.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("cart: create: %w", err)
	}

	return nil
}

// Delete removes a cart by ID. Line items are removed via ON DELETE CASCADE.
func (r *postgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM carts WHERE id = $1`

	ct, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("cart: delete: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("cart: delete: %w", ErrCartNotFound)
	}

	return nil
}

// AddItem inserts a new line item into a cart or increments the quantity when
// the same product/variant is already present (upsert).
func (r *postgresRepository) AddItem(ctx context.Context, item *CartItem) error {
	const q = `
		INSERT INTO cart_items (id, cart_id, product_id, variant_id, quantity, custom_fields)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (cart_id, product_id, (COALESCE(variant_id, '00000000-0000-0000-0000-000000000000'::uuid)))
		DO UPDATE SET quantity = cart_items.quantity + EXCLUDED.quantity
		RETURNING id`

	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}

	customFieldsJSON, err := marshalCustomFields(item.CustomFields)
	if err != nil {
		return fmt.Errorf("cart: add item: marshal custom fields: %w", err)
	}

	err = r.db.QueryRow(ctx, q,
		item.ID,
		item.CartID,
		item.ProductID,
		item.VariantID,
		item.Quantity,
		customFieldsJSON,
	).Scan(&item.ID)
	if err != nil {
		return fmt.Errorf("cart: add item: %w", err)
	}

	return nil
}

// UpdateItem changes the quantity of an existing cart line item.
func (r *postgresRepository) UpdateItem(ctx context.Context, itemID uuid.UUID, quantity int) error {
	const q = `UPDATE cart_items SET quantity = $1 WHERE id = $2`

	ct, err := r.db.Exec(ctx, q, quantity, itemID)
	if err != nil {
		return fmt.Errorf("cart: update item: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("cart: update item: %w", ErrItemNotFound)
	}

	return nil
}

// RemoveItem deletes a single line item.
func (r *postgresRepository) RemoveItem(ctx context.Context, itemID uuid.UUID) error {
	const q = `DELETE FROM cart_items WHERE id = $1`

	ct, err := r.db.Exec(ctx, q, itemID)
	if err != nil {
		return fmt.Errorf("cart: remove item: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("cart: remove item: %w", ErrItemNotFound)
	}

	return nil
}

// CleanExpired removes all carts whose expires_at timestamp has passed.
func (r *postgresRepository) CleanExpired(ctx context.Context) error {
	const q = `DELETE FROM carts WHERE expires_at IS NOT NULL AND expires_at <= now()`

	ct, err := r.db.Exec(ctx, q)
	if err != nil {
		return fmt.Errorf("cart: clean expired: %w", err)
	}

	r.logger.Info().
		Int64("removed", ct.RowsAffected()).
		Msg("cart: cleaned expired carts")

	return nil
}

// --- helpers ----------------------------------------------------------------

// scanCart executes a single-row cart query and scans the result.
func (r *postgresRepository) scanCart(ctx context.Context, query string, arg interface{}) (*Cart, error) {
	row := r.db.QueryRow(ctx, query, arg)

	c := &Cart{}
	err := row.Scan(
		&c.ID,
		&c.CustomerID,
		&c.SessionID,
		&c.Currency,
		&c.ExpiresAt,
		&c.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCartNotFound
		}
		return nil, fmt.Errorf("cart: scan: %w", err)
	}

	return c, nil
}

// loadItems fetches and attaches all line items for the given cart.
func (r *postgresRepository) loadItems(ctx context.Context, c *Cart) error {
	const q = `
		SELECT id, cart_id, product_id, variant_id, quantity, custom_fields
		FROM cart_items
		WHERE cart_id = $1
		ORDER BY id`

	rows, err := r.db.Query(ctx, q, c.ID)
	if err != nil {
		return fmt.Errorf("cart: load items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item CartItem
		var customFieldsRaw []byte

		if err := rows.Scan(
			&item.ID,
			&item.CartID,
			&item.ProductID,
			&item.VariantID,
			&item.Quantity,
			&customFieldsRaw,
		); err != nil {
			return fmt.Errorf("cart: scan item: %w", err)
		}

		if customFieldsRaw != nil {
			if err := json.Unmarshal(customFieldsRaw, &item.CustomFields); err != nil {
				return fmt.Errorf("cart: unmarshal item custom_fields: %w", err)
			}
		}

		c.Items = append(c.Items, item)
	}

	return rows.Err()
}

// marshalCustomFields serialises a custom-fields map to JSON, returning nil
// when the map is empty so the column stores a SQL NULL rather than '{}'.
func marshalCustomFields(cf map[string]interface{}) ([]byte, error) {
	if len(cf) == 0 {
		return nil, nil
	}
	return json.Marshal(cf)
}

// Sentinel errors returned by the repository.
var (
	ErrCartNotFound = errors.New("cart not found")
	ErrItemNotFound = errors.New("cart item not found")
)
