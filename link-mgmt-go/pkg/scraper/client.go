package scraper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ScraperService provides methods to interact with the scraper HTTP service
type ScraperService struct {
	baseURL string
	client  *http.Client
}

// NewScraperService creates a new scraper service client
func NewScraperService(baseURL string) *ScraperService {
	return &ScraperService{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// CheckHealth verifies the service is available
func (s *ScraperService) CheckHealth() error {
	return s.CheckHealthWithContext(context.Background())
}

// CheckHealthWithContext verifies the service is available with context support
func (s *ScraperService) CheckHealthWithContext(ctx context.Context) error {
	return s.CheckHealthWithProgress(ctx, nil)
}

// CheckHealthWithProgress verifies the service is available with context and progress support
func (s *ScraperService) CheckHealthWithProgress(ctx context.Context, onProgress ProgressCallback) error {
	if onProgress != nil {
		onProgress(StageHealthCheck, "Checking scraper service...")
	}

	// Use /scraper/health endpoint (via nginx) or /health as fallback
	healthURL := s.baseURL + "/scraper/health"
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return newNetworkError(fmt.Errorf("failed to create request: %w", err))
	}

	resp, err := s.client.Do(req)
	if err != nil {
		// Check for context cancellation
		if ctx.Err() == context.Canceled {
			return newCancelledError(err)
		}
		if ctx.Err() == context.DeadlineExceeded {
			return newTimeoutError(err)
		}

		// Fallback to /health for backward compatibility
		healthURL = s.baseURL + "/health"
		req, err = http.NewRequestWithContext(ctx, "GET", healthURL, nil)
		if err != nil {
			return newNetworkError(fmt.Errorf("failed to create request: %w", err))
		}
		resp, err = s.client.Do(req)
		if err != nil {
			return newServiceUnavailableError(err)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return newServiceUnavailableError(fmt.Errorf("service unhealthy: status %d", resp.StatusCode))
	}

	if onProgress != nil {
		onProgress(StageComplete, "Service is healthy")
	}

	return nil
}

// Scrape scrapes a single URL (backward compatibility wrapper)
func (s *ScraperService) Scrape(url string, timeout int) (*ScrapeResponse, error) {
	return s.ScrapeWithContext(context.Background(), url, timeout)
}

// ScrapeWithContext scrapes a single URL with context support for cancellation
func (s *ScraperService) ScrapeWithContext(ctx context.Context, url string, timeout int) (*ScrapeResponse, error) {
	return s.ScrapeWithProgress(ctx, url, timeout, nil)
}

// ScrapeWithProgress scrapes a single URL with context support and progress callbacks
func (s *ScraperService) ScrapeWithProgress(ctx context.Context, url string, timeout int, onProgress ProgressCallback) (*ScrapeResponse, error) {
	// Stage 1: Health check (optional, but good practice)
	if onProgress != nil {
		onProgress(StageHealthCheck, "Checking scraper service...")
	}
	// Note: We don't actually check health here to avoid extra round-trip
	// The health check is implicit in the scrape request itself

	// Stage 2: Prepare and send request
	if onProgress != nil {
		onProgress(StageFetching, "Sending scrape request...")
	}

	reqBody := ScrapeRequest{
		URL:     url,
		Timeout: timeout,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, newInvalidResponseError("failed to marshal request", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/scrape", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, newNetworkError(fmt.Errorf("failed to create request: %w", err))
	}
	req.Header.Set("Content-Type", "application/json")

	// Stage 3: Waiting for response (service is extracting)
	if onProgress != nil {
		onProgress(StageExtracting, "Extracting content from URL...")
	}

	resp, err := s.client.Do(req)
	if err != nil {
		// Check if error is due to context cancellation
		if ctx.Err() == context.Canceled {
			return nil, newCancelledError(err)
		}
		if ctx.Err() == context.DeadlineExceeded {
			return nil, newTimeoutError(err)
		}
		return nil, newNetworkError(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, newNetworkError(fmt.Errorf("failed to read response: %w", err))
	}

	if resp.StatusCode != http.StatusOK {
		// Try to parse error response
		var errorResp struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(body, &errorResp) == nil && errorResp.Error != "" {
			return nil, newExtractionError(errorResp.Error)
		}
		return nil, newInvalidResponseError(
			fmt.Sprintf("scraper service error (status %d)", resp.StatusCode),
			fmt.Errorf("response: %s", string(body)),
		)
	}

	var result ScrapeResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, newInvalidResponseError("failed to decode response", err)
	}

	// If the response indicates failure, return an extraction error
	if !result.Success {
		errorMsg := result.Error
		if errorMsg == "" {
			errorMsg = "Failed to extract content"
		}
		return nil, newExtractionError(errorMsg)
	}

	// Stage 4: Complete
	if onProgress != nil {
		onProgress(StageComplete, "Scraping completed successfully")
	}

	return &result, nil
}
