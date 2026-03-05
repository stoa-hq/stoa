package mcp

import (
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// APIError represents an error response from the Stoa API.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Body)
}

// apiErrorDetail represents an error in the Stoa error envelope.
type apiErrorDetail struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
	Field  string `json:"field,omitempty"`
}

// ErrorResult converts any error into an MCP tool error result.
func ErrorResult(err error) *mcp.CallToolResult {
	if apiErr, ok := err.(*APIError); ok {
		return mcp.NewToolResultError(formatAPIError(apiErr))
	}
	return mcp.NewToolResultError(err.Error())
}

func formatAPIError(e *APIError) string {
	var envelope struct {
		Errors []apiErrorDetail `json:"errors"`
	}
	if err := json.Unmarshal([]byte(e.Body), &envelope); err == nil && len(envelope.Errors) > 0 {
		msg := fmt.Sprintf("API Error %d:", e.StatusCode)
		for _, detail := range envelope.Errors {
			if detail.Field != "" {
				msg += fmt.Sprintf("\n- %s (%s): %s", detail.Code, detail.Field, detail.Detail)
			} else {
				msg += fmt.Sprintf("\n- %s: %s", detail.Code, detail.Detail)
			}
		}
		return msg
	}
	return fmt.Sprintf("API Error %d: %s", e.StatusCode, e.Body)
}
