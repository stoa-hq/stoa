package shipping

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

// NewPostgresRepository creates a new PostgreSQL-backed ShippingMethodRepository.
func NewPostgresRepository(db *pgxpool.Pool, logger zerolog.Logger) ShippingMethodRepository {
	return &postgresRepository{db: db, logger: logger}
}

func (r *postgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*ShippingMethod, error) {
	const q = `
		SELECT id, active, price_net, price_gross, tax_rule_id, custom_fields, created_at, updated_at
		FROM shipping_methods
		WHERE id = $1`

	m := &ShippingMethod{}
	var cfRaw []byte
	err := r.db.QueryRow(ctx, q, id).Scan(
		&m.ID, &m.Active, &m.PriceNet, &m.PriceGross, &m.TaxRuleID, &cfRaw, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("shipping: FindByID: %w", err)
	}
	if len(cfRaw) > 0 {
		if err := json.Unmarshal(cfRaw, &m.CustomFields); err != nil {
			return nil, fmt.Errorf("shipping: FindByID unmarshal custom_fields: %w", err)
		}
	}

	translations, err := r.findTranslations(ctx, id)
	if err != nil {
		return nil, err
	}
	m.Translations = translations
	return m, nil
}

func (r *postgresRepository) FindAll(ctx context.Context, filter ShippingMethodFilter) ([]ShippingMethod, int, error) {
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

	countQ := fmt.Sprintf("SELECT COUNT(*) FROM shipping_methods %s", where)
	var total int
	if err := r.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("shipping: FindAll count: %w", err)
	}

	args = append(args, filter.Limit, offset)
	dataQ := fmt.Sprintf(`
		SELECT id, active, price_net, price_gross, tax_rule_id, custom_fields, created_at, updated_at
		FROM shipping_methods %s
		ORDER BY price_gross ASC
		LIMIT $%d OFFSET $%d`, where, idx, idx+1)

	rows, err := r.db.Query(ctx, dataQ, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("shipping: FindAll: %w", err)
	}
	defer rows.Close()

	var methods []ShippingMethod
	for rows.Next() {
		var m ShippingMethod
		var cfRaw []byte
		if err := rows.Scan(
			&m.ID, &m.Active, &m.PriceNet, &m.PriceGross, &m.TaxRuleID, &cfRaw, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("shipping: FindAll scan: %w", err)
		}
		if len(cfRaw) > 0 {
			if err := json.Unmarshal(cfRaw, &m.CustomFields); err != nil {
				return nil, 0, fmt.Errorf("shipping: FindAll unmarshal custom_fields: %w", err)
			}
		}
		methods = append(methods, m)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("shipping: FindAll rows: %w", err)
	}

	// Load translations for all methods.
	for i := range methods {
		translations, err := r.findTranslations(ctx, methods[i].ID)
		if err != nil {
			return nil, 0, err
		}
		methods[i].Translations = translations
	}

	return methods, total, nil
}

func (r *postgresRepository) Create(ctx context.Context, m *ShippingMethod) error {
	cfJSON, err := json.Marshal(m.CustomFields)
	if err != nil {
		return fmt.Errorf("shipping: Create marshal custom_fields: %w", err)
	}

	const q = `
		INSERT INTO shipping_methods (id, active, price_net, price_gross, custom_fields, created_at, updated_at, tax_rule_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err = r.db.Exec(ctx, q,
		m.ID, m.Active, m.PriceNet, m.PriceGross, cfJSON, m.CreatedAt, m.UpdatedAt, m.TaxRuleID,
	)
	if err != nil {
		return fmt.Errorf("shipping: Create: %w", err)
	}

	if err := r.upsertTranslations(ctx, m.ID, m.Translations); err != nil {
		return err
	}
	return nil
}

func (r *postgresRepository) Update(ctx context.Context, m *ShippingMethod) error {
	cfJSON, err := json.Marshal(m.CustomFields)
	if err != nil {
		return fmt.Errorf("shipping: Update marshal custom_fields: %w", err)
	}

	const q = `
		UPDATE shipping_methods
		SET active = $2, price_net = $3, price_gross = $4, custom_fields = $5, updated_at = $6, tax_rule_id = $7
		WHERE id = $1`

	ct, err := r.db.Exec(ctx, q,
		m.ID, m.Active, m.PriceNet, m.PriceGross, cfJSON, m.UpdatedAt, m.TaxRuleID,
	)
	if err != nil {
		return fmt.Errorf("shipping: Update: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}

	// Replace translations: delete all then re-insert.
	if _, err := r.db.Exec(ctx, `DELETE FROM shipping_method_translations WHERE shipping_method_id = $1`, m.ID); err != nil {
		return fmt.Errorf("shipping: Update delete translations: %w", err)
	}
	if err := r.upsertTranslations(ctx, m.ID, m.Translations); err != nil {
		return err
	}
	return nil
}

func (r *postgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM shipping_methods WHERE id = $1`
	ct, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("shipping: Delete: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresRepository) findTranslations(ctx context.Context, id uuid.UUID) ([]ShippingMethodTranslation, error) {
	const q = `
		SELECT shipping_method_id, locale, name, description
		FROM shipping_method_translations
		WHERE shipping_method_id = $1
		ORDER BY locale ASC`

	rows, err := r.db.Query(ctx, q, id)
	if err != nil {
		return nil, fmt.Errorf("shipping: findTranslations: %w", err)
	}
	defer rows.Close()

	var translations []ShippingMethodTranslation
	for rows.Next() {
		var t ShippingMethodTranslation
		if err := rows.Scan(&t.ShippingMethodID, &t.Locale, &t.Name, &t.Description); err != nil {
			return nil, fmt.Errorf("shipping: findTranslations scan: %w", err)
		}
		translations = append(translations, t)
	}
	return translations, rows.Err()
}

func (r *postgresRepository) upsertTranslations(ctx context.Context, id uuid.UUID, translations []ShippingMethodTranslation) error {
	const q = `
		INSERT INTO shipping_method_translations (shipping_method_id, locale, name, description)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (shipping_method_id, locale) DO UPDATE
		SET name = EXCLUDED.name, description = EXCLUDED.description`

	for _, t := range translations {
		if _, err := r.db.Exec(ctx, q, id, t.Locale, t.Name, t.Description); err != nil {
			return fmt.Errorf("shipping: upsertTranslations: %w", err)
		}
	}
	return nil
}
