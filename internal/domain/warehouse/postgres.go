package warehouse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type postgresRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewPostgresRepository creates a new PostgreSQL-backed WarehouseRepository.
func NewPostgresRepository(db *pgxpool.Pool, logger zerolog.Logger) WarehouseRepository {
	return &postgresRepository{db: db, logger: logger}
}

// --------------------------------------------------------------------------
// Warehouse CRUD
// --------------------------------------------------------------------------

func (r *postgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*Warehouse, error) {
	const q = `
		SELECT id, name, code, active, allow_negative_stock, priority,
		       address_line1, address_line2, city, state, postal_code, country,
		       custom_fields, metadata, created_at, updated_at
		FROM warehouses
		WHERE id = $1`

	w := &Warehouse{}
	var cfRaw, mdRaw []byte
	err := r.db.QueryRow(ctx, q, id).Scan(
		&w.ID, &w.Name, &w.Code, &w.Active, &w.AllowNegativeStock, &w.Priority,
		&w.AddressLine1, &w.AddressLine2, &w.City, &w.State, &w.PostalCode, &w.Country,
		&cfRaw, &mdRaw, &w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("warehouse: FindByID: %w", err)
	}
	if len(cfRaw) > 0 {
		_ = json.Unmarshal(cfRaw, &w.CustomFields)
	}
	if len(mdRaw) > 0 {
		_ = json.Unmarshal(mdRaw, &w.Metadata)
	}
	return w, nil
}

func (r *postgresRepository) FindAll(ctx context.Context, filter WarehouseFilter) ([]Warehouse, int, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 200 {
		filter.Limit = 20
	}
	offset := (filter.Page - 1) * filter.Limit

	args := []interface{}{}
	where := "WHERE 1=1"
	idx := 1

	if filter.Active != nil {
		where += fmt.Sprintf(" AND active = $%d", idx)
		args = append(args, *filter.Active)
		idx++
	}

	countQ := fmt.Sprintf("SELECT COUNT(*) FROM warehouses %s", where)
	var total int
	if err := r.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("warehouse: FindAll count: %w", err)
	}

	args = append(args, filter.Limit, offset)
	dataQ := fmt.Sprintf(`
		SELECT id, name, code, active, allow_negative_stock, priority,
		       address_line1, address_line2, city, state, postal_code, country,
		       custom_fields, metadata, created_at, updated_at
		FROM warehouses %s
		ORDER BY priority ASC, name ASC
		LIMIT $%d OFFSET $%d`, where, idx, idx+1)

	rows, err := r.db.Query(ctx, dataQ, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("warehouse: FindAll: %w", err)
	}
	defer rows.Close()

	var warehouses []Warehouse
	for rows.Next() {
		var w Warehouse
		var cfRaw, mdRaw []byte
		if err := rows.Scan(
			&w.ID, &w.Name, &w.Code, &w.Active, &w.AllowNegativeStock, &w.Priority,
			&w.AddressLine1, &w.AddressLine2, &w.City, &w.State, &w.PostalCode, &w.Country,
			&cfRaw, &mdRaw, &w.CreatedAt, &w.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("warehouse: FindAll scan: %w", err)
		}
		if len(cfRaw) > 0 {
			_ = json.Unmarshal(cfRaw, &w.CustomFields)
		}
		if len(mdRaw) > 0 {
			_ = json.Unmarshal(mdRaw, &w.Metadata)
		}
		warehouses = append(warehouses, w)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("warehouse: FindAll rows: %w", err)
	}
	return warehouses, total, nil
}

func (r *postgresRepository) Create(ctx context.Context, w *Warehouse) error {
	cfJSON, _ := json.Marshal(w.CustomFields)
	mdJSON, _ := json.Marshal(w.Metadata)

	const q = `
		INSERT INTO warehouses (id, name, code, active, allow_negative_stock, priority,
		    address_line1, address_line2, city, state, postal_code, country,
		    custom_fields, metadata, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`

	_, err := r.db.Exec(ctx, q,
		w.ID, w.Name, w.Code, w.Active, w.AllowNegativeStock, w.Priority,
		w.AddressLine1, w.AddressLine2, w.City, w.State, w.PostalCode, w.Country,
		cfJSON, mdJSON, w.CreatedAt, w.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "warehouses_code_key") {
			return ErrDuplicateCode
		}
		return fmt.Errorf("warehouse: Create: %w", err)
	}
	return nil
}

func (r *postgresRepository) Update(ctx context.Context, w *Warehouse) error {
	cfJSON, _ := json.Marshal(w.CustomFields)
	mdJSON, _ := json.Marshal(w.Metadata)

	const q = `
		UPDATE warehouses
		SET name = $2, code = $3, active = $4, allow_negative_stock = $5, priority = $6,
		    address_line1 = $7, address_line2 = $8, city = $9, state = $10,
		    postal_code = $11, country = $12, custom_fields = $13, metadata = $14,
		    updated_at = $15
		WHERE id = $1`

	ct, err := r.db.Exec(ctx, q,
		w.ID, w.Name, w.Code, w.Active, w.AllowNegativeStock, w.Priority,
		w.AddressLine1, w.AddressLine2, w.City, w.State, w.PostalCode, w.Country,
		cfJSON, mdJSON, w.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "warehouses_code_key") {
			return ErrDuplicateCode
		}
		return fmt.Errorf("warehouse: Update: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM warehouses WHERE id = $1`
	ct, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("warehouse: Delete: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --------------------------------------------------------------------------
// Stock operations
// --------------------------------------------------------------------------

func (r *postgresRepository) SetStock(ctx context.Context, warehouseID, productID uuid.UUID, variantID *uuid.UUID, quantity int, reference string) (*WarehouseStock, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("warehouse: SetStock begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Get current quantity (0 if no row exists).
	var currentQty int
	var stockID uuid.UUID
	var scanErr error

	if variantID != nil {
		scanErr = tx.QueryRow(ctx,
			`SELECT id, quantity FROM warehouse_stock WHERE warehouse_id = $1 AND product_id = $2 AND variant_id = $3`,
			warehouseID, productID, *variantID,
		).Scan(&stockID, &currentQty)
	} else {
		scanErr = tx.QueryRow(ctx,
			`SELECT id, quantity FROM warehouse_stock WHERE warehouse_id = $1 AND product_id = $2 AND variant_id IS NULL`,
			warehouseID, productID,
		).Scan(&stockID, &currentQty)
	}

	if scanErr != nil && !errors.Is(scanErr, pgx.ErrNoRows) {
		return nil, fmt.Errorf("warehouse: SetStock query current: %w", scanErr)
	}

	// Upsert stock.
	ws := &WarehouseStock{
		WarehouseID: warehouseID,
		ProductID:   productID,
		VariantID:   variantID,
		Quantity:    quantity,
	}

	if errors.Is(scanErr, pgx.ErrNoRows) {
		ws.ID = uuid.New()
		if variantID != nil {
			_, err = tx.Exec(ctx,
				`INSERT INTO warehouse_stock (id, warehouse_id, product_id, variant_id, quantity) VALUES ($1,$2,$3,$4,$5)`,
				ws.ID, warehouseID, productID, *variantID, quantity,
			)
		} else {
			_, err = tx.Exec(ctx,
				`INSERT INTO warehouse_stock (id, warehouse_id, product_id, quantity) VALUES ($1,$2,$3,$4)`,
				ws.ID, warehouseID, productID, quantity,
			)
		}
	} else {
		ws.ID = stockID
		_, err = tx.Exec(ctx,
			`UPDATE warehouse_stock SET quantity = $2, updated_at = now() WHERE id = $1`,
			stockID, quantity,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("warehouse: SetStock upsert: %w", err)
	}

	// Record adjustment movement.
	delta := quantity - currentQty
	if delta != 0 {
		_, err = tx.Exec(ctx,
			`INSERT INTO stock_movements (id, warehouse_id, product_id, variant_id, movement_type, quantity, reference)
			 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			uuid.New(), warehouseID, productID, variantID, MovementAdjustment, delta, reference,
		)
		if err != nil {
			return nil, fmt.Errorf("warehouse: SetStock movement: %w", err)
		}
	}

	// Update denormalized stock on product/variant.
	if err := r.syncDenormalizedStock(ctx, tx, productID, variantID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("warehouse: SetStock commit: %w", err)
	}
	return ws, nil
}

func (r *postgresRepository) GetStockByWarehouse(ctx context.Context, warehouseID uuid.UUID) ([]WarehouseStock, error) {
	const q = `
		SELECT ws.id, ws.warehouse_id, ws.product_id, ws.variant_id, ws.quantity,
		       ws.created_at, ws.updated_at, w.name, w.code,
		       COALESCE(p.sku, '') AS product_sku,
		       COALESCE(pt.name, '') AS product_name,
		       COALESCE(pv.sku, '') AS variant_sku
		FROM warehouse_stock ws
		JOIN warehouses w ON w.id = ws.warehouse_id
		JOIN products p ON p.id = ws.product_id
		LEFT JOIN LATERAL (
		    SELECT name FROM product_translations
		    WHERE product_id = p.id
		    ORDER BY locale ASC LIMIT 1
		) pt ON true
		LEFT JOIN product_variants pv ON pv.id = ws.variant_id
		WHERE ws.warehouse_id = $1
		ORDER BY p.sku, pv.sku NULLS FIRST`

	rows, err := r.db.Query(ctx, q, warehouseID)
	if err != nil {
		return nil, fmt.Errorf("warehouse: GetStockByWarehouse: %w", err)
	}
	defer rows.Close()
	return scanStockRows(rows)
}

func (r *postgresRepository) GetStockByProduct(ctx context.Context, productID uuid.UUID) ([]WarehouseStock, error) {
	const q = `
		SELECT ws.id, ws.warehouse_id, ws.product_id, ws.variant_id, ws.quantity,
		       ws.created_at, ws.updated_at, w.name, w.code,
		       COALESCE(p.sku, '') AS product_sku,
		       COALESCE(pt.name, '') AS product_name,
		       COALESCE(pv.sku, '') AS variant_sku
		FROM warehouse_stock ws
		JOIN warehouses w ON w.id = ws.warehouse_id
		JOIN products p ON p.id = ws.product_id
		LEFT JOIN LATERAL (
		    SELECT name FROM product_translations
		    WHERE product_id = p.id
		    ORDER BY locale ASC LIMIT 1
		) pt ON true
		LEFT JOIN product_variants pv ON pv.id = ws.variant_id
		WHERE ws.product_id = $1
		ORDER BY w.priority ASC, w.name ASC`

	rows, err := r.db.Query(ctx, q, productID)
	if err != nil {
		return nil, fmt.Errorf("warehouse: GetStockByProduct: %w", err)
	}
	defer rows.Close()
	return scanStockRows(rows)
}

func scanStockRows(rows pgx.Rows) ([]WarehouseStock, error) {
	var stocks []WarehouseStock
	for rows.Next() {
		var s WarehouseStock
		if err := rows.Scan(
			&s.ID, &s.WarehouseID, &s.ProductID, &s.VariantID, &s.Quantity,
			&s.CreatedAt, &s.UpdatedAt, &s.WarehouseName, &s.WarehouseCode,
			&s.ProductSKU, &s.ProductName, &s.VariantSKU,
		); err != nil {
			return nil, fmt.Errorf("warehouse: scanStockRows: %w", err)
		}
		stocks = append(stocks, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("warehouse: scanStockRows rows: %w", err)
	}
	return stocks, nil
}

// DeductStock deducts inventory using priority-based warehouse selection.
// Each item is fulfilled by drawing from warehouses in priority order (lowest first).
// All operations run in a single transaction with row-level locking.
func (r *postgresRepository) DeductStock(ctx context.Context, items []StockDeductionItem) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("warehouse: DeductStock begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	for _, item := range items {
		remaining := item.Quantity

		// Load warehouse stock rows ordered by priority, with row-level locking.
		var stockRows pgx.Rows
		if item.VariantID != nil {
			stockRows, err = tx.Query(ctx, `
				SELECT ws.id, ws.warehouse_id, ws.quantity
				FROM warehouse_stock ws
				JOIN warehouses w ON w.id = ws.warehouse_id
				WHERE ws.product_id = $1 AND ws.variant_id = $2 AND w.active = true AND ws.quantity > 0
				ORDER BY w.priority ASC
				FOR UPDATE OF ws`,
				item.ProductID, *item.VariantID,
			)
		} else {
			stockRows, err = tx.Query(ctx, `
				SELECT ws.id, ws.warehouse_id, ws.quantity
				FROM warehouse_stock ws
				JOIN warehouses w ON w.id = ws.warehouse_id
				WHERE ws.product_id = $1 AND ws.variant_id IS NULL AND w.active = true AND ws.quantity > 0
				ORDER BY w.priority ASC
				FOR UPDATE OF ws`,
				item.ProductID,
			)
		}
		if err != nil {
			return fmt.Errorf("warehouse: DeductStock query: %w", err)
		}

		type stockRow struct {
			id          uuid.UUID
			warehouseID uuid.UUID
			quantity    int
		}
		var available []stockRow
		for stockRows.Next() {
			var sr stockRow
			if err := stockRows.Scan(&sr.id, &sr.warehouseID, &sr.quantity); err != nil {
				stockRows.Close()
				return fmt.Errorf("warehouse: DeductStock scan: %w", err)
			}
			available = append(available, sr)
		}
		stockRows.Close()
		if err := stockRows.Err(); err != nil {
			return fmt.Errorf("warehouse: DeductStock rows: %w", err)
		}

		// Deduct from warehouses in priority order.
		for _, sr := range available {
			if remaining <= 0 {
				break
			}

			deduct := sr.quantity
			if deduct > remaining {
				deduct = remaining
			}

			_, err = tx.Exec(ctx,
				`UPDATE warehouse_stock SET quantity = quantity - $2, updated_at = now() WHERE id = $1`,
				sr.id, deduct,
			)
			if err != nil {
				return fmt.Errorf("warehouse: DeductStock update: %w", err)
			}

			// Record sale movement.
			_, err = tx.Exec(ctx,
				`INSERT INTO stock_movements (id, warehouse_id, product_id, variant_id, order_id, movement_type, quantity, reference)
				 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
				uuid.New(), sr.warehouseID, item.ProductID, item.VariantID, item.OrderID,
				MovementSale, -deduct, fmt.Sprintf("order:%s", item.OrderID),
			)
			if err != nil {
				return fmt.Errorf("warehouse: DeductStock movement: %w", err)
			}

			remaining -= deduct
		}

		if remaining > 0 {
			// Check whether any active warehouse allows negative stock for this product.
			allowed, err := r.anyWarehouseAllowsNegativeTx(ctx, tx, item.ProductID, item.VariantID)
			if err != nil {
				return err
			}
			if !allowed {
				return ErrInsufficientStock
			}

			// Find the highest-priority active warehouse with allow_negative_stock=true
			// and apply the remaining deduction there (quantity may go negative).
			var negStockID uuid.UUID
			var negWarehouseID uuid.UUID
			var negStockQuery string
			if item.VariantID != nil {
				negStockQuery = `
					SELECT ws.id, ws.warehouse_id
					FROM warehouse_stock ws
					JOIN warehouses w ON w.id = ws.warehouse_id
					WHERE ws.product_id = $1 AND ws.variant_id = $2
					  AND w.active = true AND w.allow_negative_stock = true
					ORDER BY w.priority ASC
					LIMIT 1
					FOR UPDATE OF ws`
				err = tx.QueryRow(ctx, negStockQuery, item.ProductID, *item.VariantID).Scan(&negStockID, &negWarehouseID)
			} else {
				negStockQuery = `
					SELECT ws.id, ws.warehouse_id
					FROM warehouse_stock ws
					JOIN warehouses w ON w.id = ws.warehouse_id
					WHERE ws.product_id = $1 AND ws.variant_id IS NULL
					  AND w.active = true AND w.allow_negative_stock = true
					ORDER BY w.priority ASC
					LIMIT 1
					FOR UPDATE OF ws`
				err = tx.QueryRow(ctx, negStockQuery, item.ProductID).Scan(&negStockID, &negWarehouseID)
			}
			if err != nil {
				return fmt.Errorf("warehouse: DeductStock neg-stock query: %w", err)
			}

			_, err = tx.Exec(ctx,
				`UPDATE warehouse_stock SET quantity = quantity - $2, updated_at = now() WHERE id = $1`,
				negStockID, remaining,
			)
			if err != nil {
				return fmt.Errorf("warehouse: DeductStock neg-stock update: %w", err)
			}

			_, err = tx.Exec(ctx,
				`INSERT INTO stock_movements (id, warehouse_id, product_id, variant_id, order_id, movement_type, quantity, reference)
				 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
				uuid.New(), negWarehouseID, item.ProductID, item.VariantID, item.OrderID,
				MovementSale, -remaining, fmt.Sprintf("order:%s", item.OrderID),
			)
			if err != nil {
				return fmt.Errorf("warehouse: DeductStock neg-stock movement: %w", err)
			}
		}

		// Sync denormalized stock.
		if err := r.syncDenormalizedStock(ctx, tx, item.ProductID, item.VariantID); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("warehouse: DeductStock commit: %w", err)
	}
	return nil
}

// RestoreStock reverses all sale movements for a given order.
func (r *postgresRepository) RestoreStock(ctx context.Context, orderID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("warehouse: RestoreStock begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Find all sale movements for this order.
	rows, err := tx.Query(ctx, `
		SELECT id, warehouse_id, product_id, variant_id, quantity
		FROM stock_movements
		WHERE order_id = $1 AND movement_type = $2`,
		orderID, MovementSale,
	)
	if err != nil {
		return fmt.Errorf("warehouse: RestoreStock query: %w", err)
	}

	type movement struct {
		id          uuid.UUID
		warehouseID uuid.UUID
		productID   uuid.UUID
		variantID   *uuid.UUID
		quantity    int // negative for sales
	}
	var movements []movement
	for rows.Next() {
		var m movement
		if err := rows.Scan(&m.id, &m.warehouseID, &m.productID, &m.variantID, &m.quantity); err != nil {
			rows.Close()
			return fmt.Errorf("warehouse: RestoreStock scan: %w", err)
		}
		movements = append(movements, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("warehouse: RestoreStock rows: %w", err)
	}

	// Reverse each movement.
	productsToSync := make(map[string]struct{ productID uuid.UUID; variantID *uuid.UUID })
	for _, m := range movements {
		restoreQty := -m.quantity // original was negative, so restore is positive

		// Update warehouse_stock.
		if m.variantID != nil {
			_, err = tx.Exec(ctx,
				`UPDATE warehouse_stock SET quantity = quantity + $3, updated_at = now()
				 WHERE warehouse_id = $1 AND product_id = $2 AND variant_id = $4`,
				m.warehouseID, m.productID, restoreQty, *m.variantID,
			)
		} else {
			_, err = tx.Exec(ctx,
				`UPDATE warehouse_stock SET quantity = quantity + $3, updated_at = now()
				 WHERE warehouse_id = $1 AND product_id = $2 AND variant_id IS NULL`,
				m.warehouseID, m.productID, restoreQty,
			)
		}
		if err != nil {
			return fmt.Errorf("warehouse: RestoreStock update stock: %w", err)
		}

		// Record return movement.
		_, err = tx.Exec(ctx,
			`INSERT INTO stock_movements (id, warehouse_id, product_id, variant_id, order_id, movement_type, quantity, reference)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			uuid.New(), m.warehouseID, m.productID, m.variantID, orderID,
			MovementReturn, restoreQty, fmt.Sprintf("restore:order:%s", orderID),
		)
		if err != nil {
			return fmt.Errorf("warehouse: RestoreStock movement: %w", err)
		}

		key := m.productID.String()
		if m.variantID != nil {
			key += ":" + m.variantID.String()
		}
		productsToSync[key] = struct{ productID uuid.UUID; variantID *uuid.UUID }{m.productID, m.variantID}
	}

	// Sync denormalized stock for all affected products/variants.
	for _, p := range productsToSync {
		if err := r.syncDenormalizedStock(ctx, tx, p.productID, p.variantID); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("warehouse: RestoreStock commit: %w", err)
	}
	return nil
}

func (r *postgresRepository) RemoveStock(ctx context.Context, stockID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("warehouse: RemoveStock begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Look up the stock entry to get product/variant/quantity info.
	var productID uuid.UUID
	var variantID *uuid.UUID
	var warehouseID uuid.UUID
	var quantity int
	err = tx.QueryRow(ctx,
		`SELECT warehouse_id, product_id, variant_id, quantity FROM warehouse_stock WHERE id = $1`,
		stockID,
	).Scan(&warehouseID, &productID, &variantID, &quantity)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("warehouse: RemoveStock lookup: %w", err)
	}

	// Delete the stock entry.
	_, err = tx.Exec(ctx, `DELETE FROM warehouse_stock WHERE id = $1`, stockID)
	if err != nil {
		return fmt.Errorf("warehouse: RemoveStock delete: %w", err)
	}

	// Record adjustment movement (negative of current quantity).
	if quantity > 0 {
		_, err = tx.Exec(ctx,
			`INSERT INTO stock_movements (id, warehouse_id, product_id, variant_id, movement_type, quantity, reference)
			 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			uuid.New(), warehouseID, productID, variantID, MovementAdjustment, -quantity, "admin-remove",
		)
		if err != nil {
			return fmt.Errorf("warehouse: RemoveStock movement: %w", err)
		}
	}

	// Update denormalized stock.
	if err := r.syncDenormalizedStock(ctx, tx, productID, variantID); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("warehouse: RemoveStock commit: %w", err)
	}
	return nil
}

func (r *postgresRepository) AggregateStock(ctx context.Context, productID uuid.UUID, variantID *uuid.UUID) (int, error) {
	var total int
	var err error

	if variantID != nil {
		err = r.db.QueryRow(ctx, `
			SELECT COALESCE(SUM(ws.quantity), 0)
			FROM warehouse_stock ws
			JOIN warehouses w ON w.id = ws.warehouse_id
			WHERE ws.product_id = $1 AND ws.variant_id = $2 AND w.active = true`,
			productID, *variantID,
		).Scan(&total)
	} else {
		err = r.db.QueryRow(ctx, `
			SELECT COALESCE(SUM(ws.quantity), 0)
			FROM warehouse_stock ws
			JOIN warehouses w ON w.id = ws.warehouse_id
			WHERE ws.product_id = $1 AND ws.variant_id IS NULL AND w.active = true`,
			productID,
		).Scan(&total)
	}
	if err != nil {
		return 0, fmt.Errorf("warehouse: AggregateStock: %w", err)
	}
	return total, nil
}

func (r *postgresRepository) AnyWarehouseAllowsNegative(ctx context.Context, productID uuid.UUID, variantID *uuid.UUID) (bool, error) {
	return r.anyWarehouseAllowsNegativeDB(ctx, productID, variantID)
}

// anyWarehouseAllowsNegativeTx checks inside a transaction.
func (r *postgresRepository) anyWarehouseAllowsNegativeTx(ctx context.Context, tx pgx.Tx, productID uuid.UUID, variantID *uuid.UUID) (bool, error) {
	var exists bool
	var err error
	if variantID != nil {
		err = tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM warehouse_stock ws
				JOIN warehouses w ON w.id = ws.warehouse_id
				WHERE ws.product_id = $1 AND ws.variant_id = $2
				  AND w.active = true AND w.allow_negative_stock = true
			)`, productID, *variantID,
		).Scan(&exists)
	} else {
		err = tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM warehouse_stock ws
				JOIN warehouses w ON w.id = ws.warehouse_id
				WHERE ws.product_id = $1 AND ws.variant_id IS NULL
				  AND w.active = true AND w.allow_negative_stock = true
			)`, productID,
		).Scan(&exists)
	}
	if err != nil {
		return false, fmt.Errorf("warehouse: anyWarehouseAllowsNegative: %w", err)
	}
	return exists, nil
}

// anyWarehouseAllowsNegativeDB checks outside a transaction.
func (r *postgresRepository) anyWarehouseAllowsNegativeDB(ctx context.Context, productID uuid.UUID, variantID *uuid.UUID) (bool, error) {
	var exists bool
	var err error
	if variantID != nil {
		err = r.db.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM warehouse_stock ws
				JOIN warehouses w ON w.id = ws.warehouse_id
				WHERE ws.product_id = $1 AND ws.variant_id = $2
				  AND w.active = true AND w.allow_negative_stock = true
			)`, productID, *variantID,
		).Scan(&exists)
	} else {
		err = r.db.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM warehouse_stock ws
				JOIN warehouses w ON w.id = ws.warehouse_id
				WHERE ws.product_id = $1 AND ws.variant_id IS NULL
				  AND w.active = true AND w.allow_negative_stock = true
			)`, productID,
		).Scan(&exists)
	}
	if err != nil {
		return false, fmt.Errorf("warehouse: anyWarehouseAllowsNegative: %w", err)
	}
	return exists, nil
}

// syncDenormalizedStock updates the stock field on products or product_variants
// to match the aggregate of all warehouse_stock entries.
// tx is a pgx.Tx to run within the caller's transaction.
func (r *postgresRepository) syncDenormalizedStock(ctx context.Context, tx pgx.Tx, productID uuid.UUID, variantID *uuid.UUID) error {
	if variantID != nil {
		_, err := tx.Exec(ctx, `
			UPDATE product_variants
			SET stock = COALESCE((
				SELECT SUM(ws.quantity)
				FROM warehouse_stock ws
				JOIN warehouses w ON w.id = ws.warehouse_id
				WHERE ws.product_id = $2 AND ws.variant_id = $1 AND w.active = true
			), 0)
			WHERE id = $1`,
			*variantID, productID,
		)
		if err != nil {
			return fmt.Errorf("warehouse: syncDenormalizedStock variant: %w", err)
		}
	} else {
		_, err := tx.Exec(ctx, `
			UPDATE products
			SET stock = COALESCE((
				SELECT SUM(ws.quantity)
				FROM warehouse_stock ws
				JOIN warehouses w ON w.id = ws.warehouse_id
				WHERE ws.product_id = $1 AND ws.variant_id IS NULL AND w.active = true
			), 0)
			WHERE id = $1`,
			productID,
		)
		if err != nil {
			return fmt.Errorf("warehouse: syncDenormalizedStock product: %w", err)
		}
	}
	return nil
}
