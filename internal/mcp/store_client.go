package mcp

import (
	"fmt"
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

func validateStorePath(path string) error {
	if !strings.HasPrefix(path, "/api/v1/store/") {
		return fmt.Errorf("access denied: path %q is outside /api/v1/store/", path)
	}
	if strings.Contains(path, "..") {
		return fmt.Errorf("access denied: path traversal not allowed")
	}
	return nil
}
