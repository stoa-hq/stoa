package search

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// PostgresEngine implements full-text search using PostgreSQL.
type PostgresEngine struct {
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

func NewPostgresEngine(pool *pgxpool.Pool, logger zerolog.Logger) *PostgresEngine {
	return &PostgresEngine{pool: pool, logger: logger}
}

func (e *PostgresEngine) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 25
	}
	offset := (req.Page - 1) * req.Limit

	locale := req.Locale
	if locale == "" {
		locale = "de-DE"
	}

	// Determine the PostgreSQL text search configuration based on locale
	tsConfig := localeToTSConfig(locale)

	// Search products using full-text search
	query := fmt.Sprintf(`
		SELECT
			p.id::text,
			'product' AS type,
			ts_rank(to_tsvector('%s', coalesce(pt.name, '') || ' ' || coalesce(pt.description, '')),
				plainto_tsquery('%s', $1)) AS score,
			pt.name AS title,
			COALESCE(LEFT(pt.description, 200), '') AS description,
			pt.slug
		FROM products p
		JOIN product_translations pt ON p.id = pt.product_id AND pt.locale = $2
		WHERE p.active = true
			AND to_tsvector('%s', coalesce(pt.name, '') || ' ' || coalesce(pt.description, ''))
			@@ plainto_tsquery('%s', $1)
		ORDER BY score DESC
		LIMIT $3 OFFSET $4
	`, tsConfig, tsConfig, tsConfig, tsConfig)

	rows, err := e.pool.Query(ctx, query, req.Query, locale, req.Limit, offset)
	if err != nil {
		return nil, fmt.Errorf("searching products: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ID, &r.Type, &r.Score, &r.Title, &r.Description, &r.Slug); err != nil {
			return nil, fmt.Errorf("scanning search result: %w", err)
		}
		results = append(results, r)
	}

	// Count total results
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM products p
		JOIN product_translations pt ON p.id = pt.product_id AND pt.locale = $2
		WHERE p.active = true
			AND to_tsvector('%s', coalesce(pt.name, '') || ' ' || coalesce(pt.description, ''))
			@@ plainto_tsquery('%s', $1)
	`, tsConfig, tsConfig)

	var total int
	if err := e.pool.QueryRow(ctx, countQuery, req.Query, locale).Scan(&total); err != nil {
		return nil, fmt.Errorf("counting search results: %w", err)
	}

	return &SearchResponse{
		Results: results,
		Total:   total,
		Page:    req.Page,
		Limit:   req.Limit,
	}, nil
}

// Index is a no-op for PostgreSQL since data is already in the database.
func (e *PostgresEngine) Index(ctx context.Context, entityType string, id string, data map[string]interface{}) error {
	return nil
}

// Remove is a no-op for PostgreSQL since data is managed via the database.
func (e *PostgresEngine) Remove(ctx context.Context, entityType string, id string) error {
	return nil
}

func localeToTSConfig(locale string) string {
	switch locale {
	case "de-DE", "de":
		return "german"
	case "en-US", "en-GB", "en":
		return "english"
	case "fr-FR", "fr":
		return "french"
	case "es-ES", "es":
		return "spanish"
	case "it-IT", "it":
		return "italian"
	case "nl-NL", "nl":
		return "dutch"
	default:
		return "simple"
	}
}
