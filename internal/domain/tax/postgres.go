package tax

import (
	"context"
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

// NewPostgresRepository creates a new PostgreSQL-backed TaxRuleRepository.
func NewPostgresRepository(db *pgxpool.Pool, logger zerolog.Logger) TaxRuleRepository {
	return &postgresRepository{db: db, logger: logger}
}

func (r *postgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*TaxRule, error) {
	const q = `
		SELECT id, name, rate, country_code, type, created_at, updated_at
		FROM tax_rules
		WHERE id = $1`

	t := &TaxRule{}
	err := r.db.QueryRow(ctx, q, id).Scan(
		&t.ID, &t.Name, &t.Rate, &t.CountryCode, &t.Type, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("tax: FindByID: %w", err)
	}
	return t, nil
}

func (r *postgresRepository) FindAll(ctx context.Context, filter TaxRuleFilter) ([]TaxRule, int, error) {
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

	if filter.CountryCode != "" {
		where += fmt.Sprintf(" AND country_code = $%d", idx)
		args = append(args, filter.CountryCode)
		idx++
	}
	if filter.Type != "" {
		where += fmt.Sprintf(" AND type = $%d", idx)
		args = append(args, filter.Type)
		idx++
	}

	countQ := fmt.Sprintf("SELECT COUNT(*) FROM tax_rules %s", where)
	var total int
	if err := r.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("tax: FindAll count: %w", err)
	}

	args = append(args, filter.Limit, offset)
	dataQ := fmt.Sprintf(`
		SELECT id, name, rate, country_code, type, created_at, updated_at
		FROM tax_rules
		%s
		ORDER BY name ASC
		LIMIT $%d OFFSET $%d`, where, idx, idx+1)

	rows, err := r.db.Query(ctx, dataQ, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("tax: FindAll: %w", err)
	}
	defer rows.Close()

	var rules []TaxRule
	for rows.Next() {
		var t TaxRule
		if err := rows.Scan(
			&t.ID, &t.Name, &t.Rate, &t.CountryCode, &t.Type, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("tax: FindAll scan: %w", err)
		}
		rules = append(rules, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("tax: FindAll rows: %w", err)
	}
	return rules, total, nil
}

func (r *postgresRepository) Create(ctx context.Context, t *TaxRule) error {
	const q = `
		INSERT INTO tax_rules (id, name, rate, country_code, type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.db.Exec(ctx, q,
		t.ID, t.Name, t.Rate, t.CountryCode, t.Type, t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("tax: Create: %w", err)
	}
	return nil
}

func (r *postgresRepository) Update(ctx context.Context, t *TaxRule) error {
	const q = `
		UPDATE tax_rules
		SET name = $2, rate = $3, country_code = $4, type = $5, updated_at = $6
		WHERE id = $1`

	ct, err := r.db.Exec(ctx, q,
		t.ID, t.Name, t.Rate, t.CountryCode, t.Type, t.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("tax: Update: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM tax_rules WHERE id = $1`
	ct, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("tax: Delete: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
