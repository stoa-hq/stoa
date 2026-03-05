package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// StoaClient is an HTTP client wrapper for communicating with the Stoa API.
type StoaClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewStoaClient(cfg *Config) *StoaClient {
	return &StoaClient{
		baseURL: cfg.APIURL,
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *StoaClient) Get(path string) ([]byte, error) {
	return c.do("GET", path, nil)
}

func (c *StoaClient) Post(path string, body interface{}) ([]byte, error) {
	return c.do("POST", path, body)
}

func (c *StoaClient) Put(path string, body interface{}) ([]byte, error) {
	return c.do("PUT", path, body)
}

func (c *StoaClient) Delete(path string) ([]byte, error) {
	return c.do("DELETE", path, nil)
}

func (c *StoaClient) do(method, path string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshalling request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := c.baseURL + path
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "ApiKey "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(data),
		}
	}

	return data, nil
}
