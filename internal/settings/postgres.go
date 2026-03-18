package settings

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type postgresRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewPostgresRepository creates a new PostgreSQL-backed settings repository.
func NewPostgresRepository(db *pgxpool.Pool, logger zerolog.Logger) Repository {
	return &postgresRepository{db: db, logger: logger}
}

func (r *postgresRepository) Get(ctx context.Context) (*StoreSettings, error) {
	const q = `
		SELECT store_name, store_description, logo_url, favicon_url, contact_email,
		       currency, country, timezone, copyright_text, maintenance_mode, created_at, updated_at
		FROM store_settings
		WHERE singleton = TRUE`

	s := &StoreSettings{}
	err := r.db.QueryRow(ctx, q).Scan(
		&s.StoreName, &s.StoreDescription, &s.LogoURL, &s.FaviconURL, &s.ContactEmail,
		&s.Currency, &s.Country, &s.Timezone, &s.CopyrightText, &s.MaintenanceMode, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("settings: Get: %w", err)
	}
	return s, nil
}

func (r *postgresRepository) Upsert(ctx context.Context, s *StoreSettings) (*StoreSettings, error) {
	const q = `
		INSERT INTO store_settings (singleton, store_name, store_description, logo_url, favicon_url,
		            contact_email, currency, country, timezone, copyright_text, maintenance_mode, updated_at)
		VALUES (TRUE, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		ON CONFLICT (singleton) DO UPDATE SET
			store_name        = EXCLUDED.store_name,
			store_description = EXCLUDED.store_description,
			logo_url          = EXCLUDED.logo_url,
			favicon_url       = EXCLUDED.favicon_url,
			contact_email     = EXCLUDED.contact_email,
			currency          = EXCLUDED.currency,
			country           = EXCLUDED.country,
			timezone          = EXCLUDED.timezone,
			copyright_text    = EXCLUDED.copyright_text,
			maintenance_mode  = EXCLUDED.maintenance_mode,
			updated_at        = NOW()
		RETURNING store_name, store_description, logo_url, favicon_url, contact_email,
		          currency, country, timezone, copyright_text, maintenance_mode, created_at, updated_at`

	result := &StoreSettings{}
	err := r.db.QueryRow(ctx, q,
		s.StoreName, s.StoreDescription, s.LogoURL, s.FaviconURL, s.ContactEmail,
		s.Currency, s.Country, s.Timezone, s.CopyrightText, s.MaintenanceMode,
	).Scan(
		&result.StoreName, &result.StoreDescription, &result.LogoURL, &result.FaviconURL, &result.ContactEmail,
		&result.Currency, &result.Country, &result.Timezone, &result.CopyrightText, &result.MaintenanceMode,
		&result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("settings: Upsert: %w", err)
	}
	return result, nil
}
