package search

import (
	"context"
	"errors"
	"testing"

	"github.com/stoa-hq/stoa/pkg/sdk"
)

type mockSDKEngine struct {
	searchFn func(ctx context.Context, req sdk.SearchRequest) (*sdk.SearchResponse, error)
	indexFn  func(ctx context.Context, entityType string, id string, data map[string]interface{}) error
	removeFn func(ctx context.Context, entityType string, id string) error
}

func (m *mockSDKEngine) Search(ctx context.Context, req sdk.SearchRequest) (*sdk.SearchResponse, error) {
	if m.searchFn != nil {
		return m.searchFn(ctx, req)
	}
	return &sdk.SearchResponse{}, nil
}

func (m *mockSDKEngine) Index(ctx context.Context, entityType string, id string, data map[string]interface{}) error {
	if m.indexFn != nil {
		return m.indexFn(ctx, entityType, id, data)
	}
	return nil
}

func (m *mockSDKEngine) Remove(ctx context.Context, entityType string, id string) error {
	if m.removeFn != nil {
		return m.removeFn(ctx, entityType, id)
	}
	return nil
}

// Compile-time check: SDKEngineAdapter implements Engine.
var _ Engine = (*SDKEngineAdapter)(nil)

func TestSDKEngineAdapter_Search(t *testing.T) {
	sdkEngine := &mockSDKEngine{
		searchFn: func(_ context.Context, req sdk.SearchRequest) (*sdk.SearchResponse, error) {
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
			if len(req.Types) != 1 || req.Types[0] != "product" {
				t.Errorf("Types = %v, want [product]", req.Types)
			}

			return &sdk.SearchResponse{
				Results: []sdk.SearchResult{
					{
						ID:          "prod-1",
						Type:        "product",
						Score:       0.95,
						Title:       "Gaming Laptop",
						Description: "High-end gaming laptop",
						Slug:        "gaming-laptop",
						Data:        map[string]interface{}{"price": 199900},
					},
				},
				Total: 42,
				Page:  2,
				Limit: 10,
			}, nil
		},
	}

	adapter := NewSDKEngineAdapter(sdkEngine)
	resp, err := adapter.Search(context.Background(), SearchRequest{
		Query:  "laptop",
		Locale: "de-DE",
		Page:   2,
		Limit:  10,
		Types:  []string{"product"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 42 {
		t.Errorf("Total = %d, want %d", resp.Total, 42)
	}
	if resp.Page != 2 {
		t.Errorf("Page = %d, want %d", resp.Page, 2)
	}
	if resp.Limit != 10 {
		t.Errorf("Limit = %d, want %d", resp.Limit, 10)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("Results len = %d, want 1", len(resp.Results))
	}

	r := resp.Results[0]
	if r.ID != "prod-1" {
		t.Errorf("ID = %q, want %q", r.ID, "prod-1")
	}
	if r.Type != "product" {
		t.Errorf("Type = %q, want %q", r.Type, "product")
	}
	if r.Score != 0.95 {
		t.Errorf("Score = %f, want %f", r.Score, 0.95)
	}
	if r.Title != "Gaming Laptop" {
		t.Errorf("Title = %q, want %q", r.Title, "Gaming Laptop")
	}
	if r.Description != "High-end gaming laptop" {
		t.Errorf("Description = %q, want %q", r.Description, "High-end gaming laptop")
	}
	if r.Slug != "gaming-laptop" {
		t.Errorf("Slug = %q, want %q", r.Slug, "gaming-laptop")
	}
	if r.Data["price"] != 199900 {
		t.Errorf("Data[price] = %v, want 199900", r.Data["price"])
	}
}

func TestSDKEngineAdapter_Search_Error(t *testing.T) {
	sdkEngine := &mockSDKEngine{
		searchFn: func(_ context.Context, _ sdk.SearchRequest) (*sdk.SearchResponse, error) {
			return nil, errors.New("connection refused")
		},
	}

	adapter := NewSDKEngineAdapter(sdkEngine)
	_, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "connection refused" {
		t.Errorf("error = %q, want %q", err.Error(), "connection refused")
	}
}

func TestSDKEngineAdapter_Index(t *testing.T) {
	var gotType, gotID string
	var gotData map[string]interface{}

	sdkEngine := &mockSDKEngine{
		indexFn: func(_ context.Context, entityType string, id string, data map[string]interface{}) error {
			gotType = entityType
			gotID = id
			gotData = data
			return nil
		},
	}

	adapter := NewSDKEngineAdapter(sdkEngine)
	data := map[string]interface{}{"name": "Test Product"}
	err := adapter.Index(context.Background(), "product", "uuid-1", data)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotType != "product" {
		t.Errorf("entityType = %q, want %q", gotType, "product")
	}
	if gotID != "uuid-1" {
		t.Errorf("id = %q, want %q", gotID, "uuid-1")
	}
	if gotData["name"] != "Test Product" {
		t.Errorf("data[name] = %v, want %q", gotData["name"], "Test Product")
	}
}

func TestSDKEngineAdapter_Index_Error(t *testing.T) {
	sdkEngine := &mockSDKEngine{
		indexFn: func(_ context.Context, _ string, _ string, _ map[string]interface{}) error {
			return errors.New("index failed")
		},
	}

	adapter := NewSDKEngineAdapter(sdkEngine)
	err := adapter.Index(context.Background(), "product", "uuid-1", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSDKEngineAdapter_Remove(t *testing.T) {
	var gotType, gotID string

	sdkEngine := &mockSDKEngine{
		removeFn: func(_ context.Context, entityType string, id string) error {
			gotType = entityType
			gotID = id
			return nil
		},
	}

	adapter := NewSDKEngineAdapter(sdkEngine)
	err := adapter.Remove(context.Background(), "product", "uuid-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotType != "product" {
		t.Errorf("entityType = %q, want %q", gotType, "product")
	}
	if gotID != "uuid-1" {
		t.Errorf("id = %q, want %q", gotID, "uuid-1")
	}
}

func TestSDKEngineAdapter_Remove_Error(t *testing.T) {
	sdkEngine := &mockSDKEngine{
		removeFn: func(_ context.Context, _ string, _ string) error {
			return errors.New("remove failed")
		},
	}

	adapter := NewSDKEngineAdapter(sdkEngine)
	err := adapter.Remove(context.Background(), "product", "uuid-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSDKEngineAdapter_Search_EmptyResults(t *testing.T) {
	sdkEngine := &mockSDKEngine{
		searchFn: func(_ context.Context, _ sdk.SearchRequest) (*sdk.SearchResponse, error) {
			return &sdk.SearchResponse{
				Results: nil,
				Total:   0,
				Page:    1,
				Limit:   25,
			}, nil
		},
	}

	adapter := NewSDKEngineAdapter(sdkEngine)
	resp, err := adapter.Search(context.Background(), SearchRequest{Query: "nonexistent"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Results) != 0 {
		t.Errorf("Results len = %d, want 0", len(resp.Results))
	}
	if resp.Total != 0 {
		t.Errorf("Total = %d, want 0", resp.Total)
	}
}
