package search

import "context"

// SearchResult represents a single search result.
type SearchResult struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // "product", "category", etc.
	Score       float64                `json:"score"`
	Title       string                 `json:"title"`
	Description string                 `json:"description,omitempty"`
	Slug        string                 `json:"slug,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

// SearchRequest contains search parameters.
type SearchRequest struct {
	Query  string
	Locale string
	Page   int
	Limit  int
	Types  []string // filter by entity type
}

// SearchResponse contains search results with metadata.
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"`
	Page    int            `json:"page"`
	Limit   int            `json:"limit"`
}

// Engine defines the interface for search implementations.
type Engine interface {
	Search(ctx context.Context, req SearchRequest) (*SearchResponse, error)
	Index(ctx context.Context, entityType string, id string, data map[string]interface{}) error
	Remove(ctx context.Context, entityType string, id string) error
}
