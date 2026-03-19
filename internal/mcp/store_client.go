package mcp

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

// StoreScopedClient wraps a StoaClient and restricts API access to
// /api/v1/store/* paths only. This prevents plugins from reaching
// admin endpoints through the MCP StoreAPIClient interface.
type StoreScopedClient struct {
	inner *StoaClient
}

// NewStoreScopedClient creates a store-scoped wrapper around the given client.
func NewStoreScopedClient(client *StoaClient) *StoreScopedClient {
	return &StoreScopedClient{inner: client}
}

func (c *StoreScopedClient) Get(path string) ([]byte, error) {
	if err := validateStorePath(path); err != nil {
		return nil, err
	}
	return c.inner.Get(path)
}

func (c *StoreScopedClient) Post(path string, body interface{}) ([]byte, error) {
	if err := validateStorePath(path); err != nil {
		return nil, err
	}
	return c.inner.Post(path, body)
}

func validateStorePath(rawPath string) error {
	// Decode percent-encoded characters to catch double-encoding bypasses
	// (e.g. %2e%2e → .., %2f → /)
	decoded, err := url.PathUnescape(rawPath)
	if err != nil {
		return fmt.Errorf("access denied: invalid path encoding")
	}

	// Normalize the decoded path to resolve traversal sequences
	cleaned := path.Clean(decoded)

	if !strings.HasPrefix(cleaned, "/api/v1/store/") {
		return fmt.Errorf("access denied: path %q is outside /api/v1/store/", rawPath)
	}

	// Defense-in-depth: reject any remaining traversal attempts
	if strings.Contains(cleaned, "..") {
		return fmt.Errorf("access denied: path traversal not allowed")
	}

	return nil
}
