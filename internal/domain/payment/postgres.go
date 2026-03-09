package payment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/internal/crypto"
)

// --- PaymentMethod repository ---

type postgresMethodRepository struct {
	db            *pgxpool.Pool
	logger        zerolog.Logger
	encryptionKey []byte
}

// NewPostgresMethodRepository creates a new PostgreSQL-backed PaymentMethodRepository.
func NewPostgresMethodRepository(db *pgxpool.Pool, logger zerolog.Logger, encryptionKey []byte) PaymentMethodRepository {
	return &postgresMethodRepository{db: db, logger: logger, encryptionKey: encryptionKey}
}

func (r *postgresMethodRepository) FindByID(ctx context.Context, id uuid.UUID) (*PaymentMethod, error) {
	const q = `
		SELECT id, provider, active, config, custom_fields, created_at, updated_at
		FROM payment_methods
		WHERE id = $1`

	m := &PaymentMethod{}
	var cfRaw []byte
	err := r.db.QueryRow(ctx, q, id).Scan(
		&m.ID, &m.Provider, &m.Active, &m.Config, &cfRaw, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMethodNotFound
		}
		return nil, fmt.Errorf("payment: FindByID: %w", err)
	}
	if err := r.decryptConfig(m); err != nil {
		return nil, fmt.Errorf("payment: FindByID decrypt config: %w", err)
	}
	if len(cfRaw) > 0 {
		if err := json.Unmarshal(cfRaw, &m.CustomFields); err != nil {
			return nil, fmt.Errorf("payment: FindByID unmarshal custom_fields: %w", err)
		}
	}

	translations, err := r.findTranslations(ctx, id)
	if err != nil {
		return nil, err
	}
	m.Translations = translations
	return m, nil
}

func (r *postgresMethodRepository) FindAll(ctx context.Context, filter PaymentMethodFilter) ([]PaymentMethod, int, error) {
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

	countQ := fmt.Sprintf("SELECT COUNT(*) FROM payment_methods %s", where)
	var total int
	if err := r.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("payment: FindAll count: %w", err)
	}

	args = append(args, filter.Limit, offset)
	dataQ := fmt.Sprintf(`
		SELECT id, provider, active, config, custom_fields, created_at, updated_at
		FROM payment_methods %s
		ORDER BY created_at ASC
		LIMIT $%d OFFSET $%d`, where, idx, idx+1)

	rows, err := r.db.Query(ctx, dataQ, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("payment: FindAll: %w", err)
	}
	defer rows.Close()

	var methods []PaymentMethod
	for rows.Next() {
		var m PaymentMethod
		var cfRaw []byte
		if err := rows.Scan(
			&m.ID, &m.Provider, &m.Active, &m.Config, &cfRaw, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("payment: FindAll scan: %w", err)
		}
		if err := r.decryptConfig(&m); err != nil {
			return nil, 0, fmt.Errorf("payment: FindAll decrypt config: %w", err)
		}
		if len(cfRaw) > 0 {
			if err := json.Unmarshal(cfRaw, &m.CustomFields); err != nil {
				return nil, 0, fmt.Errorf("payment: FindAll unmarshal custom_fields: %w", err)
			}
		}
		methods = append(methods, m)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("payment: FindAll rows: %w", err)
	}

	for i := range methods {
		translations, err := r.findTranslations(ctx, methods[i].ID)
		if err != nil {
			return nil, 0, err
		}
		methods[i].Translations = translations
	}

	return methods, total, nil
}

func (r *postgresMethodRepository) Create(ctx context.Context, m *PaymentMethod) error {
	cfJSON, err := json.Marshal(m.CustomFields)
	if err != nil {
		return fmt.Errorf("payment: Create marshal custom_fields: %w", err)
	}

	encConfig, err := r.encryptConfig(m.Config)
	if err != nil {
		return fmt.Errorf("payment: Create encrypt config: %w", err)
	}

	const q = `
		INSERT INTO payment_methods (id, provider, active, config, custom_fields, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err = r.db.Exec(ctx, q,
		m.ID, m.Provider, m.Active, encConfig, cfJSON, m.CreatedAt, m.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("payment: Create: %w", err)
	}

	return r.upsertTranslations(ctx, m.ID, m.Translations)
}

func (r *postgresMethodRepository) Update(ctx context.Context, m *PaymentMethod) error {
	cfJSON, err := json.Marshal(m.CustomFields)
	if err != nil {
		return fmt.Errorf("payment: Update marshal custom_fields: %w", err)
	}

	encConfig, err := r.encryptConfig(m.Config)
	if err != nil {
		return fmt.Errorf("payment: Update encrypt config: %w", err)
	}

	const q = `
		UPDATE payment_methods
		SET provider = $2, active = $3, config = $4, custom_fields = $5, updated_at = $6
		WHERE id = $1`

	ct, err := r.db.Exec(ctx, q,
		m.ID, m.Provider, m.Active, encConfig, cfJSON, m.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("payment: Update: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrMethodNotFound
	}

	if _, err := r.db.Exec(ctx, `DELETE FROM payment_method_translations WHERE payment_method_id = $1`, m.ID); err != nil {
		return fmt.Errorf("payment: Update delete translations: %w", err)
	}
	return r.upsertTranslations(ctx, m.ID, m.Translations)
}

func (r *postgresMethodRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM payment_methods WHERE id = $1`
	ct, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("payment: Delete: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrMethodNotFound
	}
	return nil
}

func (r *postgresMethodRepository) findTranslations(ctx context.Context, id uuid.UUID) ([]PaymentMethodTranslation, error) {
	const q = `
		SELECT payment_method_id, locale, name, description
		FROM payment_method_translations
		WHERE payment_method_id = $1
		ORDER BY locale ASC`

	rows, err := r.db.Query(ctx, q, id)
	if err != nil {
		return nil, fmt.Errorf("payment: findTranslations: %w", err)
	}
	defer rows.Close()

	var translations []PaymentMethodTranslation
	for rows.Next() {
		var t PaymentMethodTranslation
		if err := rows.Scan(&t.PaymentMethodID, &t.Locale, &t.Name, &t.Description); err != nil {
			return nil, fmt.Errorf("payment: findTranslations scan: %w", err)
		}
		translations = append(translations, t)
	}
	return translations, rows.Err()
}

func (r *postgresMethodRepository) upsertTranslations(ctx context.Context, id uuid.UUID, translations []PaymentMethodTranslation) error {
	const q = `
		INSERT INTO payment_method_translations (payment_method_id, locale, name, description)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (payment_method_id, locale) DO UPDATE
		SET name = EXCLUDED.name, description = EXCLUDED.description`

	for _, t := range translations {
		if _, err := r.db.Exec(ctx, q, id, t.Locale, t.Name, t.Description); err != nil {
			return fmt.Errorf("payment: upsertTranslations: %w", err)
		}
	}
	return nil
}

// encryptConfig encrypts raw config bytes. Returns nil for nil/empty input.
func (r *postgresMethodRepository) encryptConfig(config []byte) ([]byte, error) {
	if len(config) == 0 {
		return config, nil
	}
	return crypto.Encrypt(config, r.encryptionKey)
}

// decryptConfig decrypts the Config field of a PaymentMethod in place.
func (r *postgresMethodRepository) decryptConfig(m *PaymentMethod) error {
	if len(m.Config) == 0 {
		return nil
	}
	plaintext, err := crypto.Decrypt(m.Config, r.encryptionKey)
	if err != nil {
		return err
	}
	m.Config = plaintext
	return nil
}

// MigrateEncryption reads all payment methods and encrypts any config that is
// still stored as plaintext. Safe to call multiple times.
func (r *postgresMethodRepository) MigrateEncryption(ctx context.Context) error {
	const q = `SELECT id, config FROM payment_methods WHERE config IS NOT NULL AND length(config) > 0`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return fmt.Errorf("payment: MigrateEncryption query: %w", err)
	}
	defer rows.Close()

	type row struct {
		id     uuid.UUID
		config []byte
	}
	var toMigrate []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.config); err != nil {
			return fmt.Errorf("payment: MigrateEncryption scan: %w", err)
		}
		if !crypto.IsEncrypted(r.config) {
			toMigrate = append(toMigrate, r)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("payment: MigrateEncryption rows: %w", err)
	}

	for _, row := range toMigrate {
		enc, err := crypto.Encrypt(row.config, r.encryptionKey)
		if err != nil {
			return fmt.Errorf("payment: MigrateEncryption encrypt id=%s: %w", row.id, err)
		}
		if _, err := r.db.Exec(ctx, `UPDATE payment_methods SET config = $1 WHERE id = $2`, enc, row.id); err != nil {
			return fmt.Errorf("payment: MigrateEncryption update id=%s: %w", row.id, err)
		}
		r.logger.Info().Str("id", row.id.String()).Msg("migrated payment method config to encrypted")
	}

	if len(toMigrate) > 0 {
		r.logger.Info().Int("count", len(toMigrate)).Msg("payment config encryption migration complete")
	}
	return nil
}

// --- PaymentTransaction repository ---

type postgresTransactionRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewPostgresTransactionRepository creates a new PostgreSQL-backed PaymentTransactionRepository.
func NewPostgresTransactionRepository(db *pgxpool.Pool, logger zerolog.Logger) PaymentTransactionRepository {
	return &postgresTransactionRepository{db: db, logger: logger}
}

func (r *postgresTransactionRepository) Create(ctx context.Context, t *PaymentTransaction) error {
	const q = `
		INSERT INTO payment_transactions
			(id, order_id, payment_method_id, status, currency, amount, provider_reference, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.db.Exec(ctx, q,
		t.ID, t.OrderID, t.PaymentMethodID, t.Status, t.Currency, t.Amount, t.ProviderReference, t.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("payment: transaction Create: %w", err)
	}
	return nil
}

func (r *postgresTransactionRepository) FindByOrderID(ctx context.Context, orderID uuid.UUID) ([]PaymentTransaction, error) {
	const q = `
		SELECT id, order_id, payment_method_id, status, currency, amount, provider_reference, created_at
		FROM payment_transactions
		WHERE order_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, q, orderID)
	if err != nil {
		return nil, fmt.Errorf("payment: FindByOrderID: %w", err)
	}
	defer rows.Close()

	var transactions []PaymentTransaction
	for rows.Next() {
		var t PaymentTransaction
		if err := rows.Scan(
			&t.ID, &t.OrderID, &t.PaymentMethodID, &t.Status, &t.Currency, &t.Amount, &t.ProviderReference, &t.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("payment: FindByOrderID scan: %w", err)
		}
		transactions = append(transactions, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("payment: FindByOrderID rows: %w", err)
	}
	return transactions, nil
}
