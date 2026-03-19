package search

import (
	"context"

	"github.com/stoa-hq/stoa/pkg/sdk"
)

// SDKEngineAdapter wraps an sdk.SearchEngine to satisfy the internal search.Engine interface.
// Field-by-field mapping between structurally identical types.
type SDKEngineAdapter struct {
	engine sdk.SearchEngine
}

// NewSDKEngineAdapter creates an adapter that delegates to the given SDK search engine.
func NewSDKEngineAdapter(engine sdk.SearchEngine) *SDKEngineAdapter {
	return &SDKEngineAdapter{engine: engine}
}

func (a *SDKEngineAdapter) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	sdkReq := sdk.SearchRequest{
		Query:  req.Query,
		Locale: req.Locale,
		Page:   req.Page,
		Limit:  req.Limit,
		Types:  req.Types,
	}

	sdkResp, err := a.engine.Search(ctx, sdkReq)
	if err != nil {
		return nil, err
	}

	results := make([]SearchResult, len(sdkResp.Results))
	for i, r := range sdkResp.Results {
		results[i] = SearchResult{
			ID:          r.ID,
			Type:        r.Type,
			Score:       r.Score,
			Title:       r.Title,
			Description: r.Description,
			Slug:        r.Slug,
			Data:        r.Data,
		}
	}

	return &SearchResponse{
		Results: results,
		Total:   sdkResp.Total,
		Page:    sdkResp.Page,
		Limit:   sdkResp.Limit,
	}, nil
}

func (a *SDKEngineAdapter) Index(ctx context.Context, entityType string, id string, data map[string]interface{}) error {
	return a.engine.Index(ctx, entityType, id, data)
}

func (a *SDKEngineAdapter) Remove(ctx context.Context, entityType string, id string) error {
	return a.engine.Remove(ctx, entityType, id)
}
