package sdk

import (
	"context"
	"testing"
)

// mockSearchEngine is a compile-time check for SearchEngine interface.
type mockSearchEngine struct {
	searchFn func(ctx context.Context, req SearchRequest) (*SearchResponse, error)
	indexFn  func(ctx context.Context, entityType string, id string, data map[string]interface{}) error
	removeFn func(ctx context.Context, entityType string, id string) error
}

func (m *mockSearchEngine) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	if m.searchFn != nil {
		return m.searchFn(ctx, req)
	}
	return &SearchResponse{}, nil
}

func (m *mockSearchEngine) Index(ctx context.Context, entityType string, id string, data map[string]interface{}) error {
	if m.indexFn != nil {
		return m.indexFn(ctx, entityType, id, data)
	}
	return nil
}

func (m *mockSearchEngine) Remove(ctx context.Context, entityType string, id string) error {
	if m.removeFn != nil {
		return m.removeFn(ctx, entityType, id)
	}
	return nil
}

// Compile-time interface compliance checks.
var _ SearchEngine = (*mockSearchEngine)(nil)

// mockSearchPlugin is a compile-time check for SearchPlugin interface.
type mockSearchPlugin struct {
	engine SearchEngine
}

func (m *mockSearchPlugin) Name() string                  { return "mock-search" }
func (m *mockSearchPlugin) Version() string               { return "0.1.0" }
func (m *mockSearchPlugin) Description() string           { return "mock search plugin" }
func (m *mockSearchPlugin) Init(_ *AppContext) error       { return nil }
func (m *mockSearchPlugin) Shutdown() error                { return nil }
func (m *mockSearchPlugin) SearchEngine() SearchEngine     { return m.engine }

var _ SearchPlugin = (*mockSearchPlugin)(nil)
var _ Plugin = (*mockSearchPlugin)(nil)

func TestSearchEngine_InterfaceCompliance(t *testing.T) {
	engine := &mockSearchEngine{}
	var _ SearchEngine = engine

	resp, err := engine.Search(context.Background(), SearchRequest{Query: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestSearchPlugin_InterfaceCompliance(t *testing.T) {
	engine := &mockSearchEngine{}
	plugin := &mockSearchPlugin{engine: engine}

	var _ SearchPlugin = plugin
	var _ Plugin = plugin

	if plugin.Name() != "mock-search" {
		t.Errorf("Name() = %q, want %q", plugin.Name(), "mock-search")
	}
	if plugin.SearchEngine() != engine {
		t.Error("SearchEngine() returned wrong engine")
	}
}

func TestSearchRequest_Fields(t *testing.T) {
	req := SearchRequest{
		Query:  "laptop",
		Locale: "de-DE",
		Page:   2,
		Limit:  10,
		Types:  []string{"product", "category"},
	}

	if req.Query != "laptop" {
		t.Errorf("Query = %q, want %q", req.Query, "laptop")
	}
	if req.Locale != "de-DE" {
		t.Errorf("Locale = %q, want %q", req.Locale, "de-DE")
	}
	if req.Page != 2 {
		t.Errorf("Page = %d, want %d", req.Page, 2)
	}
	if req.Limit != 10 {
		t.Errorf("Limit = %d, want %d", req.Limit, 10)
	}
	if len(req.Types) != 2 {
		t.Errorf("Types len = %d, want %d", len(req.Types), 2)
	}
}

func TestSearchResponse_Fields(t *testing.T) {
	resp := SearchResponse{
		Results: []SearchResult{
			{
				ID:          "uuid-1",
				Type:        "product",
				Score:       0.95,
				Title:       "Test Product",
				Description: "A test product",
				Slug:        "test-product",
				Data:        map[string]interface{}{"price": 1999},
			},
		},
		Total: 1,
		Page:  1,
		Limit: 25,
	}

	if len(resp.Results) != 1 {
		t.Fatalf("Results len = %d, want %d", len(resp.Results), 1)
	}
	r := resp.Results[0]
	if r.ID != "uuid-1" {
		t.Errorf("ID = %q, want %q", r.ID, "uuid-1")
	}
	if r.Type != "product" {
		t.Errorf("Type = %q, want %q", r.Type, "product")
	}
	if r.Score != 0.95 {
		t.Errorf("Score = %f, want %f", r.Score, 0.95)
	}
	if r.Slug != "test-product" {
		t.Errorf("Slug = %q, want %q", r.Slug, "test-product")
	}
}
