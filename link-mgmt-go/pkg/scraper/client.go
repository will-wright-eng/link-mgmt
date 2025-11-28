package scraper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ScraperService struct {
	baseURL string
	client  *http.Client
}

func NewScraperService(baseURL string) *ScraperService {
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}

	return &ScraperService{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

type ScrapeRequest struct {
	URL     string `json:"url"`
	Timeout int    `json:"timeout,omitempty"`
}

type ScrapeResponse struct {
	Success     bool   `json:"success"`
	URL         string `json:"url"`
	Title       string `json:"title"`
	Text        string `json:"text"`
	ExtractedAt string `json:"extracted_at"`
	Error       string `json:"error,omitempty"`
}

// CheckHealth verifies the service is available
func (s *ScraperService) CheckHealth() error {
	resp, err := s.client.Get(s.baseURL + "/health")
	if err != nil {
		return fmt.Errorf("service not available: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("service unhealthy: status %d", resp.StatusCode)
	}

	return nil
}

// Scrape scrapes a single URL
func (s *ScraperService) Scrape(url string, timeout int) (*ScrapeResponse, error) {
	reqBody := ScrapeRequest{
		URL:     url,
		Timeout: timeout,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := s.client.Post(
		s.baseURL+"/scrape",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to call scraper service: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scraper service error (status %d): %s", resp.StatusCode, string(body))
	}

	var result ScrapeResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
