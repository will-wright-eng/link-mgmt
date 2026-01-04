package utils

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidateURL trims and validates a URL string, returning a normalized value
// or an error if the URL is empty or invalid.
func ValidateURL(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", fmt.Errorf("URL is required")
	}
	if _, err := url.Parse(s); err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	return s, nil
}
