package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is an HTTP client for interacting with the link management API
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new API client
func NewClient(baseURL, apiKey string) *Client {
	// Remove trailing slash from base URL
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// buildRequest creates an HTTP request with proper headers
func (c *Client) buildRequest(method, path string, body io.Reader) (*http.Request, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, path)

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	// Only set Authorization header if API key is provided
	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}

	return req, nil
}

// doRequest performs an HTTP request and handles the response
func (c *Client) doRequest(req *http.Request, result interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errorResp struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error != "" {
			return fmt.Errorf("API error (%d): %s", resp.StatusCode, errorResp.Error)
		}
		// If JSON parsing failed, return the raw body
		errorMsg := string(body)
		if errorMsg == "" {
			errorMsg = resp.Status
		}
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, errorMsg)
	}

	// Parse JSON response if result is provided
	if result != nil {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// doJSONRequest performs a JSON request (POST, PUT, PATCH)
func (c *Client) doJSONRequest(method, path string, payload interface{}, result interface{}) error {
	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := c.buildRequest(method, path, body)
	if err != nil {
		return err
	}

	return c.doRequest(req, result)
}

// doGetRequest performs a GET request
func (c *Client) doGetRequest(path string, result interface{}) error {
	req, err := c.buildRequest(http.MethodGet, path, nil)
	if err != nil {
		return err
	}

	return c.doRequest(req, result)
}

// doDeleteRequest performs a DELETE request
func (c *Client) doDeleteRequest(path string) error {
	req, err := c.buildRequest(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}

	return c.doRequest(req, nil)
}
