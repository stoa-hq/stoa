package product

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
)

// postgresRepository is the PostgreSQL-backed implementation of ProductRepository.
type postgresRepository struct {
	db *pgxpool.Pool
}

// NewPostgresRepository constructs a ProductRepository backed by PostgreSQL.
func NewPostgresRepository(db *pgxpool.Pool) ProductRepository {
	return &postgresRepository{db: db}
}

// --------------------------------------------------------------------------
// FindByID
// --------------------------------------------------------------------------

func (r *postgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*Product, error) {
	const query = `
		SELECT
			p.id, p.sku, p.active, p.price_net, p.price_gross, p.currency,
			p.tax_rule_id, p.stock, p.weight, p.custom_fields, p.metadata,
			p.created_at, p.updated_at
		FROM products p
		WHERE p.id = $1`

	p := &Product{}
	var customFieldsRaw, metadataRaw []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.SKU, &p.Active, &p.PriceNet, &p.PriceGross, &p.Currency,
		&p.TaxRuleID, &p.Stock, &p.Weight, &customFieldsRaw, &metadataRaw,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("product FindByID: %w", err)
	}

	if err := unmarshalJSON(customFieldsRaw, &p.CustomFields); err != nil {
		return nil, fmt.Errorf("product FindByID custom_fields: %w", err)
	}
	if err := unmarshalJSON(metadataRaw, &p.Metadata); err != nil {
		return nil, fmt.Errorf("product FindByID metadata: %w", err)
	}

	if err := r.loadRelations(ctx, p); err != nil {
		return nil, err
	}

	return p, nil
}

// --------------------------------------------------------------------------
// FindBySKU
// --------------------------------------------------------------------------

func (r *postgresRepository) FindBySKU(ctx context.Context, sku string) (*Product, error) {
	const query = `
		SELECT
			p.id, p.sku, p.active, p.price_net, p.price_gross, p.currency,
			p.tax_rule_id, p.stock, p.weight, p.custom_fields, p.metadata,
			p.created_at, p.updated_at
		FROM products p
		WHERE p.sku = $1`

	p := &Product{}
	var customFieldsRaw, metadataRaw []byte

	err := r.db.QueryRow(ctx, query, sku).Scan(
		&p.ID, &p.SKU, &p.Active, &p.PriceNet, &p.PriceGross, &p.Currency,
		&p.TaxRuleID, &p.Stock, &p.Weight, &customFieldsRaw, &metadataRaw,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("product FindBySKU: %w", err)
	}

	if err := unmarshalJSON(customFieldsRaw, &p.CustomFields); err != nil {
		return nil, fmt.Errorf("product FindBySKU custom_fields: %w", err)
	}
	if err := unmarshalJSON(metadataRaw, &p.Metadata); err != nil {
		return nil, fmt.Errorf("product FindBySKU metadata: %w", err)
	}

	if err := r.loadRelations(ctx, p); err != nil {
		return nil, err
	}

	return p, nil
}

// --------------------------------------------------------------------------
// FindAll
// --------------------------------------------------------------------------

func (r *postgresRepository) FindAll(ctx context.Context, filter ProductFilter) ([]Product, int, error) {
	// Validate and apply defaults.
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 20
	}

	// Allowlisted sort columns to prevent SQL injection.
	allowedSortCols := map[string]string{
		"created_at":  "p.created_at",
		"updated_at":  "p.updated_at",
		"price_net":   "p.price_net",
		"price_gross": "p.price_gross",
		"stock":       "p.stock",
		"sku":         "p.sku",
	}
	sortCol := "p.created_at"
	if col, ok := allowedSortCols[strings.ToLower(filter.Sort)]; ok {
		sortCol = col
	}

	sortDir := "DESC"
	if strings.EqualFold(filter.Order, "asc") {
		sortDir = "ASC"
	}

	// Build WHERE clauses dynamically.
	var (
		conditions []string
		args       []interface{}
		argIdx     = 1
	)

	if filter.Active != nil {
		conditions = append(conditions, fmt.Sprintf("p.active = $%d", argIdx))
		args = append(args, *filter.Active)
		argIdx++
	}

	if filter.CategoryID != nil {
		conditions = append(conditions, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM product_categories pc WHERE pc.product_id = p.id AND pc.category_id = $%d)", argIdx,
		))
		args = append(args, *filter.CategoryID)
		argIdx++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM product_translations pt WHERE pt.product_id = p.id AND to_tsvector('german', coalesce(pt.name, '') || ' ' || coalesce(pt.description, '')) @@ plainto_tsquery('german', $%d))",
			argIdx,
		))
		args = append(args, filter.Search)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total matching rows.
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM products p %s`, where)
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("product FindAll count: %w", err)
	}

	// Fetch page.
	offset := (filter.Page - 1) * filter.Limit
	dataQuery := fmt.Sprintf(`
		SELECT
			p.id, p.sku, p.active, p.price_net, p.price_gross, p.currency,
			p.tax_rule_id, p.stock, p.weight, p.custom_fields, p.metadata,
			p.created_at, p.updated_at
		FROM products p
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		where, sortCol, sortDir, argIdx, argIdx+1,
	)
	args = append(args, filter.Limit, offset)

	rows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("product FindAll query: %w", err)
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		var customFieldsRaw, metadataRaw []byte

		if err := rows.Scan(
			&p.ID, &p.SKU, &p.Active, &p.PriceNet, &p.PriceGross, &p.Currency,
			&p.TaxRuleID, &p.Stock, &p.Weight, &customFieldsRaw, &metadataRaw,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("product FindAll scan: %w", err)
		}

		if err := unmarshalJSON(customFieldsRaw, &p.CustomFields); err != nil {
			return nil, 0, err
		}
		if err := unmarshalJSON(metadataRaw, &p.Metadata); err != nil {
			return nil, 0, err
		}

		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("product FindAll rows: %w", err)
	}

	// Bulk-load translations and media for all returned products.
	if len(products) > 0 {
		if err := r.loadTranslationsForMany(ctx, products); err != nil {
			return nil, 0, err
		}
		if err := r.loadMediaForMany(ctx, products); err != nil {
			return nil, 0, err
		}
		if err := r.loadHasVariantsForMany(ctx, products); err != nil {
			return nil, 0, err
		}
	}

	return products, total, nil
}

// --------------------------------------------------------------------------
// FindBySlug
// --------------------------------------------------------------------------

func (r *postgresRepository) FindBySlug(ctx context.Context, slug, locale string) (*Product, error) {
	const query = `
		SELECT
			p.id, p.sku, p.active, p.price_net, p.price_gross, p.currency,
			p.tax_rule_id, p.stock, p.weight, p.custom_fields, p.metadata,
			p.created_at, p.updated_at
		FROM products p
		INNER JOIN product_translations pt ON pt.product_id = p.id
		WHERE pt.slug = $1
		LIMIT 1`

	p := &Product{}
	var customFieldsRaw, metadataRaw []byte

	err := r.db.QueryRow(ctx, query, slug).Scan(
		&p.ID, &p.SKU, &p.Active, &p.PriceNet, &p.PriceGross, &p.Currency,
		&p.TaxRuleID, &p.Stock, &p.Weight, &customFieldsRaw, &metadataRaw,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("product FindBySlug: %w", err)
	}

	if err := unmarshalJSON(customFieldsRaw, &p.CustomFields); err != nil {
		return nil, fmt.Errorf("product FindBySlug custom_fields: %w", err)
	}
	if err := unmarshalJSON(metadataRaw, &p.Metadata); err != nil {
		return nil, fmt.Errorf("product FindBySlug metadata: %w", err)
	}

	if err := r.loadRelations(ctx, p); err != nil {
		return nil, err
	}

	return p, nil
}

// --------------------------------------------------------------------------
// Create
// --------------------------------------------------------------------------

func (r *postgresRepository) Create(ctx context.Context, p *Product) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now

	customFieldsJSON, err := marshalJSON(p.CustomFields)
	if err != nil {
		return fmt.Errorf("product Create marshal custom_fields: %w", err)
	}
	metadataJSON, err := marshalJSON(p.Metadata)
	if err != nil {
		return fmt.Errorf("product Create marshal metadata: %w", err)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("product Create begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	const insertProduct = `
		INSERT INTO products
			(id, sku, active, price_net, price_gross, currency, tax_rule_id, stock, weight, custom_fields, metadata, created_at, updated_at)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	_, err = tx.Exec(ctx, insertProduct,
		p.ID, p.SKU, p.Active, p.PriceNet, p.PriceGross, p.Currency,
		p.TaxRuleID, p.Stock, p.Weight, customFieldsJSON, metadataJSON,
		p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("product Create insert: %w", err)
	}

	if err := insertTranslations(ctx, tx, p.ID, p.Translations); err != nil {
		return err
	}

	if err := insertCategories(ctx, tx, p.ID, p.Categories); err != nil {
		return err
	}

	if err := insertTags(ctx, tx, p.ID, p.Tags); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("product Create commit: %w", err)
	}

	return nil
}

// --------------------------------------------------------------------------
// Update
// --------------------------------------------------------------------------

func (r *postgresRepository) Update(ctx context.Context, p *Product) error {
	p.UpdatedAt = time.Now().UTC()

	customFieldsJSON, err := marshalJSON(p.CustomFields)
	if err != nil {
		return fmt.Errorf("product Update marshal custom_fields: %w", err)
	}
	metadataJSON, err := marshalJSON(p.Metadata)
	if err != nil {
		return fmt.Errorf("product Update marshal metadata: %w", err)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("product Update begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	const updateProduct = `
		UPDATE products SET
			sku          = $2,
			active       = $3,
			price_net    = $4,
			price_gross  = $5,
			currency     = $6,
			tax_rule_id  = $7,
			stock        = $8,
			weight       = $9,
			custom_fields = $10,
			metadata     = $11,
			updated_at   = $12
		WHERE id = $1`

	tag, err := tx.Exec(ctx, updateProduct,
		p.ID, p.SKU, p.Active, p.PriceNet, p.PriceGross, p.Currency,
		p.TaxRuleID, p.Stock, p.Weight, customFieldsJSON, metadataJSON,
		p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("product Update exec: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	// Replace translations: delete existing then re-insert.
	if _, err := tx.Exec(ctx, `DELETE FROM product_translations WHERE product_id = $1`, p.ID); err != nil {
		return fmt.Errorf("product Update delete translations: %w", err)
	}
	if err := insertTranslations(ctx, tx, p.ID, p.Translations); err != nil {
		return err
	}

	// Replace media: delete existing then re-insert.
	if _, err := tx.Exec(ctx, `DELETE FROM product_media WHERE product_id = $1`, p.ID); err != nil {
		return fmt.Errorf("product Update delete media: %w", err)
	}
	for _, m := range p.Media {
		if _, err := tx.Exec(ctx,
			`INSERT INTO product_media (product_id, media_id, position) VALUES ($1, $2, $3)`,
			p.ID, m.MediaID, m.Position,
		); err != nil {
			return fmt.Errorf("product Update insert media: %w", err)
		}
	}

	// Replace categories: delete existing then re-insert.
	if _, err := tx.Exec(ctx, `DELETE FROM product_categories WHERE product_id = $1`, p.ID); err != nil {
		return fmt.Errorf("product Update delete categories: %w", err)
	}
	if err := insertCategories(ctx, tx, p.ID, p.Categories); err != nil {
		return err
	}

	// Replace tags: delete existing then re-insert.
	if _, err := tx.Exec(ctx, `DELETE FROM product_tags WHERE product_id = $1`, p.ID); err != nil {
		return fmt.Errorf("product Update delete tags: %w", err)
	}
	if err := insertTags(ctx, tx, p.ID, p.Tags); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("product Update commit: %w", err)
	}

	return nil
}

// --------------------------------------------------------------------------
// Delete
// --------------------------------------------------------------------------

func (r *postgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("product Delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --------------------------------------------------------------------------
// Helpers – relation loading
// --------------------------------------------------------------------------

// loadRelations populates all sub-resources on a single product.
func (r *postgresRepository) loadRelations(ctx context.Context, p *Product) error {
	if err := r.loadTranslations(ctx, p); err != nil {
		return err
	}
	if err := r.loadCategories(ctx, p); err != nil {
		return err
	}
	if err := r.loadTags(ctx, p); err != nil {
		return err
	}
	if err := r.loadMedia(ctx, p); err != nil {
		return err
	}
	if err := r.loadVariants(ctx, p); err != nil {
		return err
	}
	return nil
}

func (r *postgresRepository) loadTranslations(ctx context.Context, p *Product) error {
	const query = `
		SELECT product_id, locale, name, description, slug, meta_title, meta_description
		FROM product_translations
		WHERE product_id = $1`

	rows, err := r.db.Query(ctx, query, p.ID)
	if err != nil {
		return fmt.Errorf("loadTranslations query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var t ProductTranslation
		if err := rows.Scan(&t.ProductID, &t.Locale, &t.Name, &t.Description, &t.Slug, &t.MetaTitle, &t.MetaDescription); err != nil {
			return fmt.Errorf("loadTranslations scan: %w", err)
		}
		p.Translations = append(p.Translations, t)
	}
	return rows.Err()
}

// loadTranslationsForMany bulk-loads translations for a slice of products.
func (r *postgresRepository) loadTranslationsForMany(ctx context.Context, products []Product) error {
	ids := make([]uuid.UUID, len(products))
	idx := make(map[uuid.UUID]int, len(products))
	for i := range products {
		ids[i] = products[i].ID
		idx[products[i].ID] = i
	}

	const query = `
		SELECT product_id, locale, name, description, slug, meta_title, meta_description
		FROM product_translations
		WHERE product_id = ANY($1)`

	rows, err := r.db.Query(ctx, query, ids)
	if err != nil {
		return fmt.Errorf("loadTranslationsForMany query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var t ProductTranslation
		if err := rows.Scan(&t.ProductID, &t.Locale, &t.Name, &t.Description, &t.Slug, &t.MetaTitle, &t.MetaDescription); err != nil {
			return fmt.Errorf("loadTranslationsForMany scan: %w", err)
		}
		if i, ok := idx[t.ProductID]; ok {
			products[i].Translations = append(products[i].Translations, t)
		}
	}
	return rows.Err()
}

// loadMediaForMany bulk-loads media for a slice of products.
func (r *postgresRepository) loadMediaForMany(ctx context.Context, products []Product) error {
	ids := make([]uuid.UUID, len(products))
	idx := make(map[uuid.UUID]int, len(products))
	for i := range products {
		ids[i] = products[i].ID
		idx[products[i].ID] = i
	}

	const query = `
		SELECT pm.product_id, pm.media_id, pm.position, COALESCE(m.storage_path, '')
		FROM product_media pm
		LEFT JOIN media m ON m.id = pm.media_id
		WHERE pm.product_id = ANY($1)
		ORDER BY pm.product_id, pm.position`

	rows, err := r.db.Query(ctx, query, ids)
	if err != nil {
		return fmt.Errorf("loadMediaForMany query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var productID uuid.UUID
		var m ProductMedia
		if err := rows.Scan(&productID, &m.MediaID, &m.Position, &m.StoragePath); err != nil {
			return fmt.Errorf("loadMediaForMany scan: %w", err)
		}
		if i, ok := idx[productID]; ok {
			products[i].Media = append(products[i].Media, m)
		}
	}
	return rows.Err()
}

func (r *postgresRepository) loadHasVariantsForMany(ctx context.Context, products []Product) error {
	ids := make([]uuid.UUID, len(products))
	idx := make(map[uuid.UUID]int, len(products))
	for i := range products {
		ids[i] = products[i].ID
		idx[products[i].ID] = i
	}

	const query = `
		SELECT DISTINCT product_id
		FROM product_variants
		WHERE product_id = ANY($1) AND active = true`

	rows, err := r.db.Query(ctx, query, ids)
	if err != nil {
		return fmt.Errorf("loadHasVariantsForMany query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var productID uuid.UUID
		if err := rows.Scan(&productID); err != nil {
			return fmt.Errorf("loadHasVariantsForMany scan: %w", err)
		}
		if i, ok := idx[productID]; ok {
			products[i].HasVariants = true
		}
	}
	return rows.Err()
}

func (r *postgresRepository) loadCategories(ctx context.Context, p *Product) error {
	const query = `SELECT category_id FROM product_categories WHERE product_id = $1`

	rows, err := r.db.Query(ctx, query, p.ID)
	if err != nil {
		return fmt.Errorf("loadCategories query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("loadCategories scan: %w", err)
		}
		p.Categories = append(p.Categories, id)
	}
	return rows.Err()
}

func (r *postgresRepository) loadTags(ctx context.Context, p *Product) error {
	const query = `SELECT tag_id FROM product_tags WHERE product_id = $1`

	rows, err := r.db.Query(ctx, query, p.ID)
	if err != nil {
		return fmt.Errorf("loadTags query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("loadTags scan: %w", err)
		}
		p.Tags = append(p.Tags, id)
	}
	return rows.Err()
}

func (r *postgresRepository) loadMedia(ctx context.Context, p *Product) error {
	const query = `
		SELECT pm.media_id, pm.position, COALESCE(m.storage_path, '')
		FROM product_media pm
		LEFT JOIN media m ON m.id = pm.media_id
		WHERE pm.product_id = $1
		ORDER BY pm.position`

	rows, err := r.db.Query(ctx, query, p.ID)
	if err != nil {
		return fmt.Errorf("loadMedia query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var m ProductMedia
		if err := rows.Scan(&m.MediaID, &m.Position, &m.StoragePath); err != nil {
			return fmt.Errorf("loadMedia scan: %w", err)
		}
		p.Media = append(p.Media, m)
	}
	return rows.Err()
}

func (r *postgresRepository) loadVariants(ctx context.Context, p *Product) error {
	const variantQuery = `
		SELECT id, product_id, sku, price_net, price_gross, stock, active, custom_fields, created_at, updated_at
		FROM product_variants
		WHERE product_id = $1`

	rows, err := r.db.Query(ctx, variantQuery, p.ID)
	if err != nil {
		return fmt.Errorf("loadVariants query: %w", err)
	}
	defer rows.Close()

	var variants []ProductVariant
	for rows.Next() {
		var v ProductVariant
		var customFieldsRaw []byte
		if err := rows.Scan(
			&v.ID, &v.ProductID, &v.SKU, &v.PriceNet, &v.PriceGross,
			&v.Stock, &v.Active, &customFieldsRaw,
			&v.CreatedAt, &v.UpdatedAt,
		); err != nil {
			return fmt.Errorf("loadVariants scan: %w", err)
		}
		if err := unmarshalJSON(customFieldsRaw, &v.CustomFields); err != nil {
			return err
		}
		variants = append(variants, v)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("loadVariants rows: %w", err)
	}

	// Load options for each variant.
	for i := range variants {
		opts, err := r.loadVariantOptions(ctx, variants[i].ID)
		if err != nil {
			return err
		}
		variants[i].Options = opts
	}

	p.Variants = variants
	return nil
}

func (r *postgresRepository) loadVariantOptions(ctx context.Context, variantID uuid.UUID) ([]PropertyOption, error) {
	const query = `
		SELECT
			po.id, po.group_id, po.color_hex, po.position, po.created_at, po.updated_at
		FROM property_options po
		INNER JOIN product_variant_options pvo ON pvo.option_id = po.id
		WHERE pvo.variant_id = $1
		ORDER BY po.position`

	rows, err := r.db.Query(ctx, query, variantID)
	if err != nil {
		return nil, fmt.Errorf("loadVariantOptions query: %w", err)
	}
	defer rows.Close()

	var options []PropertyOption
	for rows.Next() {
		var o PropertyOption
		if err := rows.Scan(&o.ID, &o.GroupID, &o.ColorHex, &o.Position, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, fmt.Errorf("loadVariantOptions scan: %w", err)
		}

		// Load translations for each option.
		translations, err := r.loadOptionTranslations(ctx, o.ID)
		if err != nil {
			return nil, err
		}
		o.Translations = translations
		options = append(options, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("loadVariantOptions rows: %w", err)
	}
	return options, nil
}

func (r *postgresRepository) loadOptionTranslations(ctx context.Context, optionID uuid.UUID) ([]PropertyOptionTranslation, error) {
	const query = `SELECT option_id, locale, name FROM property_option_translations WHERE option_id = $1`

	rows, err := r.db.Query(ctx, query, optionID)
	if err != nil {
		return nil, fmt.Errorf("loadOptionTranslations query: %w", err)
	}
	defer rows.Close()

	var translations []PropertyOptionTranslation
	for rows.Next() {
		var t PropertyOptionTranslation
		if err := rows.Scan(&t.OptionID, &t.Locale, &t.Name); err != nil {
			return nil, fmt.Errorf("loadOptionTranslations scan: %w", err)
		}
		translations = append(translations, t)
	}
	return translations, rows.Err()
}

// --------------------------------------------------------------------------
// Helpers – DML
// --------------------------------------------------------------------------

// pgTx is the interface satisfied by both pgx.Tx and pgxpool.Pool for Exec.
type pgTx interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (interface{ RowsAffected() int64 }, error)
}

func insertTranslations(ctx context.Context, tx pgx.Tx, productID uuid.UUID, translations []ProductTranslation) error {
	for _, t := range translations {
		_, err := tx.Exec(ctx, `
			INSERT INTO product_translations
				(product_id, locale, name, description, slug, meta_title, meta_description)
			VALUES
				($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (product_id, locale) DO UPDATE SET
				name             = EXCLUDED.name,
				description      = EXCLUDED.description,
				slug             = EXCLUDED.slug,
				meta_title       = EXCLUDED.meta_title,
				meta_description = EXCLUDED.meta_description`,
			productID, t.Locale, t.Name, t.Description, t.Slug, t.MetaTitle, t.MetaDescription,
		)
		if err != nil {
			return fmt.Errorf("insertTranslations (locale=%s): %w", t.Locale, err)
		}
	}
	return nil
}

func insertCategories(ctx context.Context, tx pgx.Tx, productID uuid.UUID, categoryIDs []uuid.UUID) error {
	for _, cid := range categoryIDs {
		if _, err := tx.Exec(ctx,
			`INSERT INTO product_categories (product_id, category_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			productID, cid,
		); err != nil {
			return fmt.Errorf("insertCategories (category_id=%s): %w", cid, err)
		}
	}
	return nil
}

func insertTags(ctx context.Context, tx pgx.Tx, productID uuid.UUID, tagIDs []uuid.UUID) error {
	for _, tid := range tagIDs {
		if _, err := tx.Exec(ctx,
			`INSERT INTO product_tags (product_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			productID, tid,
		); err != nil {
			return fmt.Errorf("insertTags (tag_id=%s): %w", tid, err)
		}
	}
	return nil
}

// --------------------------------------------------------------------------
// Helpers – JSON serialisation
// --------------------------------------------------------------------------

func marshalJSON(v map[string]interface{}) ([]byte, error) {
	if v == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(v)
}

func unmarshalJSON(data []byte, target *map[string]interface{}) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	return json.Unmarshal(data, target)
}

// --------------------------------------------------------------------------
// CreateVariant
// --------------------------------------------------------------------------

func (r *postgresRepository) CreateVariant(ctx context.Context, v *ProductVariant) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	now := time.Now().UTC()
	v.CreatedAt = now
	v.UpdatedAt = now

	customFieldsJSON, err := marshalJSON(v.CustomFields)
	if err != nil {
		return fmt.Errorf("CreateVariant marshal custom_fields: %w", err)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("CreateVariant begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx, `
		INSERT INTO product_variants
			(id, product_id, sku, price_net, price_gross, stock, active, custom_fields, created_at, updated_at)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		v.ID, v.ProductID, v.SKU, v.PriceNet, v.PriceGross,
		v.Stock, v.Active, customFieldsJSON, v.CreatedAt, v.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("CreateVariant insert: %w", err)
	}

	for _, o := range v.Options {
		if _, err := tx.Exec(ctx,
			`INSERT INTO product_variant_options (variant_id, option_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			v.ID, o.ID,
		); err != nil {
			return fmt.Errorf("CreateVariant insert option pivot: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("CreateVariant commit: %w", err)
	}
	return nil
}

// --------------------------------------------------------------------------
// FindVariantByID
// --------------------------------------------------------------------------

func (r *postgresRepository) FindVariantByID(ctx context.Context, id uuid.UUID) (*ProductVariant, error) {
	const query = `
		SELECT id, product_id, sku, price_net, price_gross, stock, active, custom_fields, created_at, updated_at
		FROM product_variants
		WHERE id = $1`

	v := &ProductVariant{}
	var customFieldsRaw []byte
	err := r.db.QueryRow(ctx, query, id).Scan(
		&v.ID, &v.ProductID, &v.SKU, &v.PriceNet, &v.PriceGross,
		&v.Stock, &v.Active, &customFieldsRaw, &v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("FindVariantByID: %w", err)
	}
	if err := unmarshalJSON(customFieldsRaw, &v.CustomFields); err != nil {
		return nil, err
	}
	opts, err := r.loadVariantOptions(ctx, v.ID)
	if err != nil {
		return nil, err
	}
	v.Options = opts
	return v, nil
}

// --------------------------------------------------------------------------
// UpdateVariant
// --------------------------------------------------------------------------

func (r *postgresRepository) UpdateVariant(ctx context.Context, v *ProductVariant) error {
	v.UpdatedAt = time.Now().UTC()

	customFieldsJSON, err := marshalJSON(v.CustomFields)
	if err != nil {
		return fmt.Errorf("UpdateVariant marshal custom_fields: %w", err)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("UpdateVariant begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	tag, err := tx.Exec(ctx, `
		UPDATE product_variants SET
			sku         = $2,
			price_net   = $3,
			price_gross = $4,
			stock       = $5,
			active      = $6,
			custom_fields = $7,
			updated_at  = $8
		WHERE id = $1`,
		v.ID, v.SKU, v.PriceNet, v.PriceGross,
		v.Stock, v.Active, customFieldsJSON, v.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("UpdateVariant exec: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	if _, err := tx.Exec(ctx, `DELETE FROM product_variant_options WHERE variant_id = $1`, v.ID); err != nil {
		return fmt.Errorf("UpdateVariant delete option pivots: %w", err)
	}
	for _, o := range v.Options {
		if _, err := tx.Exec(ctx,
			`INSERT INTO product_variant_options (variant_id, option_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			v.ID, o.ID,
		); err != nil {
			return fmt.Errorf("UpdateVariant insert option pivot: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("UpdateVariant commit: %w", err)
	}
	return nil
}

// --------------------------------------------------------------------------
// DeleteVariant
// --------------------------------------------------------------------------

func (r *postgresRepository) DeleteVariant(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM product_variants WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("DeleteVariant: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --------------------------------------------------------------------------
// FindAllPropertyGroups
// --------------------------------------------------------------------------

func (r *postgresRepository) FindAllPropertyGroups(ctx context.Context) ([]PropertyGroup, error) {
	const query = `SELECT id, position, created_at, updated_at FROM property_groups ORDER BY position`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("FindAllPropertyGroups query: %w", err)
	}
	defer rows.Close()

	var groups []PropertyGroup
	for rows.Next() {
		var g PropertyGroup
		if err := rows.Scan(&g.ID, &g.Position, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, fmt.Errorf("FindAllPropertyGroups scan: %w", err)
		}
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("FindAllPropertyGroups rows: %w", err)
	}

	if len(groups) == 0 {
		return groups, nil
	}

	// Bulk-load translations and options.
	if err := r.loadGroupTranslationsForMany(ctx, groups); err != nil {
		return nil, err
	}
	if err := r.loadOptionsForManyGroups(ctx, groups); err != nil {
		return nil, err
	}

	return groups, nil
}

// --------------------------------------------------------------------------
// FindPropertyGroupByID
// --------------------------------------------------------------------------

func (r *postgresRepository) FindPropertyGroupByID(ctx context.Context, id uuid.UUID) (*PropertyGroup, error) {
	const query = `SELECT id, position, created_at, updated_at FROM property_groups WHERE id = $1`

	g := &PropertyGroup{}
	if err := r.db.QueryRow(ctx, query, id).Scan(&g.ID, &g.Position, &g.CreatedAt, &g.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("FindPropertyGroupByID: %w", err)
	}

	translations, err := r.loadGroupTranslations(ctx, g.ID)
	if err != nil {
		return nil, err
	}
	g.Translations = translations

	opts, err := r.FindOptionsByGroupID(ctx, g.ID)
	if err != nil {
		return nil, err
	}
	g.Options = opts

	return g, nil
}

// --------------------------------------------------------------------------
// CreatePropertyGroup
// --------------------------------------------------------------------------

func (r *postgresRepository) CreatePropertyGroup(ctx context.Context, g *PropertyGroup) error {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	now := time.Now().UTC()
	g.CreatedAt = now
	g.UpdatedAt = now

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("CreatePropertyGroup begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx,
		`INSERT INTO property_groups (id, position, created_at, updated_at) VALUES ($1, $2, $3, $4)`,
		g.ID, g.Position, g.CreatedAt, g.UpdatedAt,
	); err != nil {
		return fmt.Errorf("CreatePropertyGroup insert: %w", err)
	}

	for _, t := range g.Translations {
		if _, err := tx.Exec(ctx,
			`INSERT INTO property_group_translations (property_group_id, locale, name) VALUES ($1, $2, $3)`,
			g.ID, t.Locale, t.Name,
		); err != nil {
			return fmt.Errorf("CreatePropertyGroup insert translation (locale=%s): %w", t.Locale, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("CreatePropertyGroup commit: %w", err)
	}
	return nil
}

// --------------------------------------------------------------------------
// UpdatePropertyGroup
// --------------------------------------------------------------------------

func (r *postgresRepository) UpdatePropertyGroup(ctx context.Context, g *PropertyGroup) error {
	g.UpdatedAt = time.Now().UTC()

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("UpdatePropertyGroup begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	tag, err := tx.Exec(ctx,
		`UPDATE property_groups SET position = $2, updated_at = $3 WHERE id = $1`,
		g.ID, g.Position, g.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("UpdatePropertyGroup exec: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	if _, err := tx.Exec(ctx, `DELETE FROM property_group_translations WHERE property_group_id = $1`, g.ID); err != nil {
		return fmt.Errorf("UpdatePropertyGroup delete translations: %w", err)
	}
	for _, t := range g.Translations {
		if _, err := tx.Exec(ctx,
			`INSERT INTO property_group_translations (property_group_id, locale, name) VALUES ($1, $2, $3)`,
			g.ID, t.Locale, t.Name,
		); err != nil {
			return fmt.Errorf("UpdatePropertyGroup insert translation (locale=%s): %w", t.Locale, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("UpdatePropertyGroup commit: %w", err)
	}
	return nil
}

// --------------------------------------------------------------------------
// DeletePropertyGroup
// --------------------------------------------------------------------------

func (r *postgresRepository) DeletePropertyGroup(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM property_groups WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("DeletePropertyGroup: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --------------------------------------------------------------------------
// FindOptionsByGroupID
// --------------------------------------------------------------------------

func (r *postgresRepository) FindOptionsByGroupID(ctx context.Context, groupID uuid.UUID) ([]PropertyOption, error) {
	const query = `
		SELECT id, group_id, color_hex, position, created_at, updated_at
		FROM property_options
		WHERE group_id = $1
		ORDER BY position`

	rows, err := r.db.Query(ctx, query, groupID)
	if err != nil {
		return nil, fmt.Errorf("FindOptionsByGroupID query: %w", err)
	}
	defer rows.Close()

	var options []PropertyOption
	for rows.Next() {
		var o PropertyOption
		if err := rows.Scan(&o.ID, &o.GroupID, &o.ColorHex, &o.Position, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, fmt.Errorf("FindOptionsByGroupID scan: %w", err)
		}
		options = append(options, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("FindOptionsByGroupID rows: %w", err)
	}

	for i := range options {
		translations, err := r.loadOptionTranslations(ctx, options[i].ID)
		if err != nil {
			return nil, err
		}
		options[i].Translations = translations
	}

	return options, nil
}

// --------------------------------------------------------------------------
// CreatePropertyOption
// --------------------------------------------------------------------------

func (r *postgresRepository) CreatePropertyOption(ctx context.Context, o *PropertyOption) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	now := time.Now().UTC()
	o.CreatedAt = now
	o.UpdatedAt = now

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("CreatePropertyOption begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx,
		`INSERT INTO property_options (id, group_id, color_hex, position, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		o.ID, o.GroupID, o.ColorHex, o.Position, o.CreatedAt, o.UpdatedAt,
	); err != nil {
		return fmt.Errorf("CreatePropertyOption insert: %w", err)
	}

	for _, t := range o.Translations {
		if _, err := tx.Exec(ctx,
			`INSERT INTO property_option_translations (option_id, locale, name) VALUES ($1, $2, $3)`,
			o.ID, t.Locale, t.Name,
		); err != nil {
			return fmt.Errorf("CreatePropertyOption insert translation (locale=%s): %w", t.Locale, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("CreatePropertyOption commit: %w", err)
	}
	return nil
}

// --------------------------------------------------------------------------
// UpdatePropertyOption
// --------------------------------------------------------------------------

func (r *postgresRepository) UpdatePropertyOption(ctx context.Context, o *PropertyOption) error {
	o.UpdatedAt = time.Now().UTC()

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("UpdatePropertyOption begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	tag, err := tx.Exec(ctx,
		`UPDATE property_options SET color_hex = $2, position = $3, updated_at = $4 WHERE id = $1`,
		o.ID, o.ColorHex, o.Position, o.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("UpdatePropertyOption exec: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	if _, err := tx.Exec(ctx, `DELETE FROM property_option_translations WHERE option_id = $1`, o.ID); err != nil {
		return fmt.Errorf("UpdatePropertyOption delete translations: %w", err)
	}
	for _, t := range o.Translations {
		if _, err := tx.Exec(ctx,
			`INSERT INTO property_option_translations (option_id, locale, name) VALUES ($1, $2, $3)`,
			o.ID, t.Locale, t.Name,
		); err != nil {
			return fmt.Errorf("UpdatePropertyOption insert translation (locale=%s): %w", t.Locale, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("UpdatePropertyOption commit: %w", err)
	}
	return nil
}

// --------------------------------------------------------------------------
// DeletePropertyOption
// --------------------------------------------------------------------------

func (r *postgresRepository) DeletePropertyOption(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM property_options WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("DeletePropertyOption: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --------------------------------------------------------------------------
// FindOrCreatePropertyGroup
// --------------------------------------------------------------------------

// FindOrCreatePropertyGroup looks up a property group by locale + name and
// creates one if it does not exist yet. Used during CSV/bulk import.
func (r *postgresRepository) FindOrCreatePropertyGroup(ctx context.Context, locale, name string) (*PropertyGroup, error) {
	const findQuery = `
		SELECT pg.id, pg.position, pg.created_at, pg.updated_at
		FROM property_groups pg
		JOIN property_group_translations pgt ON pgt.property_group_id = pg.id
		WHERE pgt.locale = $1 AND pgt.name = $2
		LIMIT 1`

	g := &PropertyGroup{}
	err := r.db.QueryRow(ctx, findQuery, locale, name).Scan(&g.ID, &g.Position, &g.CreatedAt, &g.UpdatedAt)
	if err == nil {
		g.Translations = []PropertyGroupTranslation{{GroupID: g.ID, Locale: locale, Name: name}}
		return g, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("FindOrCreatePropertyGroup find: %w", err)
	}

	// Not found – create a new group.
	g = &PropertyGroup{
		Translations: []PropertyGroupTranslation{{Locale: locale, Name: name}},
	}
	if err := r.CreatePropertyGroup(ctx, g); err != nil {
		return nil, fmt.Errorf("FindOrCreatePropertyGroup create: %w", err)
	}
	g.Translations[0].GroupID = g.ID
	return g, nil
}

// --------------------------------------------------------------------------
// FindOrCreatePropertyOption
// --------------------------------------------------------------------------

// FindOrCreatePropertyOption looks up a property option by group ID, locale,
// and name. Creates one if it does not exist. Used during CSV/bulk import.
func (r *postgresRepository) FindOrCreatePropertyOption(ctx context.Context, groupID uuid.UUID, locale, name string) (*PropertyOption, error) {
	const findQuery = `
		SELECT po.id, po.group_id, po.color_hex, po.position, po.created_at, po.updated_at
		FROM property_options po
		JOIN property_option_translations pot ON pot.option_id = po.id
		WHERE po.group_id = $1 AND pot.locale = $2 AND pot.name = $3
		LIMIT 1`

	o := &PropertyOption{}
	err := r.db.QueryRow(ctx, findQuery, groupID, locale, name).Scan(
		&o.ID, &o.GroupID, &o.ColorHex, &o.Position, &o.CreatedAt, &o.UpdatedAt,
	)
	if err == nil {
		o.Translations = []PropertyOptionTranslation{{OptionID: o.ID, Locale: locale, Name: name}}
		return o, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("FindOrCreatePropertyOption find: %w", err)
	}

	// Not found – create a new option.
	o = &PropertyOption{
		GroupID:      groupID,
		Translations: []PropertyOptionTranslation{{Locale: locale, Name: name}},
	}
	if err := r.CreatePropertyOption(ctx, o); err != nil {
		return nil, fmt.Errorf("FindOrCreatePropertyOption create: %w", err)
	}
	o.Translations[0].OptionID = o.ID
	return o, nil
}

// --------------------------------------------------------------------------
// Helpers – property group / option loading
// --------------------------------------------------------------------------

func (r *postgresRepository) loadGroupTranslations(ctx context.Context, groupID uuid.UUID) ([]PropertyGroupTranslation, error) {
	const query = `SELECT property_group_id, locale, name FROM property_group_translations WHERE property_group_id = $1`

	rows, err := r.db.Query(ctx, query, groupID)
	if err != nil {
		return nil, fmt.Errorf("loadGroupTranslations query: %w", err)
	}
	defer rows.Close()

	var translations []PropertyGroupTranslation
	for rows.Next() {
		var t PropertyGroupTranslation
		if err := rows.Scan(&t.GroupID, &t.Locale, &t.Name); err != nil {
			return nil, fmt.Errorf("loadGroupTranslations scan: %w", err)
		}
		translations = append(translations, t)
	}
	return translations, rows.Err()
}

// loadGroupTranslationsForMany bulk-loads translations for many groups in-place.
func (r *postgresRepository) loadGroupTranslationsForMany(ctx context.Context, groups []PropertyGroup) error {
	ids := make([]uuid.UUID, len(groups))
	idx := make(map[uuid.UUID]int, len(groups))
	for i := range groups {
		ids[i] = groups[i].ID
		idx[groups[i].ID] = i
	}

	const query = `SELECT property_group_id, locale, name FROM property_group_translations WHERE property_group_id = ANY($1)`
	rows, err := r.db.Query(ctx, query, ids)
	if err != nil {
		return fmt.Errorf("loadGroupTranslationsForMany query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var t PropertyGroupTranslation
		if err := rows.Scan(&t.GroupID, &t.Locale, &t.Name); err != nil {
			return fmt.Errorf("loadGroupTranslationsForMany scan: %w", err)
		}
		if i, ok := idx[t.GroupID]; ok {
			groups[i].Translations = append(groups[i].Translations, t)
		}
	}
	return rows.Err()
}

// loadOptionsForManyGroups bulk-loads options (with translations) for many groups in-place.
func (r *postgresRepository) loadOptionsForManyGroups(ctx context.Context, groups []PropertyGroup) error {
	ids := make([]uuid.UUID, len(groups))
	idx := make(map[uuid.UUID]int, len(groups))
	for i := range groups {
		ids[i] = groups[i].ID
		idx[groups[i].ID] = i
	}

	const query = `
		SELECT id, group_id, color_hex, position, created_at, updated_at
		FROM property_options
		WHERE group_id = ANY($1)
		ORDER BY group_id, position`

	rows, err := r.db.Query(ctx, query, ids)
	if err != nil {
		return fmt.Errorf("loadOptionsForManyGroups query: %w", err)
	}
	defer rows.Close()

	var allOpts []PropertyOption
	optIdx := make(map[uuid.UUID]int) // optID → index in allOpts

	for rows.Next() {
		var o PropertyOption
		if err := rows.Scan(&o.ID, &o.GroupID, &o.ColorHex, &o.Position, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return fmt.Errorf("loadOptionsForManyGroups scan: %w", err)
		}
		optIdx[o.ID] = len(allOpts)
		allOpts = append(allOpts, o)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("loadOptionsForManyGroups rows: %w", err)
	}

	if len(allOpts) == 0 {
		return nil
	}

	// Bulk-load option translations.
	optIDs := make([]uuid.UUID, len(allOpts))
	for i := range allOpts {
		optIDs[i] = allOpts[i].ID
	}
	const tQuery = `SELECT option_id, locale, name FROM property_option_translations WHERE option_id = ANY($1)`
	tRows, err := r.db.Query(ctx, tQuery, optIDs)
	if err != nil {
		return fmt.Errorf("loadOptionsForManyGroups translations query: %w", err)
	}
	defer tRows.Close()

	for tRows.Next() {
		var t PropertyOptionTranslation
		if err := tRows.Scan(&t.OptionID, &t.Locale, &t.Name); err != nil {
			return fmt.Errorf("loadOptionsForManyGroups translations scan: %w", err)
		}
		if i, ok := optIdx[t.OptionID]; ok {
			allOpts[i].Translations = append(allOpts[i].Translations, t)
		}
	}
	if err := tRows.Err(); err != nil {
		return fmt.Errorf("loadOptionsForManyGroups translations rows: %w", err)
	}

	// Assign options back to groups.
	for _, o := range allOpts {
		if gi, ok := idx[o.GroupID]; ok {
			groups[gi].Options = append(groups[gi].Options, o)
		}
	}

	return nil
}

// --------------------------------------------------------------------------
// StockAvailable
// --------------------------------------------------------------------------

// StockAvailable reports whether at least quantity units are available.
// When variantID is non-nil the variant's stock column is queried (and the
// variant must belong to the given product). Otherwise the product-level
// stock is used.
func (r *postgresRepository) StockAvailable(ctx context.Context, productID uuid.UUID, variantID *uuid.UUID, quantity int) (bool, error) {
	var stock int
	var err error

	if variantID != nil {
		err = r.db.QueryRow(ctx,
			`SELECT stock FROM product_variants WHERE id = $1 AND product_id = $2`,
			*variantID, productID,
		).Scan(&stock)
	} else {
		err = r.db.QueryRow(ctx,
			`SELECT stock FROM products WHERE id = $1`,
			productID,
		).Scan(&stock)
	}

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("product StockAvailable: %w", err)
	}

	return stock >= quantity, nil
}

// ErrNotFound is returned when a requested product does not exist.
var ErrNotFound = errors.New("product: not found")
