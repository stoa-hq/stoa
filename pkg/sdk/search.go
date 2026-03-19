package sdk

import "context"

// SearchEngine mirrors internal/search.Engine for external plugins.
// Types are structurally identical — Go's internal/ restriction prevents
// direct import, so these are duplicated in the public SDK.
type SearchEngine interface {
	Search(ctx context.Context, req SearchRequest) (*SearchResponse, error)
	Index(ctx context.Context, entityType string, id string, data map[string]interface{}) error
	Remove(ctx context.Context, entityType string, id string) error
}

// SearchRequest contains search parameters.
type SearchRequest struct {
	Query  string
	Locale string
	Page   int
	Limit  int
	Types  []string
}

// SearchResponse contains search results with metadata.
type SearchResponse struct {
	Results []SearchResult
	Total   int
	Page    int
	Limit   int
}

// SearchResult represents a single search result.
type SearchResult struct {
	ID          string
	Type        string
	Score       float64
	Title       string
	Description string
	Slug        string
	Data        map[string]interface{}
}

// SearchPlugin is an optional interface for plugins that provide a search engine.
// When a SearchPlugin is registered, its engine replaces the default PostgreSQL
// full-text search.
type SearchPlugin interface {
	Plugin
	SearchEngine() SearchEngine
}
