package category

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type flatRow struct {
	Category
	depth int
}

// PostgresRepository implements CategoryRepository on top of pgxpool.
type PostgresRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewPostgresRepository constructs a PostgresRepository.
func NewPostgresRepository(db *pgxpool.Pool, logger zerolog.Logger) *PostgresRepository {
	return &PostgresRepository{db: db, logger: logger}
}

// -------------------------------------------------------------------------
// FindByID
// -------------------------------------------------------------------------

func (r *PostgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*Category, error) {
	const q = `
		SELECT c.id, c.parent_id, c.position, c.active, c.custom_fields,
		       c.created_at, c.updated_at,
		       ct.locale, ct.name, ct.description, ct.slug
		FROM categories c
		LEFT JOIN category_translations ct ON ct.category_id = c.id
		WHERE c.id = $1
		ORDER BY ct.locale`

	rows, err := r.db.Query(ctx, q, id)
	if err != nil {
		return nil, fmt.Errorf("category.FindByID query: %w", err)
	}
	defer rows.Close()

	cats, err := scanCategoriesWithTranslations(rows)
	if err != nil {
		return nil, fmt.Errorf("category.FindByID scan: %w", err)
	}
	if len(cats) == 0 {
		return nil, ErrNotFound
	}
	return &cats[0], nil
}

// -------------------------------------------------------------------------
// FindAll
// -------------------------------------------------------------------------

func (r *PostgresRepository) FindAll(ctx context.Context, filter CategoryFilter) ([]Category, int, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 || limit > 200 {
		limit = 20
	}
	offset := (page - 1) * limit

	// --- count query ---
	var countArgs []interface{}
	countWhere, countArgs := buildWhereClause(filter, countArgs)

	countQ := fmt.Sprintf("SELECT COUNT(*) FROM categories c %s", countWhere)
	var total int
	if err := r.db.QueryRow(ctx, countQ, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("category.FindAll count: %w", err)
	}

	if total == 0 {
		return []Category{}, 0, nil
	}

	// --- list query ---
	var listArgs []interface{}
	listWhere, listArgs := buildWhereClause(filter, listArgs)
	listArgs = append(listArgs, limit, offset)
	limitPos := len(listArgs) - 1
	offsetPos := len(listArgs)

	listQ := fmt.Sprintf(`
		SELECT c.id, c.parent_id, c.position, c.active, c.custom_fields,
		       c.created_at, c.updated_at,
		       ct.locale, ct.name, ct.description, ct.slug
		FROM categories c
		LEFT JOIN category_translations ct ON ct.category_id = c.id
		%s
		ORDER BY c.position ASC, c.created_at ASC, ct.locale
		LIMIT $%d OFFSET $%d`, listWhere, limitPos, offsetPos)

	rows, err := r.db.Query(ctx, listQ, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("category.FindAll query: %w", err)
	}
	defer rows.Close()

	cats, err := scanCategoriesWithTranslations(rows)
	if err != nil {
		return nil, 0, fmt.Errorf("category.FindAll scan: %w", err)
	}
	return cats, total, nil
}

func buildWhereClause(filter CategoryFilter, args []interface{}) (string, []interface{}) {
	var conditions []string

	if filter.ParentID != nil {
		args = append(args, *filter.ParentID)
		conditions = append(conditions, fmt.Sprintf("c.parent_id = $%d", len(args)))
	}
	if filter.Active != nil {
		args = append(args, *filter.Active)
		conditions = append(conditions, fmt.Sprintf("c.active = $%d", len(args)))
	}

	if len(conditions) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(conditions, " AND "), args
}

// -------------------------------------------------------------------------
// FindTree  – recursive CTE
// -------------------------------------------------------------------------

func (r *PostgresRepository) FindTree(ctx context.Context, locale string) ([]Category, error) {
	// The CTE traverses the category hierarchy starting from root nodes
	// (parent_id IS NULL) down through all descendants. The depth and path
	// columns allow the caller to reconstruct the tree after a flat scan.
	const q = `
		WITH RECURSIVE cat_tree AS (
			-- Anchor: root categories
			SELECT c.id, c.parent_id, c.position, c.active, c.custom_fields,
			       c.created_at, c.updated_at,
			       0 AS depth,
			       ARRAY[c.position, 0] AS sort_path
			FROM categories c
			WHERE c.parent_id IS NULL
			  AND c.active = true

			UNION ALL

			-- Recursive member: children
			SELECT c.id, c.parent_id, c.position, c.active, c.custom_fields,
			       c.created_at, c.updated_at,
			       ct2.depth + 1,
			       ct2.sort_path || c.position
			FROM categories c
			INNER JOIN cat_tree ct2 ON ct2.id = c.parent_id
			WHERE c.active = true
		)
		SELECT t.id, t.parent_id, t.position, t.active, t.custom_fields,
		       t.created_at, t.updated_at,
		       COALESCE(tr.locale, ''),
		       COALESCE(tr.name, ''),
		       COALESCE(tr.description, ''),
		       COALESCE(tr.slug, ''),
		       t.depth
		FROM cat_tree t
		LEFT JOIN category_translations tr
		       ON tr.category_id = t.id AND tr.locale = $1
		ORDER BY t.sort_path`

	rows, err := r.db.Query(ctx, q, locale)
	if err != nil {
		return nil, fmt.Errorf("category.FindTree query: %w", err)
	}
	defer rows.Close()

	// flat list preserving depth information
	var flat []flatRow
	for rows.Next() {
		var cat Category
		var tr CategoryTranslation
		var depth int
		var cfRaw []byte

		if err := rows.Scan(
			&cat.ID, &cat.ParentID, &cat.Position, &cat.Active, &cfRaw,
			&cat.CreatedAt, &cat.UpdatedAt,
			&tr.Locale, &tr.Name, &tr.Description, &tr.Slug,
			&depth,
		); err != nil {
			return nil, fmt.Errorf("category.FindTree row scan: %w", err)
		}

		if cfRaw != nil {
			if err := json.Unmarshal(cfRaw, &cat.CustomFields); err != nil {
				r.logger.Warn().Err(err).Stringer("id", cat.ID).Msg("failed to unmarshal category custom_fields")
			}
		}

		if tr.Locale != "" {
			tr.CategoryID = cat.ID
			cat.Translations = []CategoryTranslation{tr}
		}

		flat = append(flat, flatRow{Category: cat, depth: depth})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("category.FindTree rows: %w", err)
	}

	return buildTree(flat), nil
}

// buildTree converts a flat, depth-annotated slice into a nested tree.
func buildTree(flat []flatRow) []Category {
	// Use a stack to track the current ancestry chain.
	// stack[i] points to the parent Category at depth i.
	roots := make([]Category, 0)
	stack := make([]*Category, 0)

	for i := range flat {
		cat := flat[i].Category
		depth := flat[i].depth

		// Trim stack to current depth
		if depth < len(stack) {
			stack = stack[:depth]
		}

		if depth == 0 {
			roots = append(roots, cat)
			stack = append(stack, &roots[len(roots)-1])
		} else {
			parent := stack[depth-1]
			parent.Children = append(parent.Children, cat)
			stack = append(stack, &parent.Children[len(parent.Children)-1])
		}
	}
	return roots
}

// -------------------------------------------------------------------------
// FindBySlug
// -------------------------------------------------------------------------

func (r *PostgresRepository) FindBySlug(ctx context.Context, slug, locale string) (*Category, error) {
	const q = `
		SELECT c.id, c.parent_id, c.position, c.active, c.custom_fields,
		       c.created_at, c.updated_at,
		       ct2.locale, ct2.name, ct2.description, ct2.slug
		FROM categories c
		INNER JOIN category_translations ct ON ct.category_id = c.id
		       AND ct.slug = $1 AND ct.locale = $2
		LEFT JOIN category_translations ct2 ON ct2.category_id = c.id
		ORDER BY ct2.locale`

	rows, err := r.db.Query(ctx, q, slug, locale)
	if err != nil {
		return nil, fmt.Errorf("category.FindBySlug query: %w", err)
	}
	defer rows.Close()

	cats, err := scanCategoriesWithTranslations(rows)
	if err != nil {
		return nil, fmt.Errorf("category.FindBySlug scan: %w", err)
	}
	if len(cats) == 0 {
		return nil, ErrNotFound
	}
	return &cats[0], nil
}

// -------------------------------------------------------------------------
// Create
// -------------------------------------------------------------------------

func (r *PostgresRepository) Create(ctx context.Context, c *Category) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	now := time.Now().UTC()
	c.CreatedAt = now
	c.UpdatedAt = now

	cfJSON, err := json.Marshal(c.CustomFields)
	if err != nil {
		return fmt.Errorf("category.Create marshal custom_fields: %w", err)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("category.Create begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	const insertCat = `
		INSERT INTO categories (id, parent_id, position, active, custom_fields, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	if _, err = tx.Exec(ctx, insertCat,
		c.ID, c.ParentID, c.Position, c.Active, cfJSON, c.CreatedAt, c.UpdatedAt,
	); err != nil {
		return fmt.Errorf("category.Create insert category: %w", err)
	}

	if err = upsertTranslations(ctx, tx, c.ID, c.Translations); err != nil {
		return fmt.Errorf("category.Create upsert translations: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("category.Create commit: %w", err)
	}
	return nil
}

// -------------------------------------------------------------------------
// Update
// -------------------------------------------------------------------------

func (r *PostgresRepository) Update(ctx context.Context, c *Category) error {
	c.UpdatedAt = time.Now().UTC()

	cfJSON, err := json.Marshal(c.CustomFields)
	if err != nil {
		return fmt.Errorf("category.Update marshal custom_fields: %w", err)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("category.Update begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	const updateCat = `
		UPDATE categories
		SET parent_id = $2, position = $3, active = $4,
		    custom_fields = $5, updated_at = $6
		WHERE id = $1`

	tag, err := tx.Exec(ctx, updateCat,
		c.ID, c.ParentID, c.Position, c.Active, cfJSON, c.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("category.Update exec: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	if err = upsertTranslations(ctx, tx, c.ID, c.Translations); err != nil {
		return fmt.Errorf("category.Update upsert translations: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("category.Update commit: %w", err)
	}
	return nil
}

// -------------------------------------------------------------------------
// Delete
// -------------------------------------------------------------------------

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM categories WHERE id = $1`
	tag, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("category.Delete exec: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// -------------------------------------------------------------------------
// helpers
// -------------------------------------------------------------------------

// scanCategoriesWithTranslations reads rows that join categories with
// category_translations and collapses duplicate category rows that arise
// from multiple translations per category.
func scanCategoriesWithTranslations(rows pgx.Rows) ([]Category, error) {
	// index maps category ID -> position in result slice
	index := make(map[uuid.UUID]int)
	var cats []Category

	for rows.Next() {
		var (
			id       uuid.UUID
			parentID *uuid.UUID
			position int
			active   bool
			cfRaw    []byte
			created  time.Time
			updated  time.Time
			locale   *string
			name     *string
			desc     *string
			slug     *string
		)

		if err := rows.Scan(
			&id, &parentID, &position, &active, &cfRaw,
			&created, &updated,
			&locale, &name, &desc, &slug,
		); err != nil {
			return nil, err
		}

		pos, seen := index[id]
		if !seen {
			cat := Category{
				ID:        id,
				ParentID:  parentID,
				Position:  position,
				Active:    active,
				CreatedAt: created,
				UpdatedAt: updated,
			}
			if cfRaw != nil {
				if err := json.Unmarshal(cfRaw, &cat.CustomFields); err != nil {
					cat.CustomFields = nil
				}
			}
			cats = append(cats, cat)
			pos = len(cats) - 1
			index[id] = pos
		}

		if locale != nil && *locale != "" {
			tr := CategoryTranslation{
				CategoryID: id,
				Locale:     deref(locale),
				Name:       deref(name),
				Slug:       deref(slug),
			}
			if desc != nil {
				tr.Description = *desc
			}
			cats[pos].Translations = append(cats[pos].Translations, tr)
		}
	}
	return cats, rows.Err()
}

// upsertTranslations inserts or updates category_translations rows inside tx.
func upsertTranslations(ctx context.Context, tx pgx.Tx, catID uuid.UUID, translations []CategoryTranslation) error {
	if len(translations) == 0 {
		return nil
	}
	const q = `
		INSERT INTO category_translations (category_id, locale, name, description, slug)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (category_id, locale) DO UPDATE
		SET name = EXCLUDED.name,
		    description = EXCLUDED.description,
		    slug = EXCLUDED.slug`

	for _, tr := range translations {
		if _, err := tx.Exec(ctx, q, catID, tr.Locale, tr.Name, tr.Description, tr.Slug); err != nil {
			return err
		}
	}
	return nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ErrNotFound is returned when a category cannot be located.
var ErrNotFound = errors.New("category not found")
