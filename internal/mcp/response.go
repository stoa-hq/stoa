package mcp

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// APIResponse represents the standard Stoa API response envelope.
type APIResponse struct {
	Data   json.RawMessage `json:"data"`
	Meta   *APIMeta        `json:"meta,omitempty"`
	Errors []apiErrorDetail `json:"errors,omitempty"`
}

// APIMeta contains pagination metadata.
type APIMeta struct {
	Total int `json:"total"`
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Pages int `json:"pages"`
}

// ParseResponse parses raw API bytes into an APIResponse.
func ParseResponse(data []byte) (*APIResponse, error) {
	var resp APIResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing API response: %w", err)
	}
	return &resp, nil
}

// FormatResponse creates a readable MCP text result from API response bytes.
func FormatResponse(data []byte) (*mcp.CallToolResult, error) {
	resp, err := ParseResponse(data)
	if err != nil {
		return nil, err
	}

	if len(resp.Errors) > 0 {
		var parts []string
		for _, e := range resp.Errors {
			parts = append(parts, fmt.Sprintf("%s: %s", e.Code, e.Detail))
		}
		return mcp.NewToolResultError(strings.Join(parts, "\n")), nil
	}

	// Pretty-print the data portion
	var pretty json.RawMessage
	if err := json.Unmarshal(resp.Data, &pretty); err != nil {
		return mcp.NewToolResultText(string(resp.Data)), nil
	}

	out, _ := json.MarshalIndent(pretty, "", "  ")
	text := string(out)

	if resp.Meta != nil && resp.Meta.Total > 0 {
		text += fmt.Sprintf("\n\n---\nPage %d of %d (total: %d)", resp.Meta.Page, resp.Meta.Pages, resp.Meta.Total)
	}

	return mcp.NewToolResultText(text), nil
}

// FormatRaw returns raw bytes as a JSON text result.
func FormatRaw(data []byte) *mcp.CallToolResult {
	var pretty json.RawMessage
	if err := json.Unmarshal(data, &pretty); err != nil {
		return mcp.NewToolResultText(string(data))
	}
	out, _ := json.MarshalIndent(pretty, "", "  ")
	return mcp.NewToolResultText(string(out))
}
