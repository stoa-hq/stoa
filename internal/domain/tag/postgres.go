package tag

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

// NewPostgresRepository creates a new PostgreSQL-backed TagRepository.
func NewPostgresRepository(db *pgxpool.Pool, logger zerolog.Logger) TagRepository {
	return &postgresRepository{db: db, logger: logger}
}

func (r *postgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*Tag, error) {
	const q = `SELECT id, name, slug FROM tags WHERE id = $1`

	t := &Tag{}
	err := r.db.QueryRow(ctx, q, id).Scan(&t.ID, &t.Name, &t.Slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("tag: FindByID: %w", err)
	}
	return t, nil
}

func (r *postgresRepository) FindAll(ctx context.Context, filter TagFilter) ([]Tag, int, error) {
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

	if filter.Name != "" {
		where += fmt.Sprintf(" AND name ILIKE $%d", idx)
		args = append(args, "%"+filter.Name+"%")
		idx++
	}

	countQ := fmt.Sprintf("SELECT COUNT(*) FROM tags %s", where)
	var total int
	if err := r.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("tag: FindAll count: %w", err)
	}

	args = append(args, filter.Limit, offset)
	dataQ := fmt.Sprintf(`
		SELECT id, name, slug
		FROM tags
		%s
		ORDER BY name ASC
		LIMIT $%d OFFSET $%d`, where, idx, idx+1)

	rows, err := r.db.Query(ctx, dataQ, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("tag: FindAll: %w", err)
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug); err != nil {
			return nil, 0, fmt.Errorf("tag: FindAll scan: %w", err)
		}
		tags = append(tags, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("tag: FindAll rows: %w", err)
	}
	return tags, total, nil
}

func (r *postgresRepository) Create(ctx context.Context, t *Tag) error {
	const q = `INSERT INTO tags (id, name, slug) VALUES ($1, $2, $3)`
	_, err := r.db.Exec(ctx, q, t.ID, t.Name, t.Slug)
	if err != nil {
		return fmt.Errorf("tag: Create: %w", err)
	}
	return nil
}

func (r *postgresRepository) Update(ctx context.Context, t *Tag) error {
	const q = `UPDATE tags SET name = $2, slug = $3 WHERE id = $1`
	ct, err := r.db.Exec(ctx, q, t.ID, t.Name, t.Slug)
	if err != nil {
		return fmt.Errorf("tag: Update: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM tags WHERE id = $1`
	ct, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("tag: Delete: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
