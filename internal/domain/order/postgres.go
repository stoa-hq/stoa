package order

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// PostgresRepository is a PostgreSQL-backed implementation of OrderRepository.
type PostgresRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewPostgresRepository creates a new PostgreSQL order repository.
func NewPostgresRepository(db *pgxpool.Pool, logger zerolog.Logger) *PostgresRepository {
	return &PostgresRepository{db: db, logger: logger}
}

// -------------------------------------------------------------------
// FindByID
// -------------------------------------------------------------------

func (r *PostgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*Order, error) {
	const q = `
		SELECT
			id, order_number, customer_id, status, currency,
			subtotal_net, subtotal_gross, shipping_cost, tax_total, total,
			billing_address, shipping_address,
			payment_method_id, shipping_method_id,
			notes, custom_fields,
			created_at, updated_at
		FROM orders
		WHERE id = $1`

	o := &Order{}
	var billingRaw, shippingRaw, customFieldsRaw []byte

	err := r.db.QueryRow(ctx, q, id).Scan(
		&o.ID, &o.OrderNumber, &o.CustomerID, &o.Status, &o.Currency,
		&o.SubtotalNet, &o.SubtotalGross, &o.ShippingCost, &o.TaxTotal, &o.Total,
		&billingRaw, &shippingRaw,
		&o.PaymentMethodID, &o.ShippingMethodID,
		&o.Notes, &customFieldsRaw,
		&o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("order %s not found", id)
		}
		return nil, fmt.Errorf("finding order by id: %w", err)
	}

	if err := unmarshalJSONB(billingRaw, &o.BillingAddress); err != nil {
		return nil, fmt.Errorf("unmarshalling billing address: %w", err)
	}
	if err := unmarshalJSONB(shippingRaw, &o.ShippingAddress); err != nil {
		return nil, fmt.Errorf("unmarshalling shipping address: %w", err)
	}
	if err := unmarshalJSONB(customFieldsRaw, &o.CustomFields); err != nil {
		return nil, fmt.Errorf("unmarshalling custom fields: %w", err)
	}

	items, err := r.findItemsByOrderID(ctx, id)
	if err != nil {
		return nil, err
	}
	o.Items = items

	history, err := r.findStatusHistoryByOrderID(ctx, id)
	if err != nil {
		return nil, err
	}
	o.StatusHistory = history

	return o, nil
}

// -------------------------------------------------------------------
// FindAll
// -------------------------------------------------------------------

func (r *PostgresRepository) FindAll(ctx context.Context, filter OrderFilter) ([]Order, int, error) {
	where := []string{"1=1"}
	args := []interface{}{}
	idx := 1

	if filter.Status != "" {
		where = append(where, fmt.Sprintf("status = $%d", idx))
		args = append(args, filter.Status)
		idx++
	}
	if filter.CustomerID != nil {
		where = append(where, fmt.Sprintf("customer_id = $%d", idx))
		args = append(args, *filter.CustomerID)
		idx++
	}

	whereClause := strings.Join(where, " AND ")

	// Count total rows for pagination meta.
	var total int
	countQ := fmt.Sprintf("SELECT COUNT(*) FROM orders WHERE %s", whereClause)
	if err := r.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting orders: %w", err)
	}

	// Apply sort / order.
	sortCol := "created_at"
	if filter.Sort != "" {
		allowed := map[string]bool{"created_at": true, "total": true, "status": true, "order_number": true}
		if allowed[filter.Sort] {
			sortCol = filter.Sort
		}
	}
	sortDir := "DESC"
	if strings.ToLower(filter.Order) == "asc" {
		sortDir = "ASC"
	}

	// Apply pagination.
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 || limit > 200 {
		limit = 20
	}
	offset := (page - 1) * limit

	listQ := fmt.Sprintf(`
		SELECT
			id, order_number, customer_id, status, currency,
			subtotal_net, subtotal_gross, shipping_cost, tax_total, total,
			billing_address, shipping_address,
			payment_method_id, shipping_method_id,
			notes, custom_fields,
			created_at, updated_at
		FROM orders
		WHERE %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		whereClause, sortCol, sortDir, idx, idx+1,
	)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, listQ, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying orders: %w", err)
	}
	defer rows.Close()

	orders, err := r.scanRows(rows)
	if err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// -------------------------------------------------------------------
// FindByCustomerID
// -------------------------------------------------------------------

func (r *PostgresRepository) FindByCustomerID(ctx context.Context, customerID uuid.UUID) ([]Order, error) {
	const q = `
		SELECT
			id, order_number, customer_id, status, currency,
			subtotal_net, subtotal_gross, shipping_cost, tax_total, total,
			billing_address, shipping_address,
			payment_method_id, shipping_method_id,
			notes, custom_fields,
			created_at, updated_at
		FROM orders
		WHERE customer_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, q, customerID)
	if err != nil {
		return nil, fmt.Errorf("querying orders by customer id: %w", err)
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// -------------------------------------------------------------------
// Create
// -------------------------------------------------------------------

func (r *PostgresRepository) Create(ctx context.Context, o *Order) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	billingRaw, err := marshalJSONB(o.BillingAddress)
	if err != nil {
		return fmt.Errorf("marshalling billing address: %w", err)
	}
	shippingRaw, err := marshalJSONB(o.ShippingAddress)
	if err != nil {
		return fmt.Errorf("marshalling shipping address: %w", err)
	}
	customFieldsRaw, err := marshalJSONB(o.CustomFields)
	if err != nil {
		return fmt.Errorf("marshalling custom fields: %w", err)
	}

	now := time.Now().UTC()
	o.CreatedAt = now
	o.UpdatedAt = now

	const insertOrder = `
		INSERT INTO orders (
			id, order_number, customer_id, status, currency,
			subtotal_net, subtotal_gross, shipping_cost, tax_total, total,
			billing_address, shipping_address,
			payment_method_id, shipping_method_id,
			notes, guest_token, custom_fields,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12,
			$13, $14,
			$15, $16, $17,
			$18, $19
		)`

	// Store guest_token as NULL when empty.
	var guestTokenVal *string
	if o.GuestToken != "" {
		guestTokenVal = &o.GuestToken
	}

	_, err = tx.Exec(ctx, insertOrder,
		o.ID, o.OrderNumber, o.CustomerID, o.Status, o.Currency,
		o.SubtotalNet, o.SubtotalGross, o.ShippingCost, o.TaxTotal, o.Total,
		billingRaw, shippingRaw,
		o.PaymentMethodID, o.ShippingMethodID,
		o.Notes, guestTokenVal, customFieldsRaw,
		o.CreatedAt, o.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting order: %w", err)
	}

	for i := range o.Items {
		item := &o.Items[i]
		item.ID = uuid.New()
		item.OrderID = o.ID

		const insertItem = `
			INSERT INTO order_items (
				id, order_id, product_id, variant_id,
				sku, name, quantity,
				unit_price_net, unit_price_gross,
				total_net, total_gross, tax_rate
			) VALUES (
				$1, $2, $3, $4,
				$5, $6, $7,
				$8, $9,
				$10, $11, $12
			)`

		_, err = tx.Exec(ctx, insertItem,
			item.ID, item.OrderID, item.ProductID, item.VariantID,
			item.SKU, item.Name, item.Quantity,
			item.UnitPriceNet, item.UnitPriceGross,
			item.TotalNet, item.TotalGross, item.TaxRate,
		)
		if err != nil {
			return fmt.Errorf("inserting order item: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing order transaction: %w", err)
	}

	return nil
}

// -------------------------------------------------------------------
// Update
// -------------------------------------------------------------------

func (r *PostgresRepository) Update(ctx context.Context, o *Order) error {
	billingRaw, err := marshalJSONB(o.BillingAddress)
	if err != nil {
		return fmt.Errorf("marshalling billing address: %w", err)
	}
	shippingRaw, err := marshalJSONB(o.ShippingAddress)
	if err != nil {
		return fmt.Errorf("marshalling shipping address: %w", err)
	}
	customFieldsRaw, err := marshalJSONB(o.CustomFields)
	if err != nil {
		return fmt.Errorf("marshalling custom fields: %w", err)
	}

	o.UpdatedAt = time.Now().UTC()

	const q = `
		UPDATE orders SET
			status = $2,
			billing_address = $3,
			shipping_address = $4,
			payment_method_id = $5,
			shipping_method_id = $6,
			notes = $7,
			custom_fields = $8,
			updated_at = $9
		WHERE id = $1`

	ct, err := r.db.Exec(ctx, q,
		o.ID, o.Status,
		billingRaw, shippingRaw,
		o.PaymentMethodID, o.ShippingMethodID,
		o.Notes, customFieldsRaw,
		o.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("updating order: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("order %s not found", o.ID)
	}

	return nil
}

// -------------------------------------------------------------------
// UpdateStatus
// -------------------------------------------------------------------

func (r *PostgresRepository) UpdateStatus(ctx context.Context, id uuid.UUID, fromStatus, toStatus, comment string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	now := time.Now().UTC()

	const updateOrder = `
		UPDATE orders SET status = $2, updated_at = $3
		WHERE id = $1`

	ct, err := tx.Exec(ctx, updateOrder, id, toStatus, now)
	if err != nil {
		return fmt.Errorf("updating order status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("order %s not found", id)
	}

	const insertHistory = `
		INSERT INTO order_status_history (id, order_id, from_status, to_status, comment, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err = tx.Exec(ctx, insertHistory,
		uuid.New(), id, fromStatus, toStatus, comment, now,
	)
	if err != nil {
		return fmt.Errorf("inserting status history: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing status update transaction: %w", err)
	}

	return nil
}

// -------------------------------------------------------------------
// Internal helpers
// -------------------------------------------------------------------

func (r *PostgresRepository) findItemsByOrderID(ctx context.Context, orderID uuid.UUID) ([]OrderItem, error) {
	const q = `
		SELECT
			id, order_id, product_id, variant_id,
			sku, name, quantity,
			unit_price_net, unit_price_gross,
			total_net, total_gross, tax_rate
		FROM order_items
		WHERE order_id = $1
		ORDER BY id`

	rows, err := r.db.Query(ctx, q, orderID)
	if err != nil {
		return nil, fmt.Errorf("querying order items: %w", err)
	}
	defer rows.Close()

	var items []OrderItem
	for rows.Next() {
		var item OrderItem
		if err := rows.Scan(
			&item.ID, &item.OrderID, &item.ProductID, &item.VariantID,
			&item.SKU, &item.Name, &item.Quantity,
			&item.UnitPriceNet, &item.UnitPriceGross,
			&item.TotalNet, &item.TotalGross, &item.TaxRate,
		); err != nil {
			return nil, fmt.Errorf("scanning order item: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PostgresRepository) findStatusHistoryByOrderID(ctx context.Context, orderID uuid.UUID) ([]OrderStatusHistory, error) {
	const q = `
		SELECT id, order_id, from_status, to_status, comment, created_at
		FROM order_status_history
		WHERE order_id = $1
		ORDER BY created_at ASC`

	rows, err := r.db.Query(ctx, q, orderID)
	if err != nil {
		return nil, fmt.Errorf("querying status history: %w", err)
	}
	defer rows.Close()

	var history []OrderStatusHistory
	for rows.Next() {
		var h OrderStatusHistory
		if err := rows.Scan(&h.ID, &h.OrderID, &h.FromStatus, &h.ToStatus, &h.Comment, &h.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning status history: %w", err)
		}
		history = append(history, h)
	}
	return history, rows.Err()
}

func (r *PostgresRepository) scanRows(rows pgx.Rows) ([]Order, error) {
	var orders []Order
	for rows.Next() {
		var o Order
		var billingRaw, shippingRaw, customFieldsRaw []byte

		if err := rows.Scan(
			&o.ID, &o.OrderNumber, &o.CustomerID, &o.Status, &o.Currency,
			&o.SubtotalNet, &o.SubtotalGross, &o.ShippingCost, &o.TaxTotal, &o.Total,
			&billingRaw, &shippingRaw,
			&o.PaymentMethodID, &o.ShippingMethodID,
			&o.Notes, &customFieldsRaw,
			&o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning order row: %w", err)
		}

		if err := unmarshalJSONB(billingRaw, &o.BillingAddress); err != nil {
			return nil, fmt.Errorf("unmarshalling billing address: %w", err)
		}
		if err := unmarshalJSONB(shippingRaw, &o.ShippingAddress); err != nil {
			return nil, fmt.Errorf("unmarshalling shipping address: %w", err)
		}
		if err := unmarshalJSONB(customFieldsRaw, &o.CustomFields); err != nil {
			return nil, fmt.Errorf("unmarshalling custom fields: %w", err)
		}

		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func marshalJSONB(v map[string]interface{}) ([]byte, error) {
	if v == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(v)
}

func unmarshalJSONB(data []byte, v *map[string]interface{}) error {
	if len(data) == 0 || string(data) == "null" {
		*v = nil
		return nil
	}
	return json.Unmarshal(data, v)
}
