package media

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

// NewPostgresRepository creates a new PostgreSQL-backed MediaRepository.
func NewPostgresRepository(db *pgxpool.Pool, logger zerolog.Logger) MediaRepository {
	return &postgresRepository{db: db, logger: logger}
}

func (r *postgresRepository) Create(ctx context.Context, m *Media) error {
	thumbJSON, err := json.Marshal(m.Thumbnails)
	if err != nil {
		return fmt.Errorf("media: Create marshal thumbnails: %w", err)
	}
	cfJSON, err := json.Marshal(m.CustomFields)
	if err != nil {
		return fmt.Errorf("media: Create marshal custom_fields: %w", err)
	}

	const q = `
		INSERT INTO media (id, filename, mime_type, size, storage_path, alt_text, thumbnails, custom_fields, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err = r.db.Exec(ctx, q,
		m.ID, m.Filename, m.MimeType, m.Size, m.StoragePath, m.AltText, thumbJSON, cfJSON, m.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("media: Create: %w", err)
	}
	return nil
}

func (r *postgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*Media, error) {
	const q = `
		SELECT id, filename, mime_type, size, storage_path, alt_text, thumbnails, custom_fields, created_at
		FROM media
		WHERE id = $1`

	m := &Media{}
	var thumbRaw, cfRaw []byte
	err := r.db.QueryRow(ctx, q, id).Scan(
		&m.ID, &m.Filename, &m.MimeType, &m.Size, &m.StoragePath, &m.AltText, &thumbRaw, &cfRaw, &m.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("media: FindByID: %w", err)
	}
	if len(thumbRaw) > 0 {
		if err := json.Unmarshal(thumbRaw, &m.Thumbnails); err != nil {
			return nil, fmt.Errorf("media: FindByID unmarshal thumbnails: %w", err)
		}
	}
	if len(cfRaw) > 0 {
		if err := json.Unmarshal(cfRaw, &m.CustomFields); err != nil {
			return nil, fmt.Errorf("media: FindByID unmarshal custom_fields: %w", err)
		}
	}
	return m, nil
}

func (r *postgresRepository) FindAll(ctx context.Context, filter MediaFilter) ([]Media, int, error) {
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

	if filter.MimeType != "" {
		where += fmt.Sprintf(" AND mime_type ILIKE $%d", idx)
		args = append(args, filter.MimeType+"%")
		idx++
	}

	countQ := fmt.Sprintf("SELECT COUNT(*) FROM media %s", where)
	var total int
	if err := r.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("media: FindAll count: %w", err)
	}

	args = append(args, filter.Limit, offset)
	dataQ := fmt.Sprintf(`
		SELECT id, filename, mime_type, size, storage_path, alt_text, thumbnails, custom_fields, created_at
		FROM media %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, where, idx, idx+1)

	rows, err := r.db.Query(ctx, dataQ, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("media: FindAll: %w", err)
	}
	defer rows.Close()

	var items []Media
	for rows.Next() {
		var m Media
		var thumbRaw, cfRaw []byte
		if err := rows.Scan(
			&m.ID, &m.Filename, &m.MimeType, &m.Size, &m.StoragePath, &m.AltText, &thumbRaw, &cfRaw, &m.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("media: FindAll scan: %w", err)
		}
		if len(thumbRaw) > 0 {
			if err := json.Unmarshal(thumbRaw, &m.Thumbnails); err != nil {
				return nil, 0, fmt.Errorf("media: FindAll unmarshal thumbnails: %w", err)
			}
		}
		if len(cfRaw) > 0 {
			if err := json.Unmarshal(cfRaw, &m.CustomFields); err != nil {
				return nil, 0, fmt.Errorf("media: FindAll unmarshal custom_fields: %w", err)
			}
		}
		items = append(items, m)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("media: FindAll rows: %w", err)
	}
	return items, total, nil
}

func (r *postgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM media WHERE id = $1`
	ct, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("media: Delete: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
