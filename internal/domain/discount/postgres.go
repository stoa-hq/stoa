package discount

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type postgresRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewPostgresRepository creates a new PostgreSQL-backed DiscountRepository.
func NewPostgresRepository(db *pgxpool.Pool, logger zerolog.Logger) DiscountRepository {
	return &postgresRepository{db: db, logger: logger}
}

const discountColumns = `id, code, type, value, min_order_value, max_uses, used_count,
	valid_from, valid_until, active, conditions, created_at, updated_at`

func scanDiscount(row pgx.Row) (*Discount, error) {
	d := &Discount{}
	var conditionsRaw []byte
	err := row.Scan(
		&d.ID, &d.Code, &d.Type, &d.Value, &d.MinOrderValue, &d.MaxUses, &d.UsedCount,
		&d.ValidFrom, &d.ValidUntil, &d.Active, &conditionsRaw, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(conditionsRaw) > 0 {
		if err := json.Unmarshal(conditionsRaw, &d.Conditions); err != nil {
			return nil, fmt.Errorf("discount: unmarshal conditions: %w", err)
		}
	}
	return d, nil
}

func (r *postgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*Discount, error) {
	q := fmt.Sprintf(`SELECT %s FROM discounts WHERE id = $1`, discountColumns)
	d, err := scanDiscount(r.db.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("discount: FindByID: %w", err)
	}
	return d, nil
}

func (r *postgresRepository) FindByCode(ctx context.Context, code string) (*Discount, error) {
	q := fmt.Sprintf(`SELECT %s FROM discounts WHERE code = $1`, discountColumns)
	d, err := scanDiscount(r.db.QueryRow(ctx, q, code))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("discount: FindByCode: %w", err)
	}
	return d, nil
}

func (r *postgresRepository) FindAll(ctx context.Context, filter DiscountFilter) ([]Discount, int, error) {
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
	if filter.Type != "" {
		where += fmt.Sprintf(" AND type = $%d", idx)
		args = append(args, filter.Type)
		idx++
	}
	if filter.Code != "" {
		where += fmt.Sprintf(" AND code ILIKE $%d", idx)
		args = append(args, "%"+filter.Code+"%")
		idx++
	}

	countQ := fmt.Sprintf("SELECT COUNT(*) FROM discounts %s", where)
	var total int
	if err := r.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("discount: FindAll count: %w", err)
	}

	args = append(args, filter.Limit, offset)
	dataQ := fmt.Sprintf(`SELECT %s FROM discounts %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		discountColumns, where, idx, idx+1)

	rows, err := r.db.Query(ctx, dataQ, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("discount: FindAll: %w", err)
	}
	defer rows.Close()

	var discounts []Discount
	for rows.Next() {
		var conditionsRaw []byte
		d := Discount{}
		if err := rows.Scan(
			&d.ID, &d.Code, &d.Type, &d.Value, &d.MinOrderValue, &d.MaxUses, &d.UsedCount,
			&d.ValidFrom, &d.ValidUntil, &d.Active, &conditionsRaw, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("discount: FindAll scan: %w", err)
		}
		if len(conditionsRaw) > 0 {
			if err := json.Unmarshal(conditionsRaw, &d.Conditions); err != nil {
				return nil, 0, fmt.Errorf("discount: FindAll unmarshal conditions: %w", err)
			}
		}
		discounts = append(discounts, d)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("discount: FindAll rows: %w", err)
	}
	return discounts, total, nil
}

func (r *postgresRepository) Create(ctx context.Context, d *Discount) error {
	conditionsJSON, err := json.Marshal(d.Conditions)
	if err != nil {
		return fmt.Errorf("discount: Create marshal conditions: %w", err)
	}

	const q = `
		INSERT INTO discounts
			(id, code, type, value, min_order_value, max_uses, used_count,
			 valid_from, valid_until, active, conditions, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	_, err = r.db.Exec(ctx, q,
		d.ID, d.Code, d.Type, d.Value, d.MinOrderValue, d.MaxUses, d.UsedCount,
		d.ValidFrom, d.ValidUntil, d.Active, conditionsJSON, d.CreatedAt, d.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("discount: Create: %w", err)
	}
	return nil
}

func (r *postgresRepository) Update(ctx context.Context, d *Discount) error {
	conditionsJSON, err := json.Marshal(d.Conditions)
	if err != nil {
		return fmt.Errorf("discount: Update marshal conditions: %w", err)
	}

	const q = `
		UPDATE discounts
		SET code = $2, type = $3, value = $4, min_order_value = $5, max_uses = $6,
		    valid_from = $7, valid_until = $8, active = $9, conditions = $10, updated_at = $11
		WHERE id = $1`

	ct, err := r.db.Exec(ctx, q,
		d.ID, d.Code, d.Type, d.Value, d.MinOrderValue, d.MaxUses,
		d.ValidFrom, d.ValidUntil, d.Active, conditionsJSON, d.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("discount: Update: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM discounts WHERE id = $1`
	ct, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("discount: Delete: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresRepository) IncrementUsedCount(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE discounts SET used_count = used_count + 1, updated_at = NOW() WHERE id = $1`
	ct, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("discount: IncrementUsedCount: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
